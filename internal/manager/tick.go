package manager

import (
	"time"

	"github.com/godownload/godownload/internal/model"
)

// loop does the once-a-second housekeeping: speeds, progress, seeding
// limits, schedules, the "when done" action, and saving.
func (m *Manager) loop() {
	defer m.wg.Done()
	t := time.NewTicker(time.Second)
	defer t.Stop()
	lastSave := now()
	lastMinute := -1
	for {
		select {
		case <-m.ctx.Done():
			return
		case at := <-t.C:
			m.mu.Lock()
			anyRunning := m.sampleLocked(at)
			if minute := at.Hour()*60 + at.Minute(); minute != lastMinute {
				lastMinute = minute
				m.runSchedulesLocked(at)
			}
			ev := m.checkAllDoneLocked()
			if m.dirty && at.Sub(lastSave) > 5*time.Second {
				lastSave = at
				if err := m.saveLocked(); err != nil {
					ev = &Event{Kind: EventInfo, Title: "Couldn't save the download list", Message: err.Error()}
				}
			}
			m.mu.Unlock()
			if ev != nil {
				m.emit(*ev)
			}
			if anyRunning {
				m.changed()
			}
		}
	}
}

// ema smooths speeds so the numbers don't jitter.
func ema(prev float64, v float64) float64 {
	if prev == 0 {
		return v
	}
	return prev*0.6 + v*0.4
}

// sampleLocked refreshes every running download's stats; it reports whether any runs.
func (m *Manager) sampleLocked(at time.Time) bool {
	any := false
	for _, e := range m.entries {
		if !e.running() {
			continue
		}
		any = true
		dt := at.Sub(e.sample.at).Seconds()
		if e.sample.at.IsZero() || dt <= 0 {
			dt = 1
		}
		e.sample.at = at
		switch {
		case e.http != nil:
			m.sampleHTTPLocked(e, dt)
		case e.hls != nil:
			m.sampleHLSLocked(e, dt)
		case e.torrent != nil:
			m.sampleTorrentLocked(e, dt)
		}
		e.stats.ETA = -1
		if e.stats.DownSpeed > 0 && e.d.Size > 0 {
			e.stats.ETA = (e.d.Size - e.d.Done) / e.stats.DownSpeed
		}
		e.stats.Streaming = e.streams > 0
	}
	return any
}

func (m *Manager) sampleHTTPLocked(e *entry, dt float64) {
	dl := e.http
	recv := dl.Received()
	e.sample.downEMA = ema(e.sample.downEMA, float64(recv-e.sample.down)/dt)
	e.sample.down = recv
	e.stats.DownSpeed = int64(e.sample.downEMA)
	done, size := dl.Progress()
	if size >= 0 {
		e.d.Size = size
	}
	if e.d.Status != model.StatusVerifying && e.d.Status != model.StatusCompleted {
		e.d.Done = done
	}
	e.stats.Connections = dl.Connections()
	blocks, segs := dl.Snapshot(120)
	e.stats.Blocks = blocks
	e.stats.Segments = e.stats.Segments[:0]
	for _, s := range segs {
		e.stats.Segments = append(e.stats.Segments, model.Segment{Start: s[0], Pos: s[1], End: s[2]})
	}
	if info := dl.Info(); info.URL != "" {
		e.d.Resumable = info.Ranges
	}
	if e.d.Status == model.StatusConnecting && e.stats.Connections > 0 && recv > 0 {
		e.d.Status = model.StatusDownloading
	}
}

func (m *Manager) sampleHLSLocked(e *entry, dt float64) {
	recv := e.hls.Received()
	e.sample.downEMA = ema(e.sample.downEMA, float64(recv-e.sample.down)/dt)
	e.sample.down = recv
	e.stats.DownSpeed = int64(e.sample.downEMA)
	written, total, bytes := e.hls.Progress()
	e.d.Done = bytes
	if written > 0 && total > 0 {
		e.d.Size = bytes * int64(total) / int64(written)
	}
	if total > 0 {
		e.stats.Blocks = []float32{float32(written) / float32(total)}
	}
}

func (m *Manager) sampleTorrentLocked(e *entry, dt float64) {
	h := e.torrent
	c := h.Counters()
	if c.Downloaded < e.sample.down || c.Uploaded < e.sample.uploadSeen {
		// A new session (the torrent was reopened): counters restarted.
		e.sample.down, e.sample.uploadSeen = 0, 0
	}
	e.sample.downEMA = ema(e.sample.downEMA, float64(c.Downloaded-e.sample.down)/dt)
	e.sample.upEMA = ema(e.sample.upEMA, float64(c.Uploaded-e.sample.uploadSeen)/dt)
	e.d.Torrent.Uploaded += c.Uploaded - e.sample.uploadSeen
	e.sample.down, e.sample.uploadSeen = c.Downloaded, c.Uploaded
	e.stats.DownSpeed = int64(e.sample.downEMA)
	e.stats.UpSpeed = int64(e.sample.upEMA)
	e.stats.Connections, e.stats.Peers, e.stats.Seeders = c.Peers, c.Peers, c.Seeders
	if done, wanted := h.Progress(); wanted >= 0 {
		e.d.Done, e.d.Size = done, wanted
	}
	e.stats.Blocks = h.Coarse(120)
	if e.d.Status == model.StatusSeeding && seedLimitReached(e.d) {
		e.logf("Seeding goal reached")
		e.stop()
	}
}

func seedLimitReached(d model.Download) bool {
	t := d.Torrent
	if t.SeedRatio > 0 && d.Size > 0 && float64(t.Uploaded)/float64(d.Size) >= t.SeedRatio {
		return true
	}
	return t.SeedMinutes > 0 && !t.SeedingSince.IsZero() &&
		now().Sub(t.SeedingSince) >= time.Duration(t.SeedMinutes)*time.Minute
}

// checkAllDoneLocked returns the "when done" event the moment the last queued
// or running download finishes.
func (m *Manager) checkAllDoneLocked() *Event {
	work := false
	for _, e := range m.entries {
		if e.busy || (e.running() && e.d.Status.Active()) || (e.d.Status == model.StatusQueued && !e.d.Stopped) {
			work = true
			break
		}
	}
	if work {
		m.hadWork = true
		return nil
	}
	if !m.hadWork {
		return nil
	}
	m.hadWork = false
	action := m.settings.WhenDone
	if action == model.PowerNone {
		return nil
	}
	// One-shot, like "shut down when done" in other download managers.
	m.settings.WhenDone = model.PowerNone
	if err := m.store.SaveSettings(m.settings); err != nil {
		return &Event{Kind: EventInfo, Title: "Couldn't save settings", Message: err.Error()}
	}
	return &Event{Kind: EventAllDone, Title: "All downloads finished", Action: action}
}
