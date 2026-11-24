package hls

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/godownload/godownload/internal/httpdl"
)

func encrypt(data, key []byte, seq int64) []byte {
	pad := aes.BlockSize - len(data)%aes.BlockSize
	data = append(append([]byte{}, data...), bytes.Repeat([]byte{byte(pad)}, pad)...)
	iv := make([]byte, 16)
	binary.BigEndian.PutUint64(iv[8:], uint64(seq))
	block, _ := aes.NewCipher(key)
	out := make([]byte, len(data))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, data)
	return out
}

func TestDownloadEncryptedStream(t *testing.T) {
	key := []byte("0123456789abcdef")
	var want bytes.Buffer
	segs := map[string][]byte{}
	var media strings.Builder
	media.WriteString("#EXTM3U\n#EXT-X-MEDIA-SEQUENCE:5\n#EXT-X-KEY:METHOD=AES-128,URI=\"key.bin\"\n")
	for i := range 25 {
		plain := bytes.Repeat([]byte{byte(i)}, 1000+i)
		want.Write(plain)
		name := fmt.Sprintf("seg%d.ts", i)
		segs["/hd/"+name] = encrypt(plain, key, int64(5+i))
		fmt.Fprintf(&media, "#EXTINF:4.0,\n%s\n", name)
	}
	media.WriteString("#EXT-X-ENDLIST\n")
	master := "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=100000,RESOLUTION=640x360\nsd/index.m3u8\n" +
		"#EXT-X-STREAM-INF:BANDWIDTH=900000,RESOLUTION=1920x1080\nhd/index.m3u8\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/master.m3u8":
			w.Write([]byte(master))
		case "/hd/index.m3u8":
			w.Write([]byte(media.String()))
		case "/hd/key.bin":
			w.Write(key)
		default:
			if b, ok := segs[r.URL.Path]; ok {
				w.Write(b)
				return
			}
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	out := filepath.Join(dir, "video.ts")
	d := New(Config{
		Request:   httpdl.Request{URL: srv.URL + "/master.m3u8"},
		Path:      out,
		StatePath: filepath.Join(dir, "state.json"),
		Client:    srv.Client(),
		Retries:   1,
	})
	if err := d.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want.Bytes()) {
		t.Fatalf("output differs: %d vs %d bytes", len(got), want.Len())
	}
}

func TestAttrs(t *testing.T) {
	a := attrs(`METHOD=AES-128,URI="https://x/k?a=1,b=2",IV=0x01`)
	if a["URI"] != "https://x/k?a=1,b=2" || a["IV"] != "0x01" || a["METHOD"] != "AES-128" {
		t.Fatalf("attrs = %v", a)
	}
}
