package httpdl

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/godownload/godownload/internal/fileutil"
)

// SetStreamWindow changes how far ahead of a player's position is fetched first.
func (d *Downloader) SetStreamWindow(n int64) {
	d.mu.Lock()
	d.streamWindow = max(n, 1<<20)
	d.mu.Unlock()
}

// Size is the file's size once known, else -1.
func (d *Downloader) Size() int64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.info.URL == "" {
		return -1
	}
	return d.info.Size
}

// WaitAvailable blocks until at least one byte at off is on disk and returns
// how many contiguous bytes from off can be read. While it waits, the bytes at
// off become the most urgent work: a connection is moved there if none is
// about to reach them. That is what makes seeking in a player work while the
// file downloads.
func (d *Downloader) WaitAvailable(ctx context.Context, off int64) (int64, error) {
	for {
		d.mu.Lock()
		if d.info.URL != "" && d.info.Size >= 0 && off >= d.info.Size {
			d.mu.Unlock()
			return 0, io.EOF
		}
		avail := d.availableLocked(off)
		d.prioritizeLocked(off, avail == 0)
		changed := d.changed
		d.mu.Unlock()
		if avail > 0 {
			return avail, nil
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-changed:
		case <-time.After(time.Second):
		}
	}
}

func (d *Downloader) availableLocked(off int64) int64 {
	if d.blocks != nil {
		return d.blocks.DoneFrom(off)
	}
	if off < d.singleDone {
		return d.singleDone - off
	}
	return 0
}

// prioritizeLocked points new work at off. When urgent (off isn't on disk),
// it also cuts the connection that would reach off only after a long detour,
// and frees a connection for the stream if all are busy elsewhere.
func (d *Downloader) prioritizeLocked(off int64, urgent bool) {
	if d.blocks == nil {
		return
	}
	d.priority = off
	if !urgent {
		return
	}
	start, _ := d.blocks.Bounds(d.blocks.Index(off))
	for s := range d.segs {
		pos, end := s.pos.Load(), s.end.Load()
		if off < pos || off >= end {
			continue
		}
		if off-pos <= 2*d.blocks.BlockSize {
			return // arriving any moment
		}
		d.truncateAt(s, start)
		break
	}
	if d.live >= d.maxConns {
		var far *segment
		var farDist int64 = -1
		for s := range d.segs {
			pos := s.pos.Load()
			if pos >= off && pos < off+d.streamWindow {
				continue // already feeding the stream
			}
			dist := pos - off
			if dist < 0 {
				dist = -dist
			}
			if dist > farDist {
				far, farDist = s, dist
			}
		}
		if far != nil {
			d.truncateAt(far, 0)
		}
	}
	d.spawnLocked()
}

// ReadAt reads bytes that are on disk; it never waits. Use WaitAvailable first.
func (d *Downloader) ReadAt(p []byte, off int64) (int, error) {
	d.fileMu.RLock()
	f := d.file
	if f != nil {
		n, err := f.ReadAt(p, off)
		d.fileMu.RUnlock()
		return n, err
	}
	d.fileMu.RUnlock()

	// Not running: read whichever copy is on disk.
	path := d.cfg.Path
	if _, err := os.Stat(path + fileutil.PartSuffix); err == nil {
		path += fileutil.PartSuffix
	}
	rf, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer rf.Close()
	return rf.ReadAt(p, off)
}
