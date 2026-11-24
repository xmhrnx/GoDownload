package server

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/godownload/godownload/internal/manager"
)

// StreamURL is the address a player opens to watch file of a download while
// it downloads. Players seek with ordinary HTTP range requests; a seek to a
// part not yet downloaded makes that part the most urgent, and the read waits
// for it instead of failing, so players don't crash or show corrupt frames.
func (s *Server) StreamURL(id string, file int, name string) string {
	return fmt.Sprintf("http://127.0.0.1:%d/stream/%s/%s/%d/%s", s.Port(), s.token, id, file, url.PathEscape(path.Base(name)))
}

// StatusURL is the JSON status of a stream, for players that show what's buffered.
func (s *Server) StatusURL(id string, file int) string {
	return fmt.Sprintf("http://127.0.0.1:%d/status/%s/%s/%d", s.Port(), s.token, id, file)
}

func (s *Server) checkToken(r *http.Request) bool {
	return subtle.ConstantTimeCompare([]byte(r.PathValue("token")), []byte(s.token)) == 1
}

func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	if !s.checkToken(r) {
		http.NotFound(w, r)
		return
	}
	file, _ := strconv.Atoi(r.PathValue("file"))
	id := r.PathValue("id")
	media, err := s.m.Media(r.Context(), id, file)
	if err != nil {
		code := http.StatusServiceUnavailable
		if errors.Is(err, manager.ErrNotFound) {
			code = http.StatusNotFound
		}
		http.Error(w, err.Error(), code)
		return
	}
	rs, err := media.Open(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer rs.Close()
	if ct := mime.TypeByExtension(path.Ext(media.Name)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Cache-Control", "no-store")
	// Players that know GoDownload read this to show what's downloaded.
	w.Header().Set("X-GoDownload-Status", s.StatusURL(id, file))
	http.ServeContent(w, r, media.Name, time.Time{}, rs)
}

// statusBody is documented in docs/STREAMING.md for player authors.
type statusBody struct {
	Name      string     `json:"name"`
	Size      int64      `json:"size"`
	Done      int64      `json:"done"`
	Complete  bool       `json:"complete"`
	Speed     int64      `json:"speed"`
	Status    string     `json:"status"`
	Available [][2]int64 `json:"available"`
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	if !s.checkToken(r) {
		http.NotFound(w, r)
		return
	}
	it, _, err := s.m.Detail(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	body := statusBody{
		Name: it.Name, Size: it.Size, Done: it.Done, Speed: it.DownSpeed,
		Status: string(it.Status), Complete: it.Status.Finished(),
	}
	body.Available = ranges(it.Blocks, it.Size)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	writeJSON(w, http.StatusOK, body)
}

// ranges turns the coarse block map into byte ranges that are fully downloaded.
func ranges(blocks []float32, size int64) [][2]int64 {
	out := [][2]int64{}
	if len(blocks) == 0 || size <= 0 {
		return out
	}
	n := int64(len(blocks))
	for i := int64(0); i < n; i++ {
		if blocks[i] < 1 {
			continue
		}
		start, end := i*size/n, (i+1)*size/n
		if k := len(out) - 1; k >= 0 && out[k][1] == start {
			out[k][1] = end
		} else {
			out = append(out, [2]int64{start, end})
		}
	}
	return out
}
