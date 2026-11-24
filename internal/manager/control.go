package manager

import (
	"context"
	"errors"
	"slices"

	"github.com/godownload/godownload/internal/httpdl"
	"github.com/godownload/godownload/internal/model"
)

// Pause stops downloads; they stay paused until resumed. Pausing a seeding
// torrent stops seeding.
func (m *Manager) Pause(ids ...string) {
	m.mu.Lock()
	for _, id := range ids {
		e, err := m.get(id)
		if err != nil {
			continue
		}
		e.d.Stopped = true
		if e.running() {
			e.stop()
		} else if !e.d.Status.Finished() {
			e.d.Status = model.StatusPaused
		}
	}
	m.dirty = true
	m.mu.Unlock()
	m.changed()
}

// Resume queues downloads again. A finished torrent with seeding on goes
// back to seeding.
func (m *Manager) Resume(ids ...string) {
	m.mu.Lock()
	for _, id := range ids {
		e, err := m.get(id)
		if err != nil || e.running() || e.busy {
			continue
		}
		e.d.Stopped = false
		switch {
		case e.d.Status == model.StatusCompleted && e.d.Kind == model.KindTorrent && e.d.Torrent.Seed:
			e.seedDone = false
			e.d.Status = model.StatusQueued
		case !e.d.Status.Finished():
			e.d.Status = model.StatusQueued
			e.d.Error = ""
		}
	}
	m.dirty = true
	m.scheduleLocked()
	m.mu.Unlock()
	m.changed()
}

// PauseAll pauses every unfinished download.
func (m *Manager) PauseAll() {
	m.Pause(m.ids(func(e *entry) bool { return !e.d.Status.Finished() })...)
}

// ResumeAll resumes every paused or failed download.
func (m *Manager) ResumeAll() {
	m.Resume(m.ids(func(e *entry) bool {
		return e.d.Status == model.StatusPaused || e.d.Status == model.StatusFailed
	})...)
}

func (m *Manager) ids(match func(*entry) bool) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var ids []string
	for id, e := range m.entries {
		if match(e) {
			ids = append(ids, id)
		}
	}
	return ids
}

// maxActiveLocked is how many downloads run at once, a schedule's override included.
func (m *Manager) maxActiveLocked() int {
	limit := max(1, m.settings.MaxActive)
	for _, s := range m.settings.Schedules {
		if m.activeWindow[s.ID] && s.MaxActive > 0 {
			limit = s.MaxActive
		}
	}
	return limit
}

// scheduleLocked starts queued downloads, oldest first, while there are free
// slots. Streams and seeding torrents don't take a slot.
func (m *Manager) scheduleLocked() {
	if m.ctx.Err() != nil {
		return
	}
	active := 0
	var queued []*entry
	for _, e := range m.entries {
		switch {
		case e.running() && e.d.Status.Active() && e.streams == 0:
			active++
		case !e.running() && !e.busy && e.d.Status == model.StatusQueued && !e.d.Stopped:
			queued = append(queued, e)
		}
	}
	slices.SortFunc(queued, func(a, b *entry) int { return a.d.AddedAt.Compare(b.d.AddedAt) })
	for _, e := range queued {
		if active >= m.maxActiveLocked() {
			return
		}
		m.startLocked(e)
		active++
	}
}

// startLocked runs e's engine on a goroutine.
func (m *Manager) startLocked(e *entry) {
	if e.running() {
		return
	}
	ctx, cancel := context.WithCancel(m.ctx)
	e.cancel = cancel
	e.done = make(chan struct{})
	e.d.Status = model.StatusConnecting
	e.d.Error = ""
	e.seedDone = false
	done := e.done
	e.logf("Starting")
	go func() {
		err := m.run(ctx, e)
		m.finished(e, err, ctx.Err() != nil)
		close(done)
	}()
}

func (m *Manager) run(ctx context.Context, e *entry) error {
	m.mu.Lock()
	kind := e.d.Kind
	m.mu.Unlock()
	switch kind {
	case model.KindTorrent:
		return m.runTorrent(ctx, e)
	case model.KindHLS:
		return m.runHLS(ctx, e)
	default:
		return m.runHTTP(ctx, e)
	}
}

// finished records how a run ended and starts the next queued download.
func (m *Manager) finished(e *entry, err error, cancelled bool) {
	m.mu.Lock()
	e.cancel, e.done = nil, nil
	e.stats.DownSpeed, e.stats.UpSpeed, e.stats.Connections, e.stats.ETA = 0, 0, 0, -1
	e.stats.Segments = nil
	var ev *Event
	switch {
	case e.d.Status == model.StatusSeeding && m.ctx.Err() != nil:
		e.d.Status = model.StatusQueued // seed again next time
	case e.d.Status == model.StatusSeeding:
		e.d.Status = model.StatusCompleted
		e.logf("Stopped seeding")
	case e.d.Status == model.StatusCompleted:
		// The runner finished it.
	case cancelled && m.ctx.Err() != nil:
		e.d.Status = model.StatusQueued // the app is quitting: resume next time
	case cancelled && e.d.Stopped:
		e.d.Status = model.StatusPaused
		e.logf("Paused")
	case cancelled:
		e.d.Status = model.StatusQueued // a schedule or the queue took the slot back
	case err != nil:
		e.d.Status = model.StatusFailed
		e.d.Error = friendlyError(err)
		e.logf("Failed: %v", err)
		var se *httpdl.StatusError
		ev = &Event{Kind: EventFailed, ID: e.d.ID, Title: e.d.Name, Message: e.d.Error,
			Expired: errors.As(err, &se) && se.Expired()}
	}
	m.dirty = true
	m.scheduleLocked()
	m.mu.Unlock()
	if ev != nil {
		m.emit(*ev)
	}
	m.changed()
}

// completeLocked marks e finished and tells the user.
func (m *Manager) completeLocked(e *entry, status model.Status) {
	e.d.Status = status
	e.d.Done = e.d.Size
	e.d.Error = ""
	if e.d.CompletedAt.IsZero() {
		e.d.CompletedAt = now()
	}
	e.logf("Completed")
	m.dirty = true
	ev := Event{Kind: EventCompleted, ID: e.d.ID, Title: e.d.Name, Message: "Download complete"}
	go m.emit(ev)
}
