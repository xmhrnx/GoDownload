package bt

import (
	"context"
	"sync"

	"github.com/anacrolix/torrent"

	"github.com/godownload/godownload/internal/model"
)

// Handle is one open torrent.
type Handle struct {
	e   *Engine
	t   *torrent.Torrent
	dir string

	// peerAddrs are peers the magnet link named (x.pe), dialed again
	// whenever downloading restarts.
	peerAddrs []string

	mu         sync.Mutex
	prios      []model.FilePriority
	peerMeters map[string]*peerMeter
	seed       bool
	cap        *uploadCap
}

func newHandle(e *Engine, t *torrent.Torrent, dir string) *Handle {
	h := &Handle{e: e, t: t, dir: dir}
	h.cap = newUploadCap(h)
	return h
}

// InfoHash is the torrent's v1 info hash in hex.
func (h *Handle) InfoHash() string { return h.t.InfoHash().HexString() }

// HasInfo reports whether the metadata is known.
func (h *Handle) HasInfo() bool { return h.t.Info() != nil }

// Name is the torrent's name, or its hash until metadata arrives.
func (h *Handle) Name() string { return h.t.Name() }

// Private reports whether the torrent forbids DHT and peer exchange.
func (h *Handle) Private() bool {
	info := h.t.Info()
	return info != nil && info.Private != nil && *info.Private
}

// Files lists the files with their progress and priority.
func (h *Handle) Files() []model.TorrentFile {
	if !h.HasInfo() {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	files := h.t.Files()
	out := make([]model.TorrentFile, len(files))
	for i, f := range files {
		p := model.PriorityNormal
		if i < len(h.prios) {
			p = h.prios[i]
		}
		out[i] = model.TorrentFile{Path: f.DisplayPath(), Size: f.Length(), Done: f.BytesCompleted(), Priority: p}
	}
	return out
}

// Start begins downloading the wanted files. Nil prios wants everything.
func (h *Handle) Start(prios []model.FilePriority, seed bool) {
	h.SetPriorities(prios)
	h.SetSeed(seed)
	h.t.AllowDataDownload()
	h.rediscover()
}

// rediscover offers every peer seen before to the torrent again. A finished
// torrent drops connections to other seeders, so when a check later finds
// damaged pieces those peers must be dialed again to fetch them.
func (h *Handle) rediscover() {
	peers := h.t.KnownSwarm()
	for _, a := range h.peerAddrs {
		peers = append(peers, torrent.PeerInfo{Addr: torrent.StringAddr(a), Source: torrent.PeerSourceDirect})
	}
	h.t.AddPeers(peers)
}

// SetPriorities sets each file's priority, in metainfo order.
func (h *Handle) SetPriorities(prios []model.FilePriority) {
	h.mu.Lock()
	h.prios = append([]model.FilePriority(nil), prios...)
	h.mu.Unlock()
	go func() {
		<-h.t.GotInfo()
		h.mu.Lock()
		defer h.mu.Unlock()
		for i, f := range h.t.Files() {
			p := model.PriorityNormal
			if i < len(h.prios) {
				p = h.prios[i]
			}
			switch p {
			case model.PrioritySkip:
				f.SetPriority(torrent.PiecePriorityNone)
			case model.PriorityHigh:
				f.SetPriority(torrent.PiecePriorityHigh)
			default:
				f.SetPriority(torrent.PiecePriorityNormal)
			}
		}
	}()
}

// SetSeed allows or forbids uploading to other peers.
func (h *Handle) SetSeed(on bool) {
	h.mu.Lock()
	h.seed = on
	h.mu.Unlock()
	h.cap.apply()
}

// SetUploadLimit caps this torrent's upload rate; 0 removes the cap.
func (h *Handle) SetUploadLimit(bytesPerSec int64) { h.cap.set(bytesPerSec) }

func (h *Handle) seeding() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.seed
}

// Close stops the torrent. Its data and piece records stay, so opening it
// again resumes without rechecking.
func (h *Handle) Close() {
	h.cap.stop()
	h.e.forget(h)
	h.t.Drop()
}

// Progress is the bytes done and wanted across the files not skipped.
func (h *Handle) Progress() (done, wanted int64) {
	if !h.HasInfo() {
		return 0, -1
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, f := range h.t.Files() {
		if i < len(h.prios) && h.prios[i] == model.PrioritySkip {
			continue
		}
		done += f.BytesCompleted()
		wanted += f.Length()
	}
	return done, wanted
}

// Verify rehashes every piece; damaged pieces are downloaded again.
func (h *Handle) Verify(ctx context.Context) error {
	if !h.HasInfo() {
		return nil
	}
	return h.t.VerifyDataContext(ctx)
}

// Reader opens a streaming reader on file i. Reads wait for pieces, and the
// pieces under and just ahead of the read position are fetched first.
func (h *Handle) Reader(i int, readahead int64) (torrent.Reader, *torrent.File) {
	f := h.t.Files()[i]
	r := f.NewReader()
	r.SetResponsive()
	r.SetReadahead(readahead)
	return r, f
}

// PrioritizeEnds fetches the first and last pieces of file i first: players
// read the header at the start and often an index at the end before playing.
func (h *Handle) PrioritizeEnds(i int) {
	files := h.t.Files()
	if i < 0 || i >= len(files) {
		return
	}
	f := files[i]
	begin, end := f.BeginPieceIndex(), f.EndPieceIndex()
	for p := begin; p < min(begin+2, end); p++ {
		h.t.Piece(p).SetPriority(torrent.PiecePriorityHigh)
	}
	if end-1 > begin+1 {
		h.t.Piece(end - 1).SetPriority(torrent.PiecePriorityHigh)
	}
}
