package httpdl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/godownload/godownload/internal/fileutil"
)

// Info is what a server says about a file before downloading it.
type Info struct {
	// URL is the address after redirects.
	URL string
	// Size is -1 when the server doesn't say.
	Size int64
	// Ranges is true when the server can send parts of the file, which
	// multi-connection downloading, resuming and seeking all need.
	Ranges       bool
	Name         string
	ContentType  string
	ETag         string
	LastModified string
}

// Probe asks the server about a file with a one-byte range request, which
// servers answer more faithfully than HEAD.
func Probe(ctx context.Context, client *http.Client, r Request) (Info, error) {
	req, err := r.Build(ctx)
	if err != nil {
		return Info{}, err
	}
	req.Header.Set("Range", "bytes=0-0")
	resp, err := client.Do(req)
	if err != nil {
		return Info{}, err
	}
	defer resp.Body.Close()
	_, _ = io.CopyN(io.Discard, resp.Body, 1)

	info := Info{
		URL:          resp.Request.URL.String(),
		Size:         -1,
		ContentType:  resp.Header.Get("Content-Type"),
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
	}
	switch resp.StatusCode {
	case http.StatusPartialContent:
		size, err := totalFromContentRange(resp.Header.Get("Content-Range"))
		if err != nil {
			return Info{}, err
		}
		info.Size = size
		info.Ranges = size > 0
	case http.StatusOK:
		info.Size = resp.ContentLength
	case http.StatusRequestedRangeNotSatisfiable:
		// An empty file: there is no byte 0.
		info.Size = 0
	default:
		return Info{}, &StatusError{Code: resp.StatusCode}
	}
	info.Name = fileutil.WithExtension(
		fileutil.NameFromResponse(resp.Header.Get("Content-Disposition"), info.URL),
		info.ContentType,
	)
	return info, nil
}

// totalFromContentRange reads the total from "bytes 0-0/12345"; "*" is unknown (-1).
func totalFromContentRange(h string) (int64, error) {
	i := strings.LastIndexByte(h, '/')
	if i < 0 {
		return 0, fmt.Errorf("bad Content-Range %q", h)
	}
	total := h[i+1:]
	if total == "*" {
		return -1, nil
	}
	return strconv.ParseInt(total, 10, 64)
}

// startFromContentRange reads the first byte from "bytes 100-199/1000".
func startFromContentRange(h string) (int64, error) {
	h = strings.TrimPrefix(strings.TrimSpace(h), "bytes ")
	dash := strings.IndexByte(h, '-')
	if dash < 0 {
		return 0, fmt.Errorf("bad Content-Range %q", h)
	}
	return strconv.ParseInt(h[:dash], 10, 64)
}
