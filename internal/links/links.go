// Package links finds downloadable links in pasted text.
package links

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/godownload/godownload/internal/bt"
)

var urlRE = regexp.MustCompile(`(?i)\b(?:https?|ftp)://[^\s<>"'` + "`" + `]+|magnet:\?[^\s<>"']+`)

// Find returns the http(s) and magnet links and bare info hashes in text,
// in order and without duplicates.
func Find(text string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = strings.TrimRight(s, ".,;:)]}>'\"")
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	for _, m := range urlRE.FindAllString(text, -1) {
		if strings.HasPrefix(strings.ToLower(m), "magnet:") {
			add(m)
			continue
		}
		if u, err := url.Parse(strings.TrimRight(m, ".,;:)]}>'\"")); err == nil && u.Host != "" {
			add(m)
		}
	}
	for _, f := range strings.Fields(text) {
		if _, ok := bt.ParseSource(f); ok && !strings.Contains(f, ":") {
			add(f)
		}
	}
	return out
}

// IsTorrent reports whether a link is a magnet link or info hash.
func IsTorrent(link string) bool {
	_, ok := bt.ParseSource(link)
	return ok
}
