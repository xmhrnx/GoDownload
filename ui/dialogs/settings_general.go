package dialogs

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// generalSettings: look, tray, start at login, notifications, clipboard.
type generalSettings struct {
	s         *state.State
	theme     *giode.SegmentedElement
	tray      *giode.SwitchElement
	login     *giode.SwitchElement
	notify    *giode.SwitchElement
	clipboard *giode.SwitchElement
}

func newGeneralSettings(s *state.State) *generalSettings {
	return &generalSettings{
		s:         s,
		theme:     giode.Input.Segmented("System", "Light", "Dark").Styles(giode.Styles{FontSize: theme.SizeSmall, Width: 240}),
		tray:      newSwitch("Keep running in the tray when closed"),
		login:     newSwitch("Start GoDownload when I sign in"),
		notify:    newSwitch("Notify me when downloads finish"),
		clipboard: newSwitch("Offer to download links I copy"),
	}
}

func (g *generalSettings) Title() string { return "General" }
func (g *generalSettings) Icon() string  { return "sliders" }

var themes = []model.Theme{model.ThemeSystem, model.ThemeLight, model.ThemeDark}

func (g *generalSettings) Load(st model.Settings) {
	for i, t := range themes {
		if t == st.Theme {
			g.theme.SetSelected(i)
		}
	}
	g.tray.SetChecked(st.CloseToTray)
	g.login.SetChecked(g.s.App.LaunchAtLogin())
	g.notify.SetChecked(st.Notifications)
	g.clipboard.SetChecked(st.WatchClipboard)
}

func (g *generalSettings) Apply(st *model.Settings) error {
	st.Theme = themes[g.theme.Selected()]
	st.CloseToTray = g.tray.Checked()
	st.Notifications = g.notify.Checked()
	st.WatchClipboard = g.clipboard.Checked()
	theme.Apply(st.Theme)
	g.s.App.SetCloseToTray(st.CloseToTray)
	if g.login.Checked() != g.s.App.LaunchAtLogin() {
		return g.s.App.SetLaunchAtLogin(g.login.Checked())
	}
	return nil
}

func (g *generalSettings) View() giode.Element {
	return giode.Stack(giode.Styles{Gap: 4, Width: -1},
		settingRow("Appearance", "System follows your computer's light or dark mode.", g.theme),
		switchRow(g.tray, "Closing the window keeps downloads and seeding going; quit from the tray icon."),
		switchRow(g.login, ""),
		switchRow(g.notify, ""),
		switchRow(g.clipboard, "When you copy a download link anywhere, GoDownload offers to download it."),
	)
}
