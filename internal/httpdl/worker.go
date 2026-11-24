package httpdl

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/godownload/godownload/internal/ratelimit"
)

var crcTable = crc32.MakeTable(crc32.Castagnoli)

// stallTimeout ends a connection that has sent nothing for this long.
const stallTimeout = 30 * time.Second

var bufPool = sync.Pool{New: func() any { b := make([]byte, 32<<10); return &b }}

// worker is one connection: it claims work, fetches it, and retries with
// backoff until there is nothing left or it has failed too often.
func (d *Downloader) worker(ctx context.Context) {
	err := d.work(ctx)
	d.mu.Lock()
	defer d.mu.Unlock()
	d.live--
	if err != nil {
		d.lastErr = err
		var se *StatusError
		if errors.As(err, &se) && se.Expired() && d.cancelRun != nil {
			d.cancelRun(err)
		}
	}
	if d.live == 0 {
		d.idle.Broadcast()
	}
}

func (d *Downloader) work(ctx context.Context) error {
	fails := 0
	for {
		d.mu.Lock()
		seg := d.claim()
		d.mu.Unlock()
		if seg == nil {
			return nil
		}
		err := d.fetch(ctx, seg)
		d.release(seg)
		if err == nil {
			fails = 0
			continue
		}
		if ctx.Err() != nil {
			return nil
		}
		var se *StatusError
		if errors.As(err, &se) {
			if se.Expired() {
				return err
			}
			if (se.Code == http.StatusTooManyRequests || se.Code == http.StatusServiceUnavailable) && d.giveUpSeat() {
				// The server limits connections; carry on with fewer.
				return nil
			}
		}
		fails++
		if fails > d.cfg.Retries {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff(fails)):
		}
	}
}

func backoff(fails int) time.Duration {
	d := time.Second << min(fails-1, 5)
	return min(d, 30*time.Second)
}

// giveUpSeat lowers the connection limit when others are still working.
func (d *Downloader) giveUpSeat() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.live <= 1 {
		return false
	}
	d.maxConns = max(1, d.live-1)
	return true
}

func (d *Downloader) release(s *segment) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.segs, s)
	// The freed range may be work for a connection that isn't open yet.
	d.spawnLocked()
}

// fetch downloads seg, marking each block as it completes.
func (d *Downloader) fetch(ctx context.Context, seg *segment) error {
	pos, end := seg.pos.Load(), seg.end.Load()
	if pos >= end {
		return nil
	}
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	req, err := d.request().Build(ctx)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", pos, end-1))
	resp, err := d.cfg.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusPartialContent:
		start, err := startFromContentRange(resp.Header.Get("Content-Range"))
		if err != nil {
			return err
		}
		if start != pos {
			return fmt.Errorf("server sent bytes from %d, asked for %d", start, pos)
		}
	case http.StatusOK:
		if pos != 0 {
			return errors.New("the server stopped accepting ranges")
		}
	default:
		return &StatusError{Code: resp.StatusCode}
	}

	stall := time.AfterFunc(stallTimeout, func() { cancel(errors.New("connection stalled")) })
	defer stall.Stop()
	body := ratelimit.Reader(ctx, resp.Body, d.cfg.Limiters...)
	bufp := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufp)
	buf := *bufp

	var crc uint32
	for {
		pos, end = seg.pos.Load(), seg.end.Load()
		if pos >= end {
			return nil
		}
		idx := d.blocks.Index(pos)
		_, blockEnd := d.blocks.Bounds(idx)
		want := min(int64(len(buf)), end-pos, blockEnd-pos)
		n, rerr := body.Read(buf[:want])
		if n > 0 {
			stall.Reset(stallTimeout)
			if err := d.writeAt(buf[:n], pos); err != nil {
				return err
			}
			crc = crc32.Update(crc, crcTable, buf[:n])
			d.received.Add(int64(n))
			pos += int64(n)
			seg.pos.Store(pos)
			if pos == blockEnd {
				d.mu.Lock()
				d.blocks.Mark(idx, crc)
				d.notify()
				d.mu.Unlock()
				crc = 0
			}
		}
		if rerr == io.EOF {
			if pos < seg.end.Load() {
				return io.ErrUnexpectedEOF
			}
			return nil
		}
		if rerr != nil {
			if cause := context.Cause(ctx); cause != nil {
				return cause
			}
			return rerr
		}
	}
}

func (d *Downloader) writeAt(p []byte, off int64) error {
	d.fileMu.RLock()
	defer d.fileMu.RUnlock()
	if d.file == nil {
		return errors.New("file closed")
	}
	_, err := d.file.WriteAt(p, off)
	return err
}
