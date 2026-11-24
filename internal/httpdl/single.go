package httpdl

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/godownload/godownload/internal/ratelimit"
)

// runSingle downloads over one connection from the start, for servers that
// can't send ranges (or don't say how big the file is). It can't resume, so
// each attempt starts over.
func (d *Downloader) runSingle(ctx context.Context) error {
	fails := 0
	for {
		err := d.fetchWhole(ctx)
		if err == nil || ctx.Err() != nil {
			return err
		}
		var se *StatusError
		if errors.As(err, &se) && !se.Retryable() {
			return err
		}
		fails++
		if fails > d.cfg.Retries {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff(fails)):
		}
	}
}

func (d *Downloader) fetchWhole(ctx context.Context) error {
	d.mu.Lock()
	d.singleDone = 0
	d.live = 1
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		d.live = 0
		d.mu.Unlock()
	}()

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	req, err := d.request().Build(ctx)
	if err != nil {
		return err
	}
	resp, err := d.cfg.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &StatusError{Code: resp.StatusCode}
	}
	if err := d.truncate(); err != nil {
		return err
	}
	stall := time.AfterFunc(stallTimeout, func() { cancel(errors.New("connection stalled")) })
	defer stall.Stop()
	body := ratelimit.Reader(ctx, resp.Body, d.cfg.Limiters...)
	bufp := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufp)
	buf := *bufp
	var pos int64
	for {
		n, rerr := body.Read(buf)
		if n > 0 {
			stall.Reset(stallTimeout)
			if err := d.writeAt(buf[:n], pos); err != nil {
				return err
			}
			pos += int64(n)
			d.received.Add(int64(n))
			d.mu.Lock()
			d.singleDone = pos
			d.notify()
			d.mu.Unlock()
		}
		if rerr == io.EOF {
			if resp.ContentLength > 0 && pos < resp.ContentLength {
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

func (d *Downloader) truncate() error {
	d.fileMu.RLock()
	defer d.fileMu.RUnlock()
	if d.file == nil {
		return errors.New("file closed")
	}
	return d.file.Truncate(0)
}
