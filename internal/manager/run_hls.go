package manager

import (
	"context"

	"github.com/godownload/godownload/internal/hls"
	"github.com/godownload/godownload/internal/httpdl"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/internal/ratelimit"
)

func (m *Manager) runHLS(ctx context.Context, e *entry) error {
	m.mu.Lock()
	if e.hls == nil {
		conns := e.d.MaxConnections
		if conns <= 0 {
			conns = m.settings.Connections
		}
		e.hls = hls.New(hls.Config{
			Request:     httpdl.Request{URL: e.d.URL, Headers: e.d.Request},
			Path:        e.path(),
			StatePath:   m.store.StatePath(e.d.ID),
			Connections: conns,
			Retries:     m.settings.Retries,
			Client:      m.client,
			Limiters:    []*ratelimit.Limiter{e.limiter, m.down},
		})
	}
	dl := e.hls
	e.d.Status = model.StatusDownloading
	m.mu.Unlock()

	if err := dl.Run(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	_, _, written := dl.Progress()
	e.d.Size = written
	m.completeLocked(e, model.StatusCompleted)
	e.hls = nil
	m.mu.Unlock()
	return nil
}
