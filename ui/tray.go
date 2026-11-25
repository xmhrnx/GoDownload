package ui

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/assets"
)

// setupTray puts GoDownload in the system tray, so it keeps downloading and
// seeding with the window closed.
func (r *Root) setupTray() {
	if !giode.TraySupported() {
		return
	}
	s := r.s
	tray := s.App.Tray()
	if giode.IsPlatform(giode.PlatformMacOS) {
		tray.SetTemplateIcon(assets.Tray) // tinted by macOS to suit the menu bar
	} else {
		tray.SetIcon(assets.Icon)
	}
	// Set once: every tray change rebuilds the whole menu, which closes it
	// under the pointer if it happens while the menu is open.
	tray.SetTooltip("GoDownload").
		SetMenu(
			giode.TrayAction("Show GoDownload", s.App.ShowWindow),
			giode.TrayAction("New download…", func() { s.App.ShowWindow(); s.Actions.Add("") }),
			giode.TraySeparator(),
			giode.TrayAction("Resume all", s.M.ResumeAll),
			giode.TrayAction("Pause all", s.M.PauseAll),
			giode.TraySeparator(),
			giode.TrayAction("Quit", s.App.Quit),
		)
}
