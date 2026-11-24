// Package fileutil names and places downloaded files.
package fileutil

import (
	"errors"
	"io/fs"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

// Sanitize makes name safe as a file name on every desktop OS.
func Sanitize(name string) string {
	name = strings.Map(func(r rune) rune {
		switch {
		case r < 32, strings.ContainsRune(`<>:"/\|?*`, r):
			return '_'
		case unicode.IsControl(r):
			return -1
		}
		return r
	}, name)
	name = strings.Trim(strings.TrimSpace(name), ".")
	base := strings.ToUpper(strings.TrimSuffix(name, filepath.Ext(name)))
	switch base {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "LPT1", "LPT2", "LPT3":
		name = "_" + name
	}
	if len(name) > 200 {
		ext := filepath.Ext(name)
		if len(ext) > 20 {
			ext = ""
		}
		name = name[:200-len(ext)] + ext
	}
	if name == "" {
		return "download"
	}
	return name
}

// NameFromResponse picks a file name from a Content-Disposition header, falling
// back to the last path segment of the URL.
func NameFromResponse(contentDisposition, rawURL string) string {
	if contentDisposition != "" {
		if _, params, err := mime.ParseMediaType(contentDisposition); err == nil {
			if n := params["filename"]; n != "" {
				return Sanitize(path.Base(strings.ReplaceAll(n, `\`, "/")))
			}
		}
	}
	return NameFromURL(rawURL)
}

// NameFromURL is the URL's last path segment, unescaped.
func NameFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "download"
	}
	p := u.Path
	if unescaped, err := url.PathUnescape(p); err == nil {
		p = unescaped
	}
	base := path.Base(p)
	if base == "/" || base == "." || base == "" {
		base = u.Hostname()
	}
	return Sanitize(base)
}

// WithExtension adds the extension a MIME type implies when name has none.
func WithExtension(name, contentType string) string {
	if filepath.Ext(name) != "" || contentType == "" {
		return name
	}
	media, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return name
	}
	if exts, _ := mime.ExtensionsByType(media); len(exts) > 0 {
		return name + exts[0]
	}
	return name
}

// UniquePath returns dir/name, or dir/"name (2).ext" and so on when taken,
// either by a file on disk or by a path in reserved.
func UniquePath(dir, name string, reserved func(string) bool) string {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		candidate := name
		if i > 1 {
			candidate = stem + " (" + strconv.Itoa(i) + ")" + ext
		}
		p := filepath.Join(dir, candidate)
		if reserved != nil && reserved(p) {
			continue
		}
		if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
			if _, err := os.Stat(p + PartSuffix); errors.Is(err, fs.ErrNotExist) {
				return p
			}
		}
	}
}

// PartSuffix marks a file that is still downloading.
const PartSuffix = ".gdpart"
