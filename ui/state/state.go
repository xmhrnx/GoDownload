// Package state is the UI's model: the latest snapshot of the download list,
// what is filtered, sorted and selected, and the actions components trigger.
// Everything here runs on the UI goroutine.
package state

import (
	"sync/atomic"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/manager"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/internal/server"
	"github.com/godownload/godownload/ui/listmodel"
)

// State is shared by every screen and component.
type State struct {
	App    *giode.App
	M      *manager.Manager
	Server *server.Server
	Toasts *giode.ToasterElement

	Items   []model.Item
	Visible []model.Item
	Counts  listmodel.Counts
	Totals  manager.Totals
	// Speeds is the recent total download speed, for the sidebar graph.
	Speeds []float64

	Filter listmodel.Filter
	Sort   listmodel.Sort
	Sel    *listmodel.Selection

	// Detail is the focused download with its files and peers.
	Detail    model.Item
	DetailLog []string
	// DetailsOpen shows the details panel under the list.
	DetailsOpen bool

	Actions Actions
	// AfterRefresh runs after each refresh, for the tray and keep-awake.
	AfterRefresh func()

	pending atomic.Bool
}

// Actions open dialogs and other things owned by the root view.
type Actions struct {
	// Add opens the add dialog, pre-filled with text (links, magnets).
	Add func(text string)
	// AddTorrentFile picks a .torrent file and opens the add dialog with it.
	AddTorrentFile func()
	// Delete asks whether to remove downloads, and whether to delete their files.
	Delete func(ids []string)
	// Play opens a download in a media player while it downloads.
	Play func(id string)
	// RefreshLink asks for a fresh link for an expired download.
	RefreshLink func(id string)
	// UploadLimit asks for an upload speed limit for torrents.
	UploadLimit func(ids []string)
	Settings    func()
	Scheduler   func()
}

// New creates the state; call Refresh once the manager is running.
func New(app *giode.App, m *manager.Manager, srv *server.Server, toasts *giode.ToasterElement) *State {
	return &State{
		App: app, M: m, Server: srv, Toasts: toasts,
		Sort: listmodel.DefaultSort, Sel: listmodel.NewSelection(), DetailsOpen: true,
	}
}

// Changed schedules a refresh from any goroutine; bursts of changes are
// coalesced into one refresh per frame.
func (s *State) Changed() {
	if s.pending.CompareAndSwap(false, true) {
		s.App.Post(func() {
			s.pending.Store(false)
			s.Refresh()
		})
	}
}

// Refresh copies the manager's list and recomputes what's shown.
func (s *State) Refresh() {
	s.Items = s.M.Snapshot()
	s.Totals = s.M.Totals()
	s.Speeds = append(s.Speeds, float64(s.Totals.Down))
	if len(s.Speeds) > 60 {
		s.Speeds = s.Speeds[len(s.Speeds)-60:]
	}
	s.Relist()
	if s.Sel.Focus != "" {
		if it, log, err := s.M.Detail(s.Sel.Focus); err == nil {
			s.Detail, s.DetailLog = it, log
		}
	}
	if s.AfterRefresh != nil {
		s.AfterRefresh()
	}
}

// Relist reapplies the filter and sort without fetching new data.
func (s *State) Relist() {
	s.Visible = listmodel.Apply(s.Items, s.Filter, s.Sort)
	s.Counts = listmodel.Count(s.Items)
	ids := map[string]bool{}
	for _, it := range s.Items {
		ids[it.ID] = true
	}
	s.Sel.Prune(func(id string) bool { return ids[id] })
}

// VisibleIDs lists the shown downloads in order.
func (s *State) VisibleIDs() []string {
	out := make([]string, len(s.Visible))
	for i, it := range s.Visible {
		out[i] = it.ID
	}
	return out
}

// Selected lists the selected downloads in list order.
func (s *State) Selected() []string { return s.Sel.IDs(s.VisibleIDs()) }

// Focus shows a download's details.
func (s *State) Focus(id string) {
	s.Sel.Focus = id
	s.DetailsOpen = true
	if it, log, err := s.M.Detail(id); err == nil {
		s.Detail, s.DetailLog = it, log
	}
}

// Item finds a download in the latest snapshot.
func (s *State) Item(id string) (model.Item, bool) {
	for _, it := range s.Items {
		if it.ID == id {
			return it, true
		}
	}
	return model.Item{}, false
}
