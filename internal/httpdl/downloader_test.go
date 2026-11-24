package httpdl

import (
	"bytes"
	"context"
	"errors"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// testServer serves data with range support. delay slows each write so tests
// can catch a download mid-way.
type testServer struct {
	data      []byte
	ranges    bool
	delay     time.Duration
	status    atomic.Int32
	requests  atomic.Int32
	bytesSent atomic.Int64
}

func (s *testServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.requests.Add(1)
	if code := s.status.Load(); code != 0 {
		w.WriteHeader(int(code))
		return
	}
	if !s.ranges {
		r.Header.Del("Range")
		w.Header().Set("Content-Length", "")
	}
	w.Header().Set("ETag", `"v1"`)
	http.ServeContent(&countingWriter{w, s}, r, "file.bin", time.Unix(0, 0), bytes.NewReader(s.data))
}

type countingWriter struct {
	http.ResponseWriter
	s *testServer
}

func (c *countingWriter) Write(p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		n := min(len(p), 16<<10)
		if c.s.delay > 0 {
			time.Sleep(c.s.delay)
		}
		w, err := c.ResponseWriter.Write(p[:n])
		total += w
		c.s.bytesSent.Add(int64(w))
		if err != nil {
			return total, err
		}
		p = p[n:]
	}
	return total, nil
}

func randomData(n int) []byte {
	b := make([]byte, n)
	rand.New(rand.NewSource(1)).Read(b)
	return b
}

func newTest(t *testing.T, s *testServer) (*Downloader, string) {
	t.Helper()
	srv := httptest.NewServer(s)
	t.Cleanup(srv.Close)
	client, err := NewClient("")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "out.bin")
	d := New(Config{
		Request:     Request{URL: srv.URL + "/file.bin"},
		Path:        path,
		StatePath:   filepath.Join(dir, "state.json"),
		Connections: 6,
		Retries:     2,
		Client:      client,
	})
	return d, path
}

func checkFile(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("file differs from source (%d vs %d bytes)", len(got), len(want))
	}
}

func TestSegmentedDownload(t *testing.T) {
	s := &testServer{data: randomData(5<<20 + 12345), ranges: true}
	d, path := newTest(t, s)
	if err := d.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	checkFile(t, path, s.data)
	if s.requests.Load() < 3 {
		t.Fatalf("only %d requests: the file wasn't split", s.requests.Load())
	}
}

func TestPauseAndResume(t *testing.T) {
	s := &testServer{data: randomData(4 << 20), ranges: true, delay: 2 * time.Millisecond}
	d, path := newTest(t, s)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			if done, _ := d.Progress(); done > 1<<20 {
				cancel()
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()
	if err := d.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run = %v, want cancelled", err)
	}
	sentBefore := s.bytesSent.Load()
	s.delay = 0

	// A fresh downloader, as after a restart, continues from the saved state.
	d2 := New(d.cfg)
	if err := d2.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	checkFile(t, path, s.data)
	if resent := s.bytesSent.Load() - sentBefore; resent > int64(len(s.data))-512<<10 {
		t.Fatalf("resume fetched %d bytes again of %d", resent, len(s.data))
	}
}

func TestCheckAndRepair(t *testing.T) {
	s := &testServer{data: randomData(3 << 20), ranges: true}
	d, path := newTest(t, s)
	if err := d.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt([]byte("corruption!"), 1<<20+77); err != nil {
		t.Fatal(err)
	}
	f.Close()

	d2 := New(d.cfg)
	bad, err := d2.Check(context.Background(), nil)
	if err != nil || bad != 1 {
		t.Fatalf("Check = %d, %v; want 1 bad block", bad, err)
	}
	sent := s.bytesSent.Load()
	if err := d2.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	checkFile(t, path, s.data)
	if refetched := s.bytesSent.Load() - sent; refetched > 1<<20 {
		t.Fatalf("repair fetched %d bytes, expected one block", refetched)
	}
}

func TestNoRanges(t *testing.T) {
	s := &testServer{data: randomData(1 << 20), ranges: false}
	d, path := newTest(t, s)
	if err := d.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	checkFile(t, path, s.data)
}

func TestExpiredLink(t *testing.T) {
	s := &testServer{data: randomData(1 << 20), ranges: true}
	s.status.Store(http.StatusForbidden)
	d, _ := newTest(t, s)
	err := d.Run(context.Background())
	var se *StatusError
	if !errors.As(err, &se) || !se.Expired() {
		t.Fatalf("Run = %v, want an expired-link error", err)
	}
}

func TestSeekWhileDownloading(t *testing.T) {
	s := &testServer{data: randomData(16 << 20), ranges: true, delay: 3 * time.Millisecond}
	d, _ := newTest(t, s)
	d.SetConnections(2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.Run(ctx)

	// Ask for the end of the file right away, as a player seeking would.
	off := int64(15 << 20)
	wctx, wcancel := context.WithTimeout(ctx, 10*time.Second)
	defer wcancel()
	for d.Size() < 0 {
		time.Sleep(time.Millisecond)
	}
	n, err := d.WaitAvailable(wctx, off)
	if err != nil {
		t.Fatalf("WaitAvailable: %v", err)
	}
	buf := make([]byte, min(n, 4096))
	if _, err := d.ReadAt(buf, off); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf, s.data[off:off+int64(len(buf))]) {
		t.Fatal("streamed bytes differ")
	}
	if done, size := d.Progress(); done > size/2 {
		t.Fatalf("seek waited for %d of %d bytes: not prioritized", done, size)
	}
}
