package dialogs

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/links"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/state"
)

// Set is every dialog. It wires itself into state.Actions and draws
// whichever dialogs are open.
type Set struct {
	Add       *AddDialog
	Batch     *BatchDialog
	Remove    *RemoveDialog
	Refresh   *RefreshLinkDialog
	Play      *PlayDialog
	Settings  *SettingsDialog
	Scheduler *SchedulerDialog
	Power     *PowerDialog
	Upload    *UploadLimitDialog
}

// NewSet creates the dialogs and points s.Actions at them.
func NewSet(s *state.State) *Set {
	d := &Set{
		Add: NewAddDialog(s), Batch: NewBatchDialog(s), Remove: NewRemoveDialog(s),
		Refresh: NewRefreshLinkDialog(s), Play: NewPlayDialog(s), Settings: NewSettingsDialog(s),
		Scheduler: NewSchedulerDialog(s), Power: NewPowerDialog(s), Upload: NewUploadLimitDialog(s),
	}
	s.Actions = state.Actions{
		Add:            d.Add.Open,
		AddTorrentFile: d.Add.PickTorrent,
		Delete:         d.Remove.Open,
		Play:           d.Play.Play,
		RefreshLink:    d.Refresh.Open,
		UploadLimit:    d.Upload.Open,
		Settings:       d.Settings.Open,
		Scheduler:      d.Scheduler.Open,
	}
	return d
}

// RunPower starts the countdown for a power action.
func (d *Set) RunPower(a model.PowerAction, title string) { d.Power.Start(a, title) }

// HasLinks reports whether text holds something to download.
func HasLinks(text string) bool { return len(links.Find(text)) > 0 }

// Layout draws the open dialogs; closed ones take no space.
func (d *Set) Layout(gtx giode.Context) giode.Dimensions {
	return giode.Box(giode.Styles{}, d.Add, d.Batch, d.Remove, d.Refresh, d.Play, d.Settings, d.Scheduler, d.Power, d.Upload).Layout(gtx)
}
