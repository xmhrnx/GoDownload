// Package bt downloads and seeds torrents. It wraps anacrolix/torrent with what
// a download manager needs: per-torrent folders, pause by dropping, per-torrent
// seeding switches and upload caps, file priorities and streaming readers.
package bt

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	g "github.com/anacrolix/generics"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"

	"github.com/godownload/godownload/internal/ratelimit"
)

// Config sets up the torrent client.
type Config struct {
	// DataDir holds piece completion data and saved .torrent files.
	DataDir       string
	Port          int
	DHT           bool
	PortForward   bool
	Encryption    bool
	ExtraTrackers bool
	Upload        *ratelimit.Limiter
	Download      *ratelimit.Limiter
}

// Engine is the torrent client shared by every torrent download.
type Engine struct {
	cfg Config
	cl  *torrent.Client
	pc  storage.PieceCompletion

	mu      sync.Mutex
	handles map[metainfo.Hash]*Handle
}

// NewEngine starts the client and listens for peers.
func NewEngine(cfg Config) (*Engine, error) {
	if err := os.MkdirAll(filepath.Join(cfg.DataDir, "torrents"), 0o755); err != nil {
		return nil, err
	}
	pc, err := storage.NewBoltPieceCompletion(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("opening piece database: %w", err)
	}
	tc := torrent.NewDefaultClientConfig()
	tc.DataDir = cfg.DataDir
	tc.ListenPort = cfg.Port
	tc.NoDHT = !cfg.DHT
	tc.NoDefaultPortForwarding = !cfg.PortForward
	tc.Seed = true
	tc.HeaderObfuscationPolicy.Preferred = cfg.Encryption
	tc.HeaderObfuscationPolicy.RequirePreferred = false
	if cfg.Upload != nil {
		tc.UploadRateLimiter = cfg.Upload.Raw()
	}
	if cfg.Download != nil {
		tc.DownloadRateLimiter = cfg.Download.Raw()
	}
	// More peers than the default helps poorly seeded torrents get going.
	tc.EstablishedConnsPerTorrent = 80
	tc.HalfOpenConnsPerTorrent = 40
	tc.TotalHalfOpenConns = 200

	cl, err := torrent.NewClient(tc)
	if err != nil && cfg.Port != 0 {
		// The port is taken (another client running): let the system pick one.
		tc.ListenPort = 0
		cl, err = torrent.NewClient(tc)
	}
	if err != nil {
		pc.Close()
		return nil, fmt.Errorf("starting torrent client: %w", err)
	}
	return &Engine{cfg: cfg, cl: cl, pc: pc, handles: map[metainfo.Hash]*Handle{}}, nil
}

// Close stops every torrent and the client.
func (e *Engine) Close() error {
	errs := e.cl.Close()
	errs = append(errs, e.pc.Close())
	return errors.Join(errs...)
}

// Port is the port peers connect to.
func (e *Engine) Port() int { return e.cl.LocalPort() }

// Open adds a torrent that saves into dir, without downloading anything yet.
// A torrent opened before (its .torrent file saved) starts without asking
// peers for metadata.
func (e *Engine) Open(src Source, dir string) (*Handle, error) {
	spec, err := e.spec(src)
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if h, ok := e.handles[spec.InfoHash]; ok {
		return h, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	store := newThrottledStorage(storage.NewFileOpts(storage.NewFileClientOpts{
		ClientBaseDir:   dir,
		PieceCompletion: sharedCompletion{e.pc},
		UsePartFiles:    g.Some(false),
		// Memory-mapped files count every touched page of the download as
		// the app's memory; plain reads and writes keep it small.
		DisableMmap: true,
	}))
	spec.Storage = store
	spec.DisallowDataDownload = true
	t, _, err := e.cl.AddTorrentSpec(spec)
	if err != nil {
		return nil, err
	}
	if e.cfg.ExtraTrackers {
		go func() {
			<-t.GotInfo()
			if info := t.Info(); info != nil && (info.Private == nil || !*info.Private) {
				t.AddTrackers(publicTrackers)
			}
		}()
	}
	h := newHandle(e, t, dir, store)
	h.peerAddrs = spec.PeerAddrs
	e.handles[spec.InfoHash] = h
	go e.saveMetainfo(t)
	return h, nil
}

func (e *Engine) forget(h *Handle) {
	e.mu.Lock()
	delete(e.handles, h.t.InfoHash())
	e.mu.Unlock()
}

func (e *Engine) torrentFile(ih metainfo.Hash) string {
	return filepath.Join(e.cfg.DataDir, "torrents", ih.HexString()+".torrent")
}

// saveMetainfo keeps the .torrent once known, so a restart doesn't wait on
// peers for metadata.
func (e *Engine) saveMetainfo(t *torrent.Torrent) {
	select {
	case <-t.GotInfo():
		e.writeMetainfo(t)
	case <-t.Closed():
	}
}

func (e *Engine) writeMetainfo(t *torrent.Torrent) {
	path := e.torrentFile(t.InfoHash())
	if _, err := os.Stat(path); err == nil {
		return
	}
	mi := t.Metainfo()
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	if err := mi.Write(f); err != nil {
		os.Remove(path)
	}
}

// sharedCompletion lets every torrent's storage use the one piece database
// without closing it when a torrent is dropped.
//
// It must pass on Checkpoint and Persistent: storage only batches piece
// records and file syncs (about once a second) when the completion store
// has them. Hidden behind a plain wrapper, every finished piece costs a
// database commit and an fsync of the whole file, which stalls the client
// and makes the speed jump up and down.
type sharedCompletion struct{ storage.PieceCompletion }

func (sharedCompletion) Close() error { return nil }

func (s sharedCompletion) Checkpoint(keys []metainfo.PieceKey) error {
	if c, ok := s.PieceCompletion.(storage.PieceCompletionCheckpointer); ok {
		return c.Checkpoint(keys)
	}
	return nil
}

func (s sharedCompletion) Persistent() bool {
	if p, ok := s.PieceCompletion.(storage.PieceCompletionPersistenter); ok {
		return p.Persistent()
	}
	return true
}

var _ storage.PieceCompletionCheckpointer = sharedCompletion{}

// WaitInfo waits for the torrent's metadata (instant for .torrent files).
func (h *Handle) WaitInfo(ctx context.Context) error {
	select {
	case <-h.t.GotInfo():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
