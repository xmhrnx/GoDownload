package manager

import (
	"time"

	"github.com/godownload/godownload/internal/model"
)

// runSchedulesLocked opens and closes schedule windows. Opening one queues
// the downloads waiting for a schedule; closing it pauses them again and, if
// set, runs its power action. A window whose downloads all finish runs the
// action at once.
func (m *Manager) runSchedulesLocked(at time.Time) {
	limit := m.settings.DownloadLimit
	for _, s := range m.settings.Schedules {
		open := s.ActiveAt(at)
		was := m.activeWindow[s.ID]
		switch {
		case open && !was:
			m.activeWindow[s.ID] = true
			m.windowFired[s.ID] = false
			m.startScheduledLocked()
		case !open && was:
			if !m.windowFired[s.ID] && s.WhenDone != model.PowerNone {
				go m.emit(Event{Kind: EventAllDone, Title: s.Name + " ended", Action: s.WhenDone})
			}
			delete(m.activeWindow, s.ID)
			delete(m.windowFired, s.ID)
			m.stopScheduledLocked()
		case open && !m.windowFired[s.ID] && s.WhenDone != model.PowerNone && m.scheduledDoneLocked():
			// Everything this window was for has finished: act now, once.
			m.windowFired[s.ID] = true
			go m.emit(Event{Kind: EventAllDone, Title: s.Name + ": all downloads finished", Action: s.WhenDone})
		}
		if m.activeWindow[s.ID] && s.SpeedLimit > 0 {
			limit = s.SpeedLimit
		}
	}
	m.down.Set(limit)
	m.scheduleLocked()
}

func (m *Manager) startScheduledLocked() {
	for _, e := range m.entries {
		if e.d.Scheduled && !e.d.Status.Finished() {
			e.d.Stopped = false
			if !e.running() {
				e.d.Status = model.StatusQueued
			}
		}
	}
	m.dirty = true
}

func (m *Manager) stopScheduledLocked() {
	for _, e := range m.entries {
		if e.d.Scheduled && !e.d.Status.Finished() {
			e.d.Stopped = true
			if e.running() {
				e.stop()
			} else {
				e.d.Status = model.StatusPaused
			}
		}
	}
	m.dirty = true
}

func (m *Manager) scheduledDoneLocked() bool {
	found := false
	for _, e := range m.entries {
		if !e.d.Scheduled {
			continue
		}
		found = true
		if !e.d.Status.Finished() && e.d.Status != model.StatusFailed {
			return false
		}
	}
	return found
}

// SetScheduled moves downloads into (or out of) the scheduled queue.
func (m *Manager) SetScheduled(on bool, ids ...string) {
	m.mu.Lock()
	for _, id := range ids {
		e, err := m.get(id)
		if err != nil || e.d.Status.Finished() {
			continue
		}
		e.d.Scheduled = on
		if on && !m.anyWindowOpenLocked() {
			e.d.Stopped = true
			if e.running() {
				e.stop()
			} else {
				e.d.Status = model.StatusPaused
			}
		}
	}
	m.dirty = true
	m.scheduleLocked()
	m.mu.Unlock()
	m.changed()
}

func (m *Manager) anyWindowOpenLocked() bool {
	return len(m.activeWindow) > 0
}
