package hls

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/godownload/godownload/internal/fileutil"
	"github.com/godownload/godownload/internal/httpdl"
	"github.com/godownload/godownload/internal/ratelimit"
)

// Config describes one HLS download.
type Config struct {
	Request     httpdl.Request
	Path        string
	StatePath   string
	Connections int
	Retries     int
	Client      *http.Client
	Limiters    []*ratelimit.Limiter
}

// Downloader fetches every segment of a stream, several at once, and writes
// them in order into one file. It resumes at the first unwritten segment.
type Downloader struct {
	cfg Config

	mu       sync.Mutex
	total    int
	written  int
	offset   int64
	duration float64
	keys     map[string][]byte

	received atomic.Int64
}

type state struct {
	Written int   `json:"written"`
	Offset  int64 `json:"offset"`
}

// New prepares a download; nothing happens until Run.
func New(cfg Config) *Downloader {
	if cfg.Connections <= 0 {
		cfg.Connections = 6
	}
	return &Downloader{cfg: cfg, keys: map[string][]byte{}}
}

// Received is the total bytes received, for speed sampling.
func (d *Downloader) Received() int64 { return d.received.Load() }

// Progress is segments written and their total, and the bytes written.
func (d *Downloader) Progress() (written, total int, bytes int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.written, d.total, d.offset
}

// Extension is the container the stream is in: ".mp4" for fMP4, else ".ts".
func Extension(ctx context.Context, client *http.Client, r httpdl.Request) (string, error) {
	pl, err := load(ctx, client, r)
	if err != nil {
		return "", err
	}
	if pl.Init != "" {
		return ".mp4", nil
	}
	return ".ts", nil
}

// load fetches the playlist, following a master playlist to its best variant.
func load(ctx context.Context, client *http.Client, r httpdl.Request) (*Playlist, error) {
	for range 3 {
		body, base, err := get(ctx, client, r)
		if err != nil {
			return nil, err
		}
		pl, variants, err := parse(body, base)
		body.Close()
		if err != nil {
			return nil, err
		}
		if len(variants) == 0 {
			return pl, nil
		}
		r.URL = best(variants).URL
	}
	return nil, errors.New("playlist nests too deeply")
}

func get(ctx context.Context, client *http.Client, r httpdl.Request) (io.ReadCloser, *url.URL, error) {
	req, err := r.Build(ctx)
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, nil, &httpdl.StatusError{Code: resp.StatusCode}
	}
	return resp.Body, resp.Request.URL, nil
}

// Run downloads until the stream is complete, ctx ends, or a segment fails
// more often than Retries allows.
func (d *Downloader) Run(ctx context.Context) error {
	pl, err := load(ctx, d.cfg.Client, d.cfg.Request)
	if err != nil {
		return err
	}
	if !pl.Ended {
		return ErrLive
	}
	var st state
	if err := fileutil.ReadJSON(d.cfg.StatePath, &st); err != nil {
		st = state{}
	}
	part := d.cfg.Path + fileutil.PartSuffix
	f, err := os.OpenFile(part, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if fi, err := f.Stat(); err != nil || fi.Size() < st.Offset {
		st = state{} // the partial file is shorter than recorded: start over
	}
	if err := f.Truncate(st.Offset); err != nil {
		return err
	}
	d.mu.Lock()
	d.total, d.written, d.offset = len(pl.Segments), st.Written, st.Offset
	for _, s := range pl.Segments {
		d.duration += s.Duration
	}
	d.mu.Unlock()

	if st.Written == 0 && st.Offset == 0 && pl.Init != "" {
		data, err := d.fetch(ctx, pl.Init)
		if err != nil {
			return err
		}
		if err := d.write(f, data, 0); err != nil {
			return err
		}
	}
	if err := d.fetchAll(ctx, f, pl.Segments[st.Written:]); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	f.Close()
	if err := os.Rename(part, d.cfg.Path); err != nil {
		return err
	}
	os.Remove(d.cfg.StatePath)
	return nil
}

type result struct {
	data []byte
	err  error
}

// fetchAll downloads segments with a few workers and writes them in order.
// Each segment gets a slot; the writer waits on slots in order, so at most
// a window's worth of segments sits in memory.
func (d *Downloader) fetchAll(ctx context.Context, f *os.File, segs []Segment) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	window := d.cfg.Connections * 2
	slots := make([]chan result, len(segs))
	for i := range slots {
		slots[i] = make(chan result, 1)
	}
	sem := make(chan struct{}, window)
	jobs := make(chan int)
	go func() {
		defer close(jobs)
		for i := range segs {
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			select {
			case jobs <- i:
			case <-ctx.Done():
				return
			}
		}
	}()
	for range d.cfg.Connections {
		go func() {
			for i := range jobs {
				data, err := d.fetchSegment(ctx, segs[i])
				slots[i] <- result{data, err}
			}
		}()
	}
	for i := range segs {
		var r result
		select {
		case r = <-slots[i]:
		case <-ctx.Done():
			return ctx.Err()
		}
		if r.err != nil {
			return r.err
		}
		if err := d.write(f, r.data, 1); err != nil {
			return err
		}
		<-sem
	}
	return nil
}

// write appends data and records progress; segments counts the segments it holds.
func (d *Downloader) write(f *os.File, data []byte, segments int) error {
	d.mu.Lock()
	off := d.offset
	d.mu.Unlock()
	if _, err := f.WriteAt(data, off); err != nil {
		return err
	}
	d.mu.Lock()
	d.offset += int64(len(data))
	d.written += segments
	st := state{Written: d.written, Offset: d.offset}
	d.mu.Unlock()
	if d.cfg.StatePath != "" && st.Written%10 == 0 {
		if err := f.Sync(); err != nil {
			return err
		}
		return fileutil.WriteJSON(d.cfg.StatePath, st)
	}
	return nil
}

func (d *Downloader) fetchSegment(ctx context.Context, s Segment) ([]byte, error) {
	data, err := d.fetch(ctx, s.URL)
	if err != nil || s.Key == nil {
		return data, err
	}
	key, err := d.key(ctx, s.Key.URL)
	if err != nil {
		return nil, err
	}
	return decrypt(data, key, s.Key, s.Seq)
}

func (d *Downloader) key(ctx context.Context, u string) ([]byte, error) {
	d.mu.Lock()
	k, ok := d.keys[u]
	d.mu.Unlock()
	if ok {
		return k, nil
	}
	k, err := d.fetch(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("fetching key: %w", err)
	}
	if len(k) != 16 {
		return nil, errors.New("the stream's key isn't a 16-byte AES key")
	}
	d.mu.Lock()
	d.keys[u] = k
	d.mu.Unlock()
	return k, nil
}

// fetch gets a whole small resource, retrying with backoff.
func (d *Downloader) fetch(ctx context.Context, u string) ([]byte, error) {
	r := d.cfg.Request
	r.URL = u
	var lastErr error
	for attempt := 0; attempt <= d.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
		body, _, err := get(ctx, d.cfg.Client, r)
		if err == nil {
			var data []byte
			data, err = io.ReadAll(ratelimit.Reader(ctx, body, d.cfg.Limiters...))
			body.Close()
			if err == nil {
				d.received.Add(int64(len(data)))
				return data, nil
			}
		}
		lastErr = err
		var se *httpdl.StatusError
		if ctx.Err() != nil || (errors.As(err, &se) && !se.Retryable()) {
			return nil, err
		}
	}
	return nil, lastErr
}
