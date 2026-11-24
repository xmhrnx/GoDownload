// Package ui is GoDownload's window: it lays out the screen and owns the
// dialogs, wiring them to the components through state.Actions.
package ui

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/dialogs"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// Root is the whole window.
type Root struct {
	s        *state.State
	title    giode.Element
	themeBtn *components.ToolButton
	sidebar  *components.Sidebar
	toolbar  *components.Toolbar
	list     *components.DownloadList
	details  *components.DetailsPanel
	dialogs  *dialogs.Set

	release  func() // ends the keep-awake request
	lastClip string
	lastTip  string
}

// NewRoot builds the window's components and dialogs once.
func NewRoot(s *state.State) *Root {
	r := &Root{s: s}
	r.themeBtn = components.NewToolButton(themeIcon(), "Light or dark", r.toggleTheme)
	r.title = components.NewTitleBar(s.App, r.themeBtn)
	r.sidebar = components.NewSidebar(s)
	r.toolbar = components.NewToolbar(s)
	r.list = components.NewDownloadList(s)
	r.details = components.NewDetailsPanel(s)
	r.dialogs = dialogs.NewSet(s)
	return r
}

// Start sets up the tray and clipboard watching, and refreshes from the
// manager whenever it changes.
func (r *Root) Start() {
	r.s.AfterRefresh = func() {
		r.keepAwake()
		r.updateTray()
	}
	r.setupTray()
	r.watchClipboard()
	r.s.Refresh()
}

// Dialogs gives main access to dialogs opened from outside the window
// (browser links, the power countdown).
func (r *Root) Dialogs() *dialogs.Set { return r.dialogs }

func themeIcon() string {
	if giode.CurrentMode() == giode.ModeDark {
		return "sun"
	}
	return "moon"
}

func (r *Root) toggleTheme() {
	st := r.s.M.Settings()
	if giode.CurrentMode() == giode.ModeDark {
		st.Theme = model.ThemeLight
	} else {
		st.Theme = model.ThemeDark
	}
	theme.Apply(st.Theme)
	r.themeBtn.SetIcon(themeIcon())
	r.s.Notify(r.s.M.UpdateSettings(st))
}

// View is the frame's element tree.
func (r *Root) View() giode.Element {
	r.shortcuts()
	main := giode.Stack(giode.Styles{FlexGrow: 1, Height: -1, Background: "theme.background"},
		r.toolbar,
		giode.Box(giode.Styles{FlexGrow: 1, Width: -1}, r.list),
		r.details,
	)
	return giode.Stack(giode.Styles{Width: -1, Height: -1, Background: "theme.background"},
		r.title,
		giode.Divider(giode.Styles{Color: "theme.border"}),
		giode.Row(giode.Styles{FlexGrow: 1, Width: -1, Wrap: giode.WrapNoWrap},
			r.sidebar,
			giode.Box(giode.Styles{Width: 1, Height: -1, Background: "theme.border"}),
			main,
		),
		giode.Divider(giode.Styles{Color: "theme.border"}),
		components.StatusBar(r.s),
		r.dialogs,
		r.s.Toasts,
	)
}

func (r *Root) shortcuts() {
	s := r.s
	giode.Shortcuts(
		giode.Shortcut("Mod+N", func() { s.Actions.Add("") }),
		giode.Shortcut("Mod+O", func() { s.Actions.AddTorrentFile() }),
		giode.Shortcut("Mod+F", r.toolbar.Focus),
		giode.Shortcut("Mod+,", func() { s.Actions.Settings() }),
		giode.Shortcut("Mod+A", func() { s.Sel.SelectAll(s.VisibleIDs()) }),
		giode.Shortcut("Delete", func() {
			if ids := s.Selected(); len(ids) > 0 {
				s.Actions.Delete(ids)
			}
		}),
		giode.Shortcut("Mod+V", r.pasteLinks),
	)
}

// pasteLinks opens the add dialog with links from the clipboard.
func (r *Root) pasteLinks() {
	r.s.App.ReadClipboard(func(text string) {
		if dialogs.HasLinks(text) {
			r.s.Actions.Add(text)
		}
	})
}
