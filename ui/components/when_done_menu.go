package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/state"
)

// WhenDoneMenu chooses what the computer does once every download finishes.
type WhenDoneMenu struct {
	el giode.Element
}

// NewWhenDoneMenu creates the menu and the toolbar button that opens it.
func NewWhenDoneMenu(s *state.State) *WhenDoneMenu {
	menu := giode.Menu()
	for _, a := range model.PowerActions {
		menu.RadioItem(a.Title(),
			func() bool { return s.M.Settings().WhenDone == a },
			func() {
				st := s.M.Settings()
				st.WhenDone = a
				s.Notify(s.M.UpdateSettings(st))
				if a != model.PowerNone {
					s.Toasts.Show(giode.ToastInfo, "When all downloads finish: "+a.Title())
				}
			})
	}
	btn := NewToolButton("power", "When downloads finish…", nil)
	return &WhenDoneMenu{el: menu.Trigger(btn)}
}

// Layout draws the button.
func (w *WhenDoneMenu) Layout(gtx giode.Context) giode.Dimensions { return w.el.Layout(gtx) }
