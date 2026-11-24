package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// RowMenu is the right-click menu of the list. It is rebuilt each time it
// opens, so it offers only what fits the selection.
type RowMenu struct {
	s    *state.State
	menu *giode.MenuElement
}

// NewRowMenu creates the menu.
func NewRowMenu(s *state.State) *RowMenu {
	return &RowMenu{s: s, menu: giode.Menu()}
}

// OpenAt rebuilds the menu for the selection and opens it at the click.
func (r *RowMenu) OpenAt(e giode.ClickEvent) {
	s := r.s
	ids := s.Selected()
	if len(ids) == 0 {
		return
	}
	first, _ := s.Item(ids[0])
	m := r.menu.Clear()
	if first.Status.Finished() {
		m.IconItem(theme.IconSource("external"), "Open", func() { s.OpenFile(first.ID) })
	}
	if first.Category == model.CategoryVideo || first.Category == model.CategoryMusic || first.Kind == model.KindTorrent {
		m.IconItem(theme.IconSource("tv"), "Play while downloading", func() { s.Actions.Play(first.ID) })
	}
	m.IconItem(theme.IconSource("folder"), "Show in folder", func() { s.OpenFolder(first.ID) })
	m.Separator()
	m.IconItem(theme.IconSource("play"), "Resume", func() { s.M.Resume(ids...) })
	m.IconItem(theme.IconSource("pause"), "Pause", func() { s.M.Pause(ids...) })
	m.IconItem(theme.IconSource("shield"), "Check & repair", func() {
		for _, id := range ids {
			s.Notify(s.M.Repair(id))
		}
	})
	if first.Kind != model.KindTorrent {
		m.IconItem(theme.IconSource("link"), "Refresh link…", func() { s.Actions.RefreshLink(first.ID) })
		m.IconItem(theme.IconSource("refresh"), "Download again", func() { s.Notify(s.M.Restart(first.ID)) })
	}
	if t := first.Torrent; t != nil {
		if t.Seed {
			m.IconItem(theme.IconSource("stop"), "Stop seeding", func() { r.setSeed(ids, false) })
		} else {
			m.IconItem(theme.IconSource("upload"), "Start seeding", func() { r.setSeed(ids, true) })
		}
		m.IconItem(theme.IconSource("gauge"), "Upload speed limit…", func() { s.Actions.UploadLimit(ids) })
	}
	m.IconItem(theme.IconSource("copy"), "Copy link", func() { s.CopyLink(first.ID) })
	if !first.Status.Finished() {
		if first.Scheduled {
			m.IconItem(theme.IconSource("calendar"), "Remove from schedule", func() { s.M.SetScheduled(false, ids...) })
		} else {
			m.IconItem(theme.IconSource("calendar"), "Download on schedule", func() { s.M.SetScheduled(true, ids...) })
		}
	}
	m.Separator()
	m.IconItem(theme.IconSource("trash"), "Remove…", func() { s.Actions.Delete(ids) })
	m.OpenAtClick(e)
}

// setSeed turns seeding on or off for the torrents among ids.
func (r *RowMenu) setSeed(ids []string, on bool) {
	for _, id := range ids {
		if it, ok := r.s.Item(id); ok && it.Torrent != nil {
			r.s.Notify(r.s.M.SetSeed(id, on))
		}
	}
}

// Layout lets the menu draw while open; it takes no space.
func (r *RowMenu) Layout(gtx giode.Context) giode.Dimensions { return r.menu.Layout(gtx) }
