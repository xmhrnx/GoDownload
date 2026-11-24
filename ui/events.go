package ui

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/manager"
)

// OnEvent shows what the manager reports. Runs on the UI goroutine.
func (r *Root) OnEvent(ev manager.Event) {
	s := r.s
	switch ev.Kind {
	case manager.EventCompleted:
		s.Toasts.ShowAction(giode.ToastSuccess, "Finished: "+ev.Title, "Open", func() { s.OpenFile(ev.ID) })
		if s.M.Settings().Notifications && !s.App.Focused() {
			// A notification that can't show is not worth interrupting for.
			_ = s.App.Notify("Download complete", ev.Title)
		}
	case manager.EventFailed:
		if ev.Expired {
			s.Toasts.ShowAction(giode.ToastError, ev.Title+": link expired", "Refresh link", func() { s.Actions.RefreshLink(ev.ID) })
			return
		}
		s.Toasts.Show(giode.ToastError, ev.Title+": "+ev.Message)
	case manager.EventInfo:
		s.Toasts.Show(giode.ToastInfo, ev.Title+": "+ev.Message)
	case manager.EventAllDone:
		r.dialogs.RunPower(ev.Action, ev.Title)
	}
}
