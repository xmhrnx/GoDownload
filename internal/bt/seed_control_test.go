package bt

import (
	"bytes"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
)

// seedFor makes the engine seed a file and returns a separate client that
// downloads it from the engine.
func seedFor(t *testing.T, size int) (*Handle, *torrent.Torrent) {
	t.Helper()
	dir := t.TempDir()
	data := make([]byte, size)
	rand.New(rand.NewSource(4)).Read(data)
	if err := os.WriteFile(filepath.Join(dir, "f.bin"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	info := metainfo.Info{PieceLength: 256 << 10}
	if err := info.BuildFromFilePath(filepath.Join(dir, "f.bin")); err != nil {
		t.Fatal(err)
	}
	var mi metainfo.MetaInfo
	mi.InfoBytes, _ = bencode.Marshal(info)
	var buf bytes.Buffer
	if err := mi.Write(&buf); err != nil {
		t.Fatal(err)
	}
	e := newEngine(t)
	h, err := e.Open(Source{MetaInfo: buf.Bytes()}, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Verify(t.Context()); err != nil {
		t.Fatal(err)
	}
	h.Start(nil, true)
	return h, leech(t, &mi, h)
}

func leech(t *testing.T, mi *metainfo.MetaInfo, h *Handle) *torrent.Torrent {
	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir, cfg.NoDHT, cfg.ListenPort, cfg.NoDefaultPortForwarding = t.TempDir(), true, 0, true
	cl, err := torrent.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cl.Close() })
	lt, err := cl.AddTorrent(mi)
	if err != nil {
		t.Fatal(err)
	}
	lt.DownloadAll()
	h.t.AddClientPeer(cl)
	return lt
}

// rate is how fast lt downloads over d, in bytes per second.
func rate(lt *torrent.Torrent, d time.Duration) float64 {
	a := lt.BytesCompleted()
	time.Sleep(d)
	return float64(lt.BytesCompleted()-a) / d.Seconds()
}

func TestUploadLimitAndSeedSwitch(t *testing.T) {
	h, lt := seedFor(t, 64<<20)
	h.SetUploadLimit(1 << 20)
	time.Sleep(time.Second) // connect and get going
	if r := rate(lt, 3*time.Second); r > 1.4*(1<<20) || r < 0.5*(1<<20) {
		t.Fatalf("limited to 1 MB/s, uploaded %.2f MB/s", r/(1<<20))
	}
	h.SetUploadLimit(3 << 20)
	time.Sleep(time.Second)
	if r := rate(lt, 3*time.Second); r > 3.6*(1<<20) || r < 1.8*(1<<20) {
		t.Fatalf("limited to 3 MB/s, uploaded %.2f MB/s", r/(1<<20))
	}
	h.SetSeed(false)
	time.Sleep(time.Second) // requests already being sent may finish
	if r := rate(lt, 3*time.Second); r > 0 {
		t.Fatalf("seeding off, still uploaded %.2f MB/s", r/(1<<20))
	}
	if lt.BytesCompleted() == lt.Length() {
		t.Fatal("download finished before the switch was tested")
	}
}
