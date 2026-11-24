package ui

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/assets"
	"github.com/godownload/godownload/internal/format"
)

// setupTray puts GoDownload in the system tray, so it keeps downloading and
// seeding with the window closed.
func (r *Root) setupTray() {
	if !giode.TraySupported() {
		return
	}
	s := r.s
	s.App.Tray().
		SetTemplateIcon(assets.Tray).
		SetTooltip("GoDownload").
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

// updateTray shows the speeds in the tray tooltip.
func (r *Root) updateTray() {
	if !giode.TraySupported() {
		return
	}
	t := r.s.Totals
	tip := "GoDownload"
	if t.Active > 0 || t.Seeding > 0 {
		tip += " — ↓ " + speed(t.Down) + "  ↑ " + speed(t.Up)
	}
	if tip != r.lastTip {
		r.lastTip = tip
		r.s.App.Tray().SetTooltip(tip)
	}
}

func speed(bps int64) string {
	if bps <= 0 {
		return "0 B/s"
	}
	return format.Speed(bps)
}
