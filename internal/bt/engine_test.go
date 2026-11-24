package bt

import (
	"bytes"
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"

	"github.com/godownload/godownload/internal/model"
)

// seed creates a file, a torrent for it, and a client seeding it.
func seed(t *testing.T) (data []byte, mi []byte, seeder *torrent.Client) {
	t.Helper()
	dir := t.TempDir()
	data = make([]byte, 3<<20+999)
	rand.New(rand.NewSource(2)).Read(data)
	if err := os.WriteFile(filepath.Join(dir, "movie.mp4"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	info := metainfo.Info{PieceLength: 256 << 10}
	if err := info.BuildFromFilePath(filepath.Join(dir, "movie.mp4")); err != nil {
		t.Fatal(err)
	}
	var m metainfo.MetaInfo
	var err error
	m.InfoBytes, err = bencode.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := m.Write(&buf); err != nil {
		t.Fatal(err)
	}

	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = dir
	cfg.Seed = true
	cfg.NoDHT = true
	cfg.NoDefaultPortForwarding = true
	cfg.ListenPort = 0
	seeder, err = torrent.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { seeder.Close() })
	st, err := seeder.AddTorrent(&m)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.VerifyData(); err != nil {
		t.Fatal(err)
	}
	return data, buf.Bytes(), seeder
}

func newEngine(t *testing.T) *Engine {
	t.Helper()
	e, err := NewEngine(Config{DataDir: t.TempDir(), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { e.Close() })
	return e
}

func TestDownloadAndStream(t *testing.T) {
	data, mi, seeder := seed(t)
	e := newEngine(t)
	out := t.TempDir()
	h, err := e.Open(Source{MetaInfo: mi}, out)
	if err != nil {
		t.Fatal(err)
	}
	h.Start(nil, false)
	h.t.AddClientPeer(seeder)

	// Stream from the middle first, as a player seeking would.
	r, _ := h.Reader(0, 1<<20)
	defer r.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r.SetContext(ctx)
	if _, err := r.Seek(2<<20, 0); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, 4096)
	if _, err := r.Read(got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data[2<<20:2<<20+4096]) {
		t.Fatal("streamed bytes differ")
	}

	for {
		if done, wanted := h.Progress(); wanted > 0 && done == wanted {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("download did not finish")
		case <-time.After(50 * time.Millisecond):
		}
	}
	disk, err := os.ReadFile(filepath.Join(out, "movie.mp4"))
	if err != nil || !bytes.Equal(disk, data) {
		t.Fatalf("file on disk differs: %v", err)
	}
	if files := h.Files(); len(files) != 1 || files[0].Done != int64(len(data)) {
		t.Fatalf("Files() = %+v", files)
	}
}

func TestSkippedFileNotWanted(t *testing.T) {
	_, mi, _ := seed(t)
	e := newEngine(t)
	h, err := e.Open(Source{MetaInfo: mi}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	h.Start([]model.FilePriority{model.PrioritySkip}, true)
	time.Sleep(100 * time.Millisecond)
	if _, wanted := h.Progress(); wanted != 0 {
		t.Fatalf("wanted = %d with every file skipped", wanted)
	}
}

func TestParseSource(t *testing.T) {
	if _, ok := ParseSource("magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567"); !ok {
		t.Fatal("magnet not recognized")
	}
	if _, ok := ParseSource("0123456789abcdef0123456789abcdef01234567"); !ok {
		t.Fatal("info hash not recognized")
	}
	if _, ok := ParseSource("https://example.com/a.zip"); ok {
		t.Fatal("URL taken for a torrent")
	}
}

func TestVerifyRepairsCorruption(t *testing.T) {
	data, mi, seeder := seed(t)
	e := newEngine(t)
	out := t.TempDir()
	h, err := e.Open(Source{MetaInfo: mi}, out)
	if err != nil {
		t.Fatal(err)
	}
	h.Start(nil, true)
	h.t.AddClientPeer(seeder)
	waitDone(t, h)
	h.Close()

	path := filepath.Join(out, "movie.mp4")
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteAt([]byte("damage"), 1<<20+5)
	f.Close()

	h, err = e.Open(Source{InfoHash: h.InfoHash()}, out)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Verify(context.Background()); err != nil {
		t.Fatal(err)
	}
	if done, wanted := h.Progress(); done >= wanted {
		t.Fatal("verify missed the damage")
	}
	h.Start(nil, true)
	h.t.AddClientPeer(seeder)
	waitDone(t, h)
	disk, _ := os.ReadFile(path)
	if !bytes.Equal(disk, data) {
		t.Fatal("not repaired")
	}
}

func waitDone(t *testing.T, h *Handle) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if done, wanted := h.Progress(); wanted > 0 && done == wanted {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("download did not finish")
}
