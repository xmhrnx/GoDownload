package dialogs

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/theme"
)

// ScheduleEditor edits one schedule's window, days, limits and power action.
type ScheduleEditor struct {
	sched     model.Schedule
	enabled   *giode.SwitchElement
	name      *giode.InputElement
	start     *giode.InputElement
	stop      *giode.InputElement
	days      *components.DayPicker
	maxActive *giode.NumberInputElement
	limit     *components.RateField
	limitVal  int64
	whenDone  *giode.SelectElement
}

// NewScheduleEditor creates the controls.
func NewScheduleEditor() *ScheduleEditor {
	st := theme.Field
	st.Padding = giode.SymmetricInset(10, 7)
	tst := st
	tst.Width = 110
	e := &ScheduleEditor{
		enabled:   newSwitch("Enabled"),
		name:      giode.Input.Text("Name", st),
		start:     giode.Input.Text("23:00", tst),
		stop:      giode.Input.Text("Never", tst),
		days:      components.NewDayPicker(),
		maxActive: newNumber(0, 20, 1),
	}
	e.limit = components.NewRateField(140, func(v int64) { e.limitVal = v })
	opts := make([]string, len(model.PowerActions))
	for i, a := range model.PowerActions {
		opts[i] = a.Title()
	}
	e.whenDone = giode.Input.Select(theme.Field, opts...)
	return e
}

// Load shows s.
func (e *ScheduleEditor) Load(s model.Schedule) {
	e.sched = s
	e.enabled.SetChecked(s.Enabled)
	e.name.SetText(s.Name)
	e.start.SetText(format.Minutes(s.Start))
	if s.Stop >= 0 {
		e.stop.SetText(format.Minutes(s.Stop))
	} else {
		e.stop.SetText("")
	}
	e.days.Days = s.Days
	e.maxActive.SetValue(float64(s.MaxActive))
	e.limit.Set(s.SpeedLimit)
	e.limitVal = s.SpeedLimit
	for i, a := range model.PowerActions {
		if a == s.WhenDone {
			e.whenDone.SetSelected(i)
		}
	}
}

// Read returns the edited schedule, or false after marking what's wrong.
func (e *ScheduleEditor) Read() (model.Schedule, bool) {
	s := e.sched
	s.Enabled = e.enabled.Checked()
	s.Name = e.name.Text()
	start, err := format.ParseClock(e.start.Text())
	if err != nil {
		e.start.Invalid("HH:MM")
		return s, false
	}
	e.start.Invalid("")
	s.Start, s.Stop = start, -1
	if t := e.stop.Text(); t != "" {
		stop, err := format.ParseClock(t)
		if err != nil {
			e.stop.Invalid("HH:MM or empty")
			return s, false
		}
		s.Stop = stop
	}
	e.stop.Invalid("")
	s.Days = e.days.Days
	s.MaxActive = int(e.maxActive.Value())
	s.SpeedLimit = e.limitVal
	s.WhenDone = model.PowerActions[e.whenDone.Selected()]
	return s, true
}

// Layout draws the editor.
func (e *ScheduleEditor) Layout(gtx giode.Context) giode.Dimensions {
	return giode.Stack(giode.Styles{Gap: 14, Width: -1},
		giode.Row(giode.Styles{Gap: 16, Align: giode.AlignEnd, Width: -1, Wrap: giode.WrapNoWrap},
			giode.Box(giode.Styles{FlexGrow: 1}, components.Labeled("Name", e.name)),
			giode.Box(giode.Styles{Width: 130, Padding: giode.Inset{Bottom: 6}}, e.enabled),
		),
		giode.Row(giode.Styles{Gap: 16, Align: giode.AlignEnd},
			components.Labeled("Start at", e.start),
			components.Labeled("Stop at (optional)", e.stop),
		),
		components.Labeled("On these days", e.days),
		giode.Row(giode.Styles{Gap: 16, Align: giode.AlignEnd},
			components.Labeled("Downloads at once (0 = setting)", e.maxActive),
			components.Labeled("Speed limit (KB/s)", e.limit),
		),
		giode.Box(giode.Styles{Width: 260}, components.Labeled("When the scheduled downloads finish", e.whenDone)),
		giode.Text("Downloads added with “On schedule”, or moved there from their menu, start when the window opens and pause when it closes. A power action runs when they all finish, or at the stop time.",
			giode.Styles{FontSize: theme.SizeSmall, Color: "theme.muted", LineHeight: 17}),
	).Layout(gtx)
}
