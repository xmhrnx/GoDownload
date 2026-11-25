package manager

import (
	"slices"

	"github.com/godownload/godownload/internal/bt"
	"github.com/godownload/godownload/internal/model"
)

// Snapshot copies every download with its live stats, oldest first. Torrent
// file and peer lists are left out; Detail has them.
func (m *Manager) Snapshot() []model.Item {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]model.Item, 0, len(m.entries))
	for _, e := range m.entries {
		it := e.item()
		it.Blocks = slices.Clone(e.stats.Blocks)
		it.Segments = slices.Clone(e.stats.Segments)
		if e.d.Torrent != nil {
			t := *e.d.Torrent
			t.Files = slices.Clone(t.Files)
			it.Torrent = &t
		}
		out = append(out, it)
	}
	slices.SortFunc(out, func(a, b model.Item) int { return a.AddedAt.Compare(b.AddedAt) })
	return out
}

// Detail is one download with its torrent files, peers and log.
func (m *Manager) Detail(id string) (model.Item, []string, error) {
	m.mu.Lock()
	e, err := m.get(id)
	if err != nil {
		m.mu.Unlock()
		return model.Item{}, nil, err
	}
	it := e.item()
	it.Blocks = slices.Clone(e.stats.Blocks)
	it.Segments = slices.Clone(e.stats.Segments)
	if e.d.Torrent != nil {
		t := *e.d.Torrent
		t.Files = slices.Clone(t.Files)
		it.Torrent = &t
	}
	log := slices.Clone(e.log)
	h := e.torrent
	m.mu.Unlock()
	switch {
	case h != nil:
		it.Files = h.Files()
		it.PeerList = h.Peers()
	case it.Torrent != nil:
		// Not running: list the files from the saved .torrent so they can
		// still be chosen; progress per file isn't known until it runs.
		if files, err := bt.SavedFiles(m.store.TorrentDir(), it.Torrent.InfoHash); err == nil {
			for i := range files {
				if i < len(it.Torrent.Files) {
					files[i].Priority = it.Torrent.Files[i]
				}
				if it.Status.Finished() && files[i].Priority != model.PrioritySkip {
					files[i].Done = files[i].Size
				}
			}
			it.Files = files
		}
	}
	return it, log, nil
}

// Totals is the combined transfer state for the status bar.
type Totals struct {
	Down, Up       int64
	Active, Queued int
	Seeding        int
}

// Totals sums speeds and counts across all downloads.
func (m *Manager) Totals() Totals {
	m.mu.Lock()
	defer m.mu.Unlock()
	var t Totals
	for _, e := range m.entries {
		t.Down += e.stats.DownSpeed
		t.Up += e.stats.UpSpeed
		switch {
		case e.d.Status == model.StatusSeeding:
			t.Seeding++
		case e.d.Status.Active():
			t.Active++
		case e.d.Status == model.StatusQueued && !e.d.Stopped:
			t.Queued++
		}
	}
	return t
}

// Path is where a download's file (or torrent folder) is on disk.
func (m *Manager) Path(id string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, err := m.get(id)
	if err != nil {
		return "", err
	}
	return e.path(), nil
}
