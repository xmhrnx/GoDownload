package httpdl

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/godownload/godownload/internal/model"
)

// DefaultUserAgent is sent when the browser didn't say which it uses.
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0 Safari/537.36 GoDownload/1.0"

// NewClient returns an HTTP client for downloads. It speaks HTTP/1.1 only:
// HTTP/2 would multiplex every connection of a download onto one TCP stream,
// which defeats per-connection server throttling, the point of splitting.
func NewClient(proxy string) (*http.Client, error) {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   20 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   20 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       60 * time.Second,
		MaxIdleConnsPerHost:   32,
		TLSNextProto:          map[string]func(string, *tls.Conn) http.RoundTripper{},
		DisableCompression:    true,
	}
	if proxy != "" {
		u, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("proxy: %w", err)
		}
		tr.Proxy = http.ProxyURL(u)
	}
	return &http.Client{Transport: tr}, nil
}

// Request is what to fetch and how the browser would have fetched it.
type Request struct {
	URL     string
	Headers model.RequestHeaders
}

// Build makes the GET request with the browser's headers.
func (r Request) Build(ctx context.Context) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.URL, nil)
	if err != nil {
		return nil, err
	}
	ua := r.Headers.UserAgent
	if ua == "" {
		ua = DefaultUserAgent
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "identity")
	if r.Headers.Referrer != "" {
		req.Header.Set("Referer", r.Headers.Referrer)
	}
	if r.Headers.Cookies != "" {
		req.Header.Set("Cookie", r.Headers.Cookies)
	}
	return req, nil
}

// StatusError is a response the server refused with.
type StatusError struct {
	Code int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("server answered %d %s", e.Code, http.StatusText(e.Code))
}

// Expired reports whether the link has likely expired, so a fresh one from the
// page would work (the "refresh link" case).
func (e *StatusError) Expired() bool {
	return e.Code == http.StatusForbidden || e.Code == http.StatusGone ||
		e.Code == http.StatusNotFound || e.Code == http.StatusUnauthorized
}

// Retryable reports whether trying again later may succeed.
func (e *StatusError) Retryable() bool {
	return e.Code == http.StatusTooManyRequests || e.Code == http.StatusRequestTimeout || e.Code >= 500
}
