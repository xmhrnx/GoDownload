package manager

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/godownload/godownload/internal/bt"
	"github.com/godownload/godownload/internal/hls"
	"github.com/godownload/godownload/internal/httpdl"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/internal/ratelimit"
	"github.com/godownload/godownload/internal/speed"
)

// entry is a download and everything running for it. Its fields are guarded
// by the manager's mutex.
type entry struct {
	d       model.Download
	stats   model.Stats
	limiter *ratelimit.Limiter

	cancel context.CancelFunc
	done   chan struct{}
	// busy is set while a check runs outside the runner, so the queue leaves
	// the download alone.
	busy bool
	// seedDone is set when seeding reached its ratio or time.
	seedDone bool
	// verify asks the next torrent run to rehash its data first.
	verify bool

	http    *httpdl.Downloader
	hls     *hls.Downloader
	torrent *bt.Handle

	sample  sample
	streams int
	log     []string
}

// sample measures a running download's speeds.
type sample struct {
	down, up speed.Meter
	// uploadSeen is the session upload counter already added to the total.
	uploadSeen int64
}

func newEntry(d model.Download) *entry {
	return &entry{d: d, limiter: ratelimit.New(d.SpeedLimit), stats: model.Stats{ETA: -1}}
}

func (e *entry) running() bool { return e.done != nil }

// stop cancels the runner; it finishes on its own goroutine.
func (e *entry) stop() {
	if e.cancel != nil {
		e.cancel()
	}
}

// path is where the finished file (or torrent folder) is.
func (e *entry) path() string {
	return filepath.Join(e.d.Dir, e.d.Name)
}

// logf records a line in the download's log, shown in the details panel.
func (e *entry) logf(format string, args ...any) {
	line := time.Now().Format("15:04:05 ") + fmt.Sprintf(format, args...)
	e.log = append(e.log, line)
	if len(e.log) > 200 {
		e.log = e.log[len(e.log)-200:]
	}
}

func (e *entry) item() model.Item {
	return model.Item{Download: e.d, Stats: e.stats}
}
