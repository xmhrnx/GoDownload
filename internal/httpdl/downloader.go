// Package httpdl downloads a file over HTTP(S) with several connections at
// once, resumes after interruptions, repairs damaged blocks, and serves the
// parts already on disk to a player while the rest arrives.
package httpdl

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/godownload/godownload/internal/fileutil"
	"github.com/godownload/godownload/internal/ratelimit"
)

// Config describes one download.
type Config struct {
	Request Request
	// Path is where the finished file goes. While downloading, data is written
	// to Path + fileutil.PartSuffix.
	Path string
	// StatePath holds the resume data (which blocks are done, with checksums).
	StatePath   string
	Connections int
	Retries     int
	Client      *http.Client
	Limiters    []*ratelimit.Limiter
	// StreamWindow is how far ahead of a player's position to fetch first.
	StreamWindow int64
	// Log receives problems that don't stop the download.
	Log func(format string, args ...any)
}

// Downloader runs one HTTP download. Create it with New, call Run, and cancel
// Run's context to pause.
type Downloader struct {
	cfg Config

	mu       sync.Mutex
	info     Info
	blocks   *BlockMap // nil when the server can't send ranges
	segs     map[*segment]struct{}
	priority int64
	// changed is closed and replaced whenever a block lands, waking streams.
	changed      chan struct{}
	streamWindow int64
	maxConns     int
	live         int
	runCtx       context.Context
	cancelRun    context.CancelCauseFunc
	lastErr      error
	idle         *sync.Cond
	singleDone   int64 // bytes written in single-connection mode

	fileMu sync.RWMutex
	file   *os.File
	final  bool // the file has its final name

	received atomic.Int64
	running  atomic.Bool
}

// ErrChanged means the file on the server is no longer the one partly downloaded.
var ErrChanged = errors.New("the file on the server changed")

// New prepares a download; nothing happens until Run.
func New(cfg Config) *Downloader {
	if cfg.Connections <= 0 {
		cfg.Connections = 8
	}
	if cfg.StreamWindow <= 0 {
		cfg.StreamWindow = 16 << 20
	}
	return &Downloader{
		cfg:          cfg,
		segs:         map[*segment]struct{}{},
		priority:     -1,
		changed:      make(chan struct{}),
		streamWindow: cfg.StreamWindow,
	}
}

// Received is the total bytes received from the network, for speed sampling.
func (d *Downloader) Received() int64 { return d.received.Load() }

// Running reports whether Run is in progress.
func (d *Downloader) Running() bool { return d.running.Load() }

// Info is what the server said about the file, after Run has started.
func (d *Downloader) Info() Info {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.info
}

// SetConnections changes the connection count while running.
func (d *Downloader) SetConnections(n int) {
	d.mu.Lock()
	d.cfg.Connections = max(1, n)
	d.maxConns = d.cfg.Connections
	d.spawnLocked()
	d.mu.Unlock()
}

// SetRequest replaces the URL and headers, for an expired link refreshed from
// the page. It applies to connections opened from now on.
func (d *Downloader) SetRequest(r Request) {
	d.mu.Lock()
	d.cfg.Request = r
	d.mu.Unlock()
}

func (d *Downloader) request() Request {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.cfg.Request
}

// Progress is bytes done and the total (-1 when unknown).
func (d *Downloader) Progress() (done, size int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.blocks == nil {
		return d.singleDone, d.info.Size
	}
	return d.blocks.DoneBytes(), d.blocks.Size
}

// Snapshot returns a coarse block map and the active segments for display.
func (d *Downloader) Snapshot(buckets int) ([]float32, [][3]int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var segs [][3]int64
	for s := range d.segs {
		segs = append(segs, [3]int64{s.start, s.pos.Load(), s.end.Load()})
	}
	if d.blocks == nil {
		if d.info.Size > 0 {
			return []float32{float32(d.singleDone) / float32(d.info.Size)}, segs
		}
		return nil, segs
	}
	return d.blocks.Coarse(buckets), segs
}

// Connections is the number of open connections.
func (d *Downloader) Connections() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.live
}

// Run downloads until the file is complete, ctx is cancelled (a pause), or an
// error that retrying can't fix.
func (d *Downloader) Run(ctx context.Context) error {
	if !d.running.CompareAndSwap(false, true) {
		return errors.New("already running")
	}
	defer d.running.Store(false)

	info, err := Probe(ctx, d.cfg.Client, d.request())
	if err != nil {
		return err
	}
	if err := d.prepare(info); err != nil {
		return err
	}
	if d.blocks == nil {
		err = d.runSingle(ctx)
	} else {
		err = d.runSegmented(ctx)
	}
	if err != nil {
		d.saveState()
		d.Close()
		return err
	}
	return d.finish()
}

// prepare loads resume data, checks the server's file is the same one, and
// opens the file on disk.
func (d *Downloader) prepare(info Info) error {
	saved := d.loadState()
	d.mu.Lock()
	d.info = info
	d.maxConns = d.cfg.Connections
	switch {
	case !info.Ranges || info.Size < 0:
		d.blocks = nil
	case saved != nil && saved.matches(info):
		d.blocks = saved.blocks
	default:
		d.blocks = NewBlockMap(info.Size)
	}
	d.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(d.cfg.Path), 0o755); err != nil {
		return err
	}
	path := d.cfg.Path + fileutil.PartSuffix
	d.final = false
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if _, err := os.Stat(d.cfg.Path); err == nil && d.blocks != nil && d.blocks.Count() > 0 {
			// A finished file being repaired: work on it in place.
			path, d.final = d.cfg.Path, true
		}
	}
	_, statErr := os.Stat(path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	if os.IsNotExist(statErr) {
		// The partial file is gone, so whatever the resume data says is done isn't.
		d.mu.Lock()
		if d.blocks != nil {
			d.blocks.Reset()
		}
		d.mu.Unlock()
	}
	if d.blocks != nil {
		if st, err := f.Stat(); err == nil && st.Size() != info.Size {
			if err := f.Truncate(info.Size); err != nil {
				f.Close()
				return fmt.Errorf("reserving space: %w", err)
			}
		}
	}
	d.fileMu.Lock()
	d.file = f
	d.fileMu.Unlock()
	return nil
}

func (d *Downloader) runSegmented(ctx context.Context) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	stopSaving := d.saveEvery(2 * time.Second)
	d.mu.Lock()
	d.runCtx, d.cancelRun, d.lastErr = ctx, cancel, nil
	d.idle = sync.NewCond(&d.mu)
	d.spawnLocked()
	for d.live > 0 {
		d.idle.Wait()
	}
	d.runCtx, d.cancelRun = nil, nil
	complete, lastErr := d.blocks.Complete(), d.lastErr
	d.mu.Unlock()
	stopSaving()

	if err := context.Cause(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if !complete {
		if lastErr != nil {
			return lastErr
		}
		return errors.New("download stopped before the file was complete")
	}
	return nil
}

// spawnLocked opens connections while there is work and room for them.
// Called with d.mu held.
func (d *Downloader) spawnLocked() {
	if d.runCtx == nil {
		return
	}
	for d.live < d.maxConns && d.hasWork() {
		d.live++
		go d.worker(d.runCtx)
	}
}

// hasWork reports whether a new connection would find something to do.
// Called with d.mu held.
func (d *Downloader) hasWork() bool {
	if d.blocks.Complete() {
		return false
	}
	claimed := d.claimedMask()
	if _, n := d.largestFreeRun(claimed); n > 0 {
		return true
	}
	_, rem := d.largestSegment()
	return rem >= 2*minSplit
}

// saveEvery persists resume data periodically until the returned func is called.
func (d *Downloader) saveEvery(every time.Duration) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				d.saveState()
			}
		}
	}()
	return func() { close(stop); <-done }
}

// finish flushes the file, gives it its final name and records the result.
func (d *Downloader) finish() error {
	if err := d.closeAndRename(); err != nil {
		return err
	}
	d.saveState()
	return nil
}

func (d *Downloader) closeAndRename() error {
	d.fileMu.Lock()
	defer d.fileMu.Unlock()
	if d.file == nil {
		return nil
	}
	if err := d.file.Sync(); err != nil {
		return err
	}
	if err := d.file.Close(); err != nil {
		return err
	}
	d.file = nil
	if !d.final {
		if err := os.Rename(d.cfg.Path+fileutil.PartSuffix, d.cfg.Path); err != nil {
			return fmt.Errorf("renaming finished file: %w", err)
		}
		d.final = true
	}
	return nil
}

// Close releases the file. Call it when the download is removed or paused.
func (d *Downloader) Close() error {
	d.fileMu.Lock()
	defer d.fileMu.Unlock()
	if d.file == nil {
		return nil
	}
	err := d.file.Close()
	d.file = nil
	return err
}

// notify wakes everything waiting for new data. Called with d.mu held.
func (d *Downloader) notify() {
	close(d.changed)
	d.changed = make(chan struct{})
}
