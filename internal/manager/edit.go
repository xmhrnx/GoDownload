package manager

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/godownload/godownload/internal/fileutil"
	"github.com/godownload/godownload/internal/httpdl"
	"github.com/godownload/godownload/internal/model"
)

// Remove takes downloads off the list. With deleteFiles, their files go too
// (finished or partial); without, files on disk are left alone.
func (m *Manager) Remove(deleteFiles bool, ids ...string) error {
	m.mu.Lock()
	var waits []chan struct{}
	var gone []*entry
	for _, id := range ids {
		e, err := m.get(id)
		if err != nil {
			continue
		}
		e.d.Stopped = true
		if e.running() {
			waits = append(waits, e.done)
			e.stop()
		}
		gone = append(gone, e)
		delete(m.entries, id)
	}
	m.dirty = true
	m.mu.Unlock()
	for _, w := range waits {
		<-w
	}

	var errs []error
	for _, e := range gone {
		if e.torrent != nil {
			e.torrent.Close()
		}
		if e.http != nil {
			errs = append(errs, e.http.Close())
		}
		if err := os.Remove(m.store.StatePath(e.d.ID)); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err)
		}
		if deleteFiles {
			errs = append(errs, deleteDownloadFiles(e.d))
		}
	}
	m.mu.Lock()
	errs = append(errs, m.saveLocked())
	m.scheduleLocked()
	m.mu.Unlock()
	m.changed()
	return errors.Join(errs...)
}

// deleteDownloadFiles removes exactly what a download wrote: its file and
// partial file, or the torrent's file or folder.
func deleteDownloadFiles(d model.Download) error {
	if d.Name == "" || d.Dir == "" {
		return nil
	}
	path := filepath.Join(d.Dir, d.Name)
	if filepath.Clean(path) == filepath.Clean(d.Dir) {
		return nil // never the whole folder
	}
	var errs []error
	for _, p := range []string{path, path + fileutil.PartSuffix} {
		if err := os.RemoveAll(p); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// SetSpeedLimit caps one download's speed; 0 removes the cap.
func (m *Manager) SetSpeedLimit(id string, bps int64) error {
	return m.edit(id, func(e *entry) {
		e.d.SpeedLimit = bps
		e.limiter.Set(bps)
	})
}

// SetConnections changes how many connections an HTTP download opens.
func (m *Manager) SetConnections(id string, n int) error {
	return m.edit(id, func(e *entry) {
		e.d.MaxConnections = n
		if e.http != nil {
			e.http.SetConnections(n)
		}
	})
}

// SetSeed turns uploading on or off for a torrent, taking effect at once:
// off stops a seeding torrent and sends nothing more to peers (also while it
// downloads); on starts a finished torrent seeding again.
func (m *Manager) SetSeed(id string, on bool) error {
	return m.edit(id, func(e *entry) {
		if e.d.Torrent == nil {
			return
		}
		e.d.Torrent.Seed = on
		if e.torrent != nil {
			e.torrent.SetSeed(on)
		}
		switch {
		case !on && e.d.Status == model.StatusSeeding:
			e.logf("Seeding stopped")
			e.stop()
		case on && e.d.Status == model.StatusCompleted && !e.running():
			e.logf("Seeding started")
			e.seedDone, e.d.Stopped = false, false
			e.d.Status = model.StatusQueued
		}
	})
}

// SetUploadLimit caps a torrent's upload speed; 0 removes the cap.
func (m *Manager) SetUploadLimit(id string, bps int64) error {
	return m.edit(id, func(e *entry) {
		if e.d.Torrent == nil {
			return
		}
		e.d.Torrent.UploadLimit = bps
		if e.torrent != nil {
			e.torrent.SetUploadLimit(bps)
		}
	})
}

// SetSeedGoal sets when seeding stops: a ratio and/or minutes; 0 for no limit.
func (m *Manager) SetSeedGoal(id string, ratio float64, minutes int) error {
	return m.edit(id, func(e *entry) {
		if e.d.Torrent != nil {
			e.d.Torrent.SeedRatio, e.d.Torrent.SeedMinutes = ratio, minutes
		}
	})
}

// SetFilePriorities chooses which torrent files to download.
func (m *Manager) SetFilePriorities(id string, prios []model.FilePriority) error {
	return m.edit(id, func(e *entry) {
		if e.d.Torrent == nil {
			return
		}
		e.d.Torrent.Files = append([]model.FilePriority(nil), prios...)
		if e.torrent != nil {
			e.torrent.SetPriorities(prios)
		}
		// Wanting more of a finished torrent means downloading again.
		if e.d.Status.Finished() && !e.running() {
			e.d.Status, e.d.Stopped = model.StatusQueued, false
		}
	})
}

// UpdateURL replaces an expired link with a fresh one and retries.
func (m *Manager) UpdateURL(id, url string, h model.RequestHeaders) error {
	return m.edit(id, func(e *entry) {
		e.d.URL = url
		if h != (model.RequestHeaders{}) {
			e.d.Request = h
		}
		if e.http != nil {
			e.http.SetRequest(httpdl.Request{URL: url, Headers: e.d.Request})
		}
		e.hls = nil
		if e.d.Status == model.StatusFailed {
			e.d.Status, e.d.Stopped, e.d.Error = model.StatusQueued, false, ""
		}
		e.logf("Link refreshed")
	})
}

func (m *Manager) edit(id string, fn func(*entry)) error {
	m.mu.Lock()
	e, err := m.get(id)
	if err == nil {
		fn(e)
		m.dirty = true
		m.scheduleLocked()
	}
	m.mu.Unlock()
	m.changed()
	return err
}
