package manager

import (
	"context"
	"time"

	"github.com/godownload/godownload/internal/bt"
	"github.com/godownload/godownload/internal/categorize"
	"github.com/godownload/godownload/internal/model"
)

func torrentSource(d model.Download) bt.Source {
	if d.Torrent.Magnet != "" {
		return bt.Source{Magnet: d.Torrent.Magnet}
	}
	return bt.Source{InfoHash: d.Torrent.InfoHash}
}

// openTorrentLocked opens e's torrent in the engine if it isn't open.
func (m *Manager) openTorrentLocked(e *entry) (*bt.Handle, error) {
	if e.torrent != nil {
		return e.torrent, nil
	}
	eng, err := m.torrentEngine()
	if err != nil {
		return nil, err
	}
	h, err := eng.Open(torrentSource(e.d), e.d.Dir)
	if err != nil {
		return nil, err
	}
	e.torrent = h
	return h, nil
}

func (m *Manager) runTorrent(ctx context.Context, e *entry) error {
	m.mu.Lock()
	h, err := m.openTorrentLocked(e)
	verify := e.verify
	e.verify = false
	m.mu.Unlock()
	if err != nil {
		return err
	}
	defer func() {
		m.mu.Lock()
		if e.streams == 0 {
			h.Close()
			e.torrent = nil
		}
		m.mu.Unlock()
	}()

	m.logf(e.d.ID, "Looking for peers")
	if err := h.WaitInfo(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	m.adoptMetaLocked(e, h)
	opts := *e.d.Torrent
	m.mu.Unlock()

	if verify {
		if err := m.verifyTorrent(ctx, e, h); err != nil {
			return err
		}
	}
	h.Start(opts.Files, true)
	h.SetUploadLimit(opts.UploadLimit)
	m.setStatus(e, model.StatusDownloading)

	if err := waitComplete(ctx, h); err != nil {
		return err
	}
	_, wanted := h.Progress()
	m.mu.Lock()
	e.d.Size = wanted
	seed := e.d.Torrent.Seed
	if seed {
		if e.d.Torrent.SeedingSince.IsZero() {
			e.d.Torrent.SeedingSince = now()
		}
		m.completeLocked(e, model.StatusSeeding)
	} else {
		m.completeLocked(e, model.StatusCompleted)
	}
	m.mu.Unlock()
	if !seed {
		return nil
	}
	h.SetSeed(true)
	<-ctx.Done() // seeding ends on pause or when the tick sees the ratio or time reached
	return nil
}

// adoptMetaLocked fills in what the metadata tells: name, size, category, files.
func (m *Manager) adoptMetaLocked(e *entry, h *bt.Handle) {
	files := h.Files()
	if e.d.Name == "" || e.d.Name == e.d.Torrent.InfoHash {
		e.d.Name = h.Name()
	}
	if len(e.d.Torrent.Files) != len(files) {
		e.d.Torrent.Files = make([]model.FilePriority, len(files))
		for i := range files {
			e.d.Torrent.Files[i] = model.PriorityNormal
		}
	}
	if e.d.Category == "" || e.d.Category == model.CategoryOther {
		names := make([]string, len(files))
		sizes := make([]int64, len(files))
		for i, f := range files {
			names[i], sizes[i] = f.Path, f.Size
		}
		e.d.Category = categorize.Largest(names, sizes)
	}
	m.dirty = true
}

func (m *Manager) verifyTorrent(ctx context.Context, e *entry, h *bt.Handle) error {
	m.setStatus(e, model.StatusVerifying)
	before, _ := h.Progress()
	if err := h.Verify(ctx); err != nil {
		return err
	}
	after, wanted := h.Progress()
	msg := "All data is intact."
	if after < before {
		msg = "Damaged pieces found; downloading them again."
	} else if after < wanted {
		msg = "The data on disk is intact; the rest will download now."
	}
	m.logf(e.d.ID, "Check: %s", msg)
	m.emit(Event{Kind: EventInfo, ID: e.d.ID, Title: e.d.Name, Message: msg})
	return nil
}

func waitComplete(ctx context.Context, h *bt.Handle) error {
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for {
		if done, wanted := h.Progress(); wanted >= 0 && done >= wanted {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
}

func (m *Manager) setStatus(e *entry, s model.Status) {
	m.mu.Lock()
	e.d.Status = s
	m.mu.Unlock()
	m.changed()
}
