package dialogs

import (
	"fmt"
	"time"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/internal/power"
	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// countdown is how long the user has to cancel a power action.
const countdown = 60 * time.Second

// PowerDialog counts down before shutting down, restarting, sleeping or
// signing out, so someone at the computer can cancel.
type PowerDialog struct {
	s      *state.State
	open   bool
	action model.PowerAction
	title  string
	ends   time.Time
	timer  *time.Timer
	quit   chan struct{}
	dlg    *giode.DialogElement
	cancel *giode.ButtonElement
	now    *giode.ButtonElement
}

// NewPowerDialog creates the dialog.
func NewPowerDialog(s *state.State) *PowerDialog {
	d := &PowerDialog{s: s}
	d.cancel = giode.Button("Cancel", theme.QuietButton).OnClick(d.stop)
	d.now = giode.Button("Do it now", theme.DangerButton).OnClick(d.fire)
	d.dlg = giode.Dialog(&d.open, d.content, components.DialogStyles(420)).Dismissible(false)
	return d
}

// Start begins the countdown to action.
func (d *PowerDialog) Start(action model.PowerAction, title string) {
	if action == model.PowerQuitApp {
		d.s.App.Quit()
		return
	}
	d.stop()
	d.action, d.title = action, title
	d.ends = time.Now().Add(countdown)
	d.open = true
	d.s.App.ShowWindow()
	d.timer = time.AfterFunc(countdown, func() { d.s.App.Post(d.fire) })
	d.quit = make(chan struct{})
	go d.tick(d.quit)
}

// tick redraws once a second so the countdown moves.
func (d *PowerDialog) tick(quit chan struct{}) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-quit:
			return
		case <-t.C:
			d.s.App.Post(func() {})
		}
	}
}

func (d *PowerDialog) stop() {
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
		close(d.quit)
	}
	d.open = false
}

func (d *PowerDialog) fire() {
	if !d.open {
		return
	}
	action := d.action
	d.stop()
	if err := power.Perform(action); err != nil {
		d.s.Notify(fmt.Errorf("couldn't %s: %w", action.Title(), err))
	}
}

func (d *PowerDialog) content() giode.Element {
	left := max(0, int(time.Until(d.ends).Round(time.Second).Seconds()))
	body := giode.Stack(giode.Styles{Gap: 12, Width: -1, Align: giode.AlignCenter},
		giode.Text(fmt.Sprint(left), giode.Styles{FontSize: 44, FontWeight: giode.FontWeightBold, Color: "theme.warning"}),
		giode.Text(fmt.Sprintf("The computer will %s in %d seconds.", lower(d.action.Title()), left),
			giode.Styles{FontSize: theme.SizeBody, Color: "theme.muted"}),
	)
	return components.DialogFrame("power", d.title, "", body, d.cancel, d.now)
}

func lower(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]+'a'-'A') + s[1:]
}

// Layout draws the dialog while open.
func (d *PowerDialog) Layout(gtx giode.Context) giode.Dimensions { return d.dlg.Layout(gtx) }
