package ratelimit

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestReaderThrottles(t *testing.T) {
	data := bytes.Repeat([]byte{1}, 96*1024)
	l := New(64 * 1024)
	start := time.Now()
	n, err := io.Copy(io.Discard, Reader(context.Background(), bytes.NewReader(data), l, nil))
	if err != nil || n != int64(len(data)) {
		t.Fatalf("copy: %d, %v", n, err)
	}
	// 96 KB at 64 KB/s with a 16 KB burst takes about 1.25 s.
	if el := time.Since(start); el < time.Second {
		t.Fatalf("finished in %v, limit not applied", el)
	}
}

func TestUnlimited(t *testing.T) {
	l := New(0)
	if l.Rate() != 0 {
		t.Fatal("zero should be unlimited")
	}
	data := bytes.Repeat([]byte{1}, 4<<20)
	start := time.Now()
	if _, err := io.Copy(io.Discard, Reader(context.Background(), bytes.NewReader(data), l)); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatal("unlimited reader was throttled")
	}
}
