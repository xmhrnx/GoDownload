package bt

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/anacrolix/torrent/metainfo"
)

func TestReopenFindsMagnetPeer(t *testing.T) {
	_, mi, seeder := seed(t)
	m, err := metainfo.Load(bytes.NewReader(mi))
	if err != nil {
		t.Fatal(err)
	}
	info, _ := m.UnmarshalInfo()
	magnet := m.Magnet(nil, &info).String() + fmt.Sprintf("&x.pe=127.0.0.1:%d", seeder.LocalPort())
	e := newEngine(t)
	out := t.TempDir()
	h, err := e.Open(Source{Magnet: magnet}, out)
	if err != nil {
		t.Fatal(err)
	}
	h.Start(nil, true)
	waitDone(t, h)
	h.Close()

	f, err := os.OpenFile(filepath.Join(out, "movie.mp4"), os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteAt([]byte("damage"), 2<<20)
	f.Close()

	// Reopened from the saved metadata, the magnet's peer must still be dialed.
	h, err = e.Open(Source{Magnet: magnet}, out)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Verify(context.Background()); err != nil {
		t.Fatal(err)
	}
	h.Start(nil, true)
	waitDone(t, h)
}
