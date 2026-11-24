package components

import (
	"fmt"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/listmodel"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// Toolbar sits above the list: the view's title, search, actions on the
// selection, and app-wide actions.
type Toolbar struct {
	s        *state.State
	search   *giode.InputElement
	resume   *ToolButton
	pause    *ToolButton
	play     *ToolButton
	folder   *ToolButton
	repair   *ToolButton
	remove   *ToolButton
	whenDone *WhenDoneMenu
	schedule *ToolButton
	settings *ToolButton
}

// NewToolbar creates the toolbar's controls once.
func NewToolbar(s *state.State) *Toolbar {
	t := &Toolbar{s: s}
	withSelection := func(fn func(ids []string)) func() {
		return func() {
			if ids := s.Selected(); len(ids) > 0 {
				fn(ids)
				return
			}
			s.Toasts.Show(giode.ToastInfo, "Select a download first")
		}
	}
	first := func(fn func(id string)) func() {
		return withSelection(func(ids []string) { fn(ids[0]) })
	}
	t.search = giode.Input.Text("Search downloads", giode.Styles{
		Background: "theme.surface", BorderColor: "theme.border", BorderRadius: 8,
		FontSize: theme.SizeBody, Width: 220, Padding: giode.SymmetricInset(10, 7),
	}).OnChange(func(q string) {
		s.Filter.Query = q
		s.Relist()
	})
	t.resume = NewToolButton("play", "Resume", withSelection(func(ids []string) { s.M.Resume(ids...) }))
	t.pause = NewToolButton("pause", "Pause", withSelection(func(ids []string) { s.M.Pause(ids...) }))
	t.play = NewToolButton("tv", "Play while downloading", first(func(id string) { s.Actions.Play(id) }))
	t.folder = NewToolButton("folder", "Show in folder", first(s.OpenFolder))
	t.repair = NewToolButton("shield", "Check & repair", withSelection(func(ids []string) {
		for _, id := range ids {
			s.Notify(s.M.Repair(id))
		}
	}))
	t.remove = NewToolButton("trash", "Remove…", withSelection(func(ids []string) { s.Actions.Delete(ids) }))
	t.whenDone = NewWhenDoneMenu(s)
	t.schedule = NewToolButton("calendar", "Scheduler", func() { s.Actions.Scheduler() })
	t.settings = NewToolButton("settings", "Settings", func() { s.Actions.Settings() })
	return t
}

// Focus puts the cursor in the search field.
func (t *Toolbar) Focus() { t.search.Focus() }

func (t *Toolbar) title() string {
	f := t.s.Filter
	switch {
	case f.Category != "":
		return f.Category.Title()
	case f.Status != listmodel.ShowAll:
		return f.Status.Title()
	default:
		return "All downloads"
	}
}

// Layout draws the toolbar.
func (t *Toolbar) Layout(gtx giode.Context) giode.Dimensions {
	sep := giode.Box(giode.Styles{Width: 1, Height: 20, Background: "theme.border", Margin: giode.SymmetricInset(4, 0)})
	return giode.Row(giode.Styles{
		Align: giode.AlignCenter, Gap: 4, Padding: giode.Inset{Left: 20, Right: 12, Top: 12, Bottom: 10},
		Width: -1, Wrap: giode.WrapNoWrap,
	},
		giode.Stack(giode.Styles{Gap: 2, FlexGrow: 1},
			giode.Text(t.title(), giode.Styles{FontSize: theme.SizeHeading, FontWeight: giode.FontWeightBold, Color: "theme.text"}),
			giode.Text(fmt.Sprintf("%d items", len(t.s.Visible)), theme.Muted(theme.SizeSmall)),
		),
		t.search,
		giode.Box(giode.Styles{Width: 8}),
		t.resume, t.pause, t.play, t.folder, t.repair, t.remove,
		sep,
		t.whenDone, t.schedule, t.settings,
	).Layout(gtx)
}
