package manager

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/internal/store"
)

func newManager(t *testing.T) (*Manager, chan Event) {
	t.Helper()
	dir := t.TempDir()
	st := store.Open(filepath.Join(dir, "data"))
	s := model.DefaultSettings(filepath.Join(dir, "Downloads"))
	if err := st.SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	events := make(chan Event, 16)
	m, err := New(st, nil, func(e Event) { events <- e })
	if err != nil {
		t.Fatal(err)
	}
	m.Start()
	t.Cleanup(func() { m.Close() })
	return m, events
}

func waitStatus(t *testing.T, m *Manager, id string, want model.Status) model.Item {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		it, _, err := m.Detail(id)
		if err != nil {
			t.Fatal(err)
		}
		if it.Status == want {
			return it
		}
		if it.Status == model.StatusFailed && want != model.StatusFailed {
			t.Fatalf("download failed: %s", it.Error)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("status never became %s", want)
	return model.Item{}
}

func TestAddStreamRepairRemove(t *testing.T) {
	data := make([]byte, 3<<20)
	rand.New(rand.NewSource(3)).Read(data)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "clip.mp4", time.Unix(0, 0), bytes.NewReader(data))
	}))
	defer srv.Close()

	m, events := newManager(t)
	ctx := context.Background()
	p, err := m.Probe(ctx, srv.URL+"/clip.mp4", model.RequestHeaders{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Category != model.CategoryVideo || p.Size != int64(len(data)) || !p.Resumable {
		t.Fatalf("probe = %+v", p)
	}
	if filepath.Base(p.Dir) != "Video" {
		t.Fatalf("default folder = %s, want …/Video", p.Dir)
	}
	id, err := m.Add(AddRequest{Kind: p.Kind, URL: p.URL, Name: p.Name, Dir: p.Dir, Category: p.Category, Size: p.Size})
	if err != nil {
		t.Fatal(err)
	}

	// Stream while (or after) it downloads.
	media, err := m.Media(ctx, id, 0)
	if err != nil {
		t.Fatal(err)
	}
	r, err := media.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Seek(2<<20, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, 1000)
	if _, err := io.ReadFull(r, got); err != nil {
		t.Fatal(err)
	}
	r.Close()
	if !bytes.Equal(got, data[2<<20:2<<20+1000]) {
		t.Fatal("streamed bytes differ")
	}

	waitStatus(t, m, id, model.StatusCompleted)
	if ev := <-events; ev.Kind != EventCompleted {
		t.Fatalf("event = %+v", ev)
	}
	path, _ := m.Path(id)

	// Damage the file, then repair it.
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteAt([]byte("broken"), 100)
	f.Close()
	if err := m.Repair(id); err != nil {
		t.Fatal(err)
	}
	ev := <-events
	if ev.Kind != EventInfo || ev.Message == "No damage found." {
		t.Fatalf("repair event = %+v", ev)
	}
	waitStatus(t, m, id, model.StatusCompleted)
	disk, _ := os.ReadFile(path)
	if !bytes.Equal(disk, data) {
		t.Fatal("repaired file differs")
	}

	if err := m.Remove(true, id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file not deleted")
	}
	if len(m.Snapshot()) != 0 {
		t.Fatal("download still listed")
	}
}

func TestRemoveKeepsFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "a.zip", time.Unix(0, 0), bytes.NewReader([]byte("hello")))
	}))
	defer srv.Close()
	m, _ := newManager(t)
	id, err := m.Add(AddRequest{Kind: model.KindHTTP, URL: srv.URL + "/a.zip", Name: "a.zip"})
	if err != nil {
		t.Fatal(err)
	}
	waitStatus(t, m, id, model.StatusCompleted)
	path, _ := m.Path(id)
	if err := m.Remove(false, id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("removing from the list deleted the file")
	}
}

func TestWhenDoneFiresOnce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("x"))
	}))
	defer srv.Close()
	m, events := newManager(t)
	s := m.Settings()
	s.WhenDone = model.PowerSleep
	if err := m.UpdateSettings(s); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Add(AddRequest{Kind: model.KindHTTP, URL: srv.URL + "/x.bin", Name: "x.bin"}); err != nil {
		t.Fatal(err)
	}
	timeout := time.After(10 * time.Second)
	for {
		select {
		case ev := <-events:
			if ev.Kind == EventAllDone {
				if ev.Action != model.PowerSleep || m.Settings().WhenDone != model.PowerNone {
					t.Fatalf("event = %+v, setting %v", ev, m.Settings().WhenDone)
				}
				return
			}
		case <-timeout:
			t.Fatal("no all-done event")
		}
	}
}

func TestParseChecksum(t *testing.T) {
	got, err := ParseChecksum("E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855")
	if err != nil || got[:7] != "sha256:" {
		t.Fatalf("ParseChecksum = %q, %v", got, err)
	}
	if _, err := ParseChecksum("crc:zz"); err == nil {
		t.Fatal("accepted a bad checksum")
	}
}
