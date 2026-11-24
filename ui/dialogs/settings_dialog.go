package dialogs

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// SettingsDialog edits the preferences, one section at a time.
type SettingsDialog struct {
	s        *state.State
	open     bool
	dlg      *giode.DialogElement
	sections []settingsSection
	nav      []*giode.PressableElement
	current  int
	scroll   *giode.ScrollElement
	cancel   *giode.ButtonElement
	save     *giode.ButtonElement
}

// NewSettingsDialog creates the dialog and its sections.
func NewSettingsDialog(s *state.State) *SettingsDialog {
	d := &SettingsDialog{s: s}
	d.sections = []settingsSection{
		newGeneralSettings(s), newDownloadSettings(), newFolderSettings(s),
		newTorrentSettings(), newStreamingSettings(s), newBrowserSettings(s),
	}
	for i, sec := range d.sections {
		d.nav = append(d.nav, giode.Pressable(func(p giode.PressState) giode.Element {
			return d.navItem(i, sec, p.Hovered)
		}).Label(sec.Title()).OnClick(func(giode.ClickEvent) { d.current = i }))
	}
	d.scroll = giode.Scroll(func() giode.Element { return d.sections[d.current].View() },
		giode.Styles{ScrollBar: giode.ScrollBar{AutoHide: true, Width: 5}})
	d.cancel = giode.Button("Cancel", theme.QuietButton).OnClick(func() { d.open = false })
	d.save = giode.Button("Save", theme.PrimaryButton).OnClick(d.apply)
	d.dlg = giode.Dialog(&d.open, d.content, components.DialogStyles(820)).Padding(giode.Inset{})
	return d
}

// Open shows the settings as they are now.
func (d *SettingsDialog) Open() {
	st := d.s.M.Settings()
	for _, sec := range d.sections {
		sec.Load(st)
	}
	d.open = true
}

func (d *SettingsDialog) apply() {
	st := d.s.M.Settings()
	for _, sec := range d.sections {
		if err := sec.Apply(&st); err != nil {
			d.s.Notify(err)
		}
	}
	if err := d.s.M.UpdateSettings(st); err != nil {
		d.s.Notify(err)
		return
	}
	d.open = false
	d.s.Toasts.Show(giode.ToastSuccess, "Settings saved")
}

func (d *SettingsDialog) navItem(i int, sec settingsSection, hovered bool) giode.Element {
	bg, fg := giode.Color("transparent"), giode.Color("theme.text")
	if i == d.current {
		bg, fg = theme.Selected, "theme.primary"
	} else if hovered {
		bg = theme.Raised
	}
	return giode.Row(giode.Styles{Gap: 10, Align: giode.AlignCenter, Width: -1, Padding: giode.SymmetricInset(12, 8),
		Background: bg, BorderRadius: 8, Wrap: giode.WrapNoWrap},
		theme.Icon(sec.Icon(), 16, fg),
		giode.Text(sec.Title(), giode.Styles{FontSize: theme.SizeBody, Color: fg, FontWeight: giode.FontWeightMedium}),
	)
}

func (d *SettingsDialog) content() giode.Element {
	nav := []giode.Element{giode.Box(giode.Styles{Padding: giode.Inset{Left: 12, Bottom: 12, Top: 4}},
		giode.Text("Settings", theme.Strong(17)))}
	for _, n := range d.nav {
		nav = append(nav, n)
	}
	return giode.Row(giode.Styles{Width: -1, Height: 560, Wrap: giode.WrapNoWrap},
		giode.Stack(giode.Styles{Width: 200, Height: -1, Gap: 2, Padding: giode.UniformInset(14),
			Background: theme.Sidebar, BorderRadii: giode.Corners{TopLeft: 16, BottomLeft: 16}}, nav...),
		giode.Stack(giode.Styles{FlexGrow: 1, Height: -1, Padding: giode.UniformInset(22), Gap: 14},
			giode.Text(d.sections[d.current].Title(), theme.Strong(theme.SizeHeading)),
			giode.Box(giode.Styles{FlexGrow: 1, Width: -1}, d.scroll),
			giode.Row(giode.Styles{Gap: 8, Width: -1, Justify: giode.JustifyEnd}, d.cancel, d.save),
		),
	)
}

// Layout draws the dialog while open.
func (d *SettingsDialog) Layout(gtx giode.Context) giode.Dimensions { return d.dlg.Layout(gtx) }
