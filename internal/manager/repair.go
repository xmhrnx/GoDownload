package manager

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/godownload/godownload/internal/fileutil"
	"github.com/godownload/godownload/internal/httpdl"
	"github.com/godownload/godownload/internal/model"
)

// Repair checks a download's data and fetches whatever is damaged or missing.
//
// HTTP downloads compare every block with the checksum recorded when it was
// written, then download just the bad blocks again from the link (refresh the
// link first if it expired). Torrents rehash every piece against the
// torrent's own hashes and download the pieces that fail. Video streams, which
// keep no per-block checksums, download again.
func (m *Manager) Repair(id string) error {
	m.mu.Lock()
	e, err := m.get(id)
	if err != nil {
		m.mu.Unlock()
		return err
	}
	if e.busy {
		m.mu.Unlock()
		return errors.New("already checking")
	}
	switch e.d.Kind {
	case model.KindTorrent:
		e.verify = true
		e.d.Stopped = false
		if e.running() {
			e.stop() // the next run verifies first
		}
		e.d.Status = model.StatusQueued
		m.scheduleLocked()
		m.mu.Unlock()
		m.changed()
		return nil
	case model.KindHLS:
		m.mu.Unlock()
		return m.restart(id)
	}
	e.busy = true
	wait := e.done
	e.stop()
	m.mu.Unlock()
	if wait != nil {
		<-wait
	}
	go m.repairHTTP(e)
	return nil
}

func (m *Manager) repairHTTP(e *entry) {
	m.mu.Lock()
	dl := m.httpDownloaderLocked(e)
	wasDone := e.d.Status.Finished()
	e.d.Status = model.StatusVerifying
	size := e.d.Size
	m.mu.Unlock()
	m.changed()

	bad, err := dl.Check(m.ctx, func(done, total int64) {
		m.mu.Lock()
		e.d.Done, e.d.Size = done, total
		m.mu.Unlock()
	})
	var msg string
	switch {
	case errors.Is(err, httpdl.ErrNoChecksums):
		msg = "This file has no block checksums to compare, so it will be downloaded again."
		err = dl.Forget()
	case err != nil:
		msg = "Check failed: " + err.Error()
	case bad > 0:
		msg = fmt.Sprintf("Found %d damaged %s; downloading %s again.", bad, plural(bad, "block", "blocks"), plural(bad, "it", "them"))
	default:
		msg = "No damage found."
	}

	m.mu.Lock()
	e.busy = false
	e.d.Size = size
	e.logf("Check: %s", msg)
	_, statErr := os.Stat(e.path())
	intact := err == nil && bad == 0 && wasDone && statErr == nil
	if intact {
		e.d.Status, e.d.Done = model.StatusCompleted, size
	} else {
		e.d.Status, e.d.Stopped, e.d.Error = model.StatusQueued, false, ""
		m.hadWork = true
	}
	m.dirty = true
	m.scheduleLocked()
	m.mu.Unlock()
	m.emit(Event{Kind: EventInfo, ID: e.d.ID, Title: e.d.Name, Message: msg})
	m.changed()
}

// restart forgets all progress and downloads from scratch.
func (m *Manager) restart(id string) error {
	m.mu.Lock()
	e, err := m.get(id)
	if err != nil {
		m.mu.Unlock()
		return err
	}
	e.busy = true
	wait := e.done
	e.stop()
	m.mu.Unlock()
	if wait != nil {
		<-wait
	}
	m.mu.Lock()
	defer func() {
		m.mu.Unlock()
		m.changed()
	}()
	e.busy = false
	if e.http != nil {
		e.http.Close()
		e.http = nil
	}
	e.hls = nil
	for _, p := range []string{m.store.StatePath(id), e.path() + fileutil.PartSuffix} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	e.d.Done, e.d.Error, e.d.CompletedAt = 0, "", time.Time{}
	e.d.Status, e.d.Stopped = model.StatusQueued, false
	e.logf("Restarted from scratch")
	m.dirty, m.hadWork = true, true
	m.scheduleLocked()
	return nil
}

// Restart downloads the file again from the beginning.
func (m *Manager) Restart(id string) error { return m.restart(id) }

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
