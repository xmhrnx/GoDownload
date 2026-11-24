package manager

import (
	"context"
	"fmt"

	"github.com/godownload/godownload/internal/httpdl"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/internal/ratelimit"
)

// httpDownloaderLocked returns e's downloader, creating it the first time.
// It outlives runs so a player can keep reading a paused download.
func (m *Manager) httpDownloaderLocked(e *entry) *httpdl.Downloader {
	if e.http != nil {
		return e.http
	}
	conns := e.d.MaxConnections
	if conns <= 0 {
		conns = m.settings.Connections
	}
	id := e.d.ID
	e.http = httpdl.New(httpdl.Config{
		Request:      httpdl.Request{URL: e.d.URL, Headers: e.d.Request},
		Path:         e.path(),
		StatePath:    m.store.StatePath(id),
		Connections:  conns,
		Retries:      m.settings.Retries,
		Client:       m.client,
		Limiters:     []*ratelimit.Limiter{e.limiter, m.down},
		StreamWindow: int64(max(m.settings.StreamBufferMB, 4)) << 20,
		Log:          func(format string, args ...any) { m.logf(id, format, args...) },
	})
	return e.http
}

func (m *Manager) runHTTP(ctx context.Context, e *entry) error {
	m.mu.Lock()
	dl := m.httpDownloaderLocked(e)
	m.mu.Unlock()

	if err := dl.Run(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	info := dl.Info()
	if info.Size >= 0 {
		e.d.Size = info.Size
	}
	sum := e.d.Checksum
	m.mu.Unlock()

	if sum != "" {
		if err := m.verifyChecksum(ctx, e, sum); err != nil {
			return err
		}
	}
	m.mu.Lock()
	m.completeLocked(e, model.StatusCompleted)
	m.mu.Unlock()
	return nil
}

// verifyChecksum hashes the finished file and compares it with the expected sum.
func (m *Manager) verifyChecksum(ctx context.Context, e *entry, sum string) error {
	m.mu.Lock()
	e.d.Status = model.StatusVerifying
	path := e.path()
	m.mu.Unlock()
	m.changed()
	ok, got, err := checkSum(ctx, path, sum, func(done, total int64) {
		m.mu.Lock()
		e.d.Done, e.d.Size = done, total
		m.mu.Unlock()
	})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("checksum mismatch: expected %s, got %s. Use Check & repair", sum, got)
	}
	m.logf(e.d.ID, "Checksum verified")
	return nil
}

func (m *Manager) logf(id, format string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.entries[id]; ok {
		e.logf(format, args...)
	}
}
