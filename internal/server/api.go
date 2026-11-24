package server

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// Link is one download the browser sends.
type Link struct {
	URL       string `json:"url"`
	Filename  string `json:"filename"`
	Referrer  string `json:"referrer"`
	Cookies   string `json:"cookies"`
	UserAgent string `json:"userAgent"`
	MIME      string `json:"mime"`
	Size      int64  `json:"size"`
	PageURL   string `json:"pageUrl"`
	PageTitle string `json:"pageTitle"`
	// Kind is file, media, hls, dash, torrent or magnet.
	Kind   string `json:"kind"`
	Silent bool   `json:"silent"`
}

type batch struct {
	Items     []Link `json:"items"`
	PageURL   string `json:"pageUrl"`
	PageTitle string `json:"pageTitle"`
}

const maxBody = 4 << 20

// api serves the extension. Only extension pages may call it: a web page's
// fetch always carries its own Origin, which is refused, and the custom
// header forces browsers to ask first (a preflight) for cross-origin calls.
func (s *Server) api(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if !allowedOrigin(origin) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	}
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-GoDownload-Client")
		w.Header().Set("Access-Control-Max-Age", "600")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !strings.HasPrefix(r.Header.Get("X-GoDownload-Client"), "extension") {
		http.Error(w, "missing client header", http.StatusForbidden)
		return
	}
	if tok := s.pairingToken(); tok != "" {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), []byte(tok)) != 1 {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
	}
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/ping":
		intercept := s.cfg.Intercept == nil || s.cfg.Intercept()
		writeJSON(w, http.StatusOK, map[string]any{"app": "GoDownload", "version": Version, "interceptEnabled": intercept})
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/downloads":
		var l Link
		if !readJSON(w, r, &l) {
			return
		}
		if !validLink(l.URL) {
			http.Error(w, "unsupported link", http.StatusBadRequest)
			return
		}
		s.cfg.OnAdd([]Link{l}, false)
		writeJSON(w, http.StatusAccepted, map[string]string{"id": ""})
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/downloads/batch":
		var b batch
		if !readJSON(w, r, &b) {
			return
		}
		var items []Link
		for _, l := range b.Items {
			if validLink(l.URL) {
				if l.PageURL == "" {
					l.PageURL, l.PageTitle = b.PageURL, b.PageTitle
				}
				items = append(items, l)
			}
		}
		if len(items) == 0 {
			http.Error(w, "no usable links", http.StatusBadRequest)
			return
		}
		s.cfg.OnAdd(items, true)
		writeJSON(w, http.StatusAccepted, map[string][]string{"ids": {}})
	default:
		http.NotFound(w, r)
	}
}

func allowedOrigin(o string) bool {
	return o == "" || o == "null" ||
		strings.HasPrefix(o, "chrome-extension://") ||
		strings.HasPrefix(o, "moz-extension://") ||
		strings.HasPrefix(o, "safari-web-extension://")
}

func validLink(u string) bool {
	l := strings.ToLower(u)
	return strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://") || strings.HasPrefix(l, "magnet:")
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody)).Decode(v); err != nil {
		http.Error(w, "bad JSON: "+err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
