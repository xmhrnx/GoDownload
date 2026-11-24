package dialogs

import (
	"fmt"
	"strconv"
	"time"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// SchedulerDialog manages schedules: when queued downloads run, at what
// speed, and what the computer does afterwards.
type SchedulerDialog struct {
	s       *state.State
	open    bool
	dlg     *giode.DialogElement
	list    []model.Schedule
	current int
	editor  *ScheduleEditor
	addBtn  *giode.ButtonElement
	delBtn  *giode.ButtonElement
	cancel  *giode.ButtonElement
	save    *giode.ButtonElement
}

// NewSchedulerDialog creates the dialog.
func NewSchedulerDialog(s *state.State) *SchedulerDialog {
	d := &SchedulerDialog{s: s, editor: NewScheduleEditor()}
	d.addBtn = giode.Button("New schedule", theme.QuietButton).Icon(theme.Icon("plus", 14, "")).OnClick(d.add)
	d.delBtn = giode.Button("Delete", giode.Styles{Color: "theme.danger", FontSize: theme.SizeBody}).
		Variant(giode.VariantGhost).OnClick(d.remove)
	d.cancel = giode.Button("Cancel", theme.QuietButton).OnClick(func() { d.open = false })
	d.save = giode.Button("Save", theme.PrimaryButton).OnClick(d.apply)
	d.dlg = giode.Dialog(&d.open, d.content, components.DialogStyles(780)).Top(60)
	return d
}

// Open shows the saved schedules.
func (d *SchedulerDialog) Open() {
	d.list = append([]model.Schedule(nil), d.s.M.Settings().Schedules...)
	d.current = -1
	if len(d.list) == 0 {
		d.add()
	} else {
		d.selectSchedule(0)
	}
	d.open = true
}

func (d *SchedulerDialog) add() {
	if !d.commit() {
		return
	}
	s := model.Schedule{
		ID: strconv.FormatInt(time.Now().UnixNano(), 36), Name: fmt.Sprintf("Schedule %d", len(d.list)+1),
		Enabled: true, Start: 2 * 60, Stop: 7 * 60,
	}
	for i := range s.Days {
		s.Days[i] = true
	}
	d.list = append(d.list, s)
	d.current = -1
	d.selectSchedule(len(d.list) - 1)
}

func (d *SchedulerDialog) remove() {
	if d.current < 0 {
		return
	}
	d.list = append(d.list[:d.current], d.list[d.current+1:]...)
	d.current = -1
	if len(d.list) > 0 {
		d.selectSchedule(0)
	}
}

// commit keeps the editor's changes to the current schedule.
func (d *SchedulerDialog) commit() bool {
	if d.current < 0 || d.current >= len(d.list) {
		return true
	}
	s, ok := d.editor.Read()
	if ok {
		d.list[d.current] = s
	}
	return ok
}

func (d *SchedulerDialog) selectSchedule(i int) {
	if !d.commit() {
		return
	}
	d.current = i
	d.editor.Load(d.list[i])
}

func (d *SchedulerDialog) apply() {
	if !d.commit() {
		return
	}
	st := d.s.M.Settings()
	st.Schedules = d.list
	if err := d.s.M.UpdateSettings(st); err != nil {
		d.s.Notify(err)
		return
	}
	d.open = false
	d.s.Toasts.Show(giode.ToastSuccess, "Schedules saved")
}

func (d *SchedulerDialog) index(id string) int {
	for i, s := range d.list {
		if s.ID == id {
			return i
		}
	}
	return -1
}

func (d *SchedulerDialog) scheduleButton(id string) giode.Element {
	return giode.Keyed("schedule-btn", id, func() *giode.PressableElement {
		return giode.Pressable(func(p giode.PressState) giode.Element {
			i := d.index(id)
			if i < 0 {
				return giode.Box(giode.Styles{})
			}
			s := d.list[i]
			bg := giode.Color("transparent")
			if i == d.current {
				bg = theme.Selected
			} else if p.Hovered {
				bg = theme.Raised
			}
			when := format.Minutes(s.Start)
			if s.Stop >= 0 {
				when += " – " + format.Minutes(s.Stop)
			}
			if !s.Enabled {
				when += " · off"
			}
			return giode.Stack(giode.Styles{Gap: 2, Width: -1, Padding: giode.SymmetricInset(12, 8), Background: bg, BorderRadius: 8},
				giode.Text(s.Name, theme.Ellipsis(theme.Strong(theme.SizeBody))),
				giode.Text(when, theme.Muted(theme.SizeSmall)),
			)
		}).OnClick(func(giode.ClickEvent) {
			if i := d.index(id); i >= 0 {
				d.selectSchedule(i)
			}
		})
	})
}

func (d *SchedulerDialog) waiting() int {
	n := 0
	for _, it := range d.s.Items {
		if it.Scheduled && !it.Status.Finished() {
			n++
		}
	}
	return n
}

func (d *SchedulerDialog) content() giode.Element {
	left := []giode.Element{}
	for _, s := range d.list {
		left = append(left, d.scheduleButton(s.ID))
	}
	left = append(left, giode.Box(giode.Styles{Padding: giode.Inset{Top: 8}}, d.addBtn))
	var right giode.Element = giode.Text("No schedules.", theme.Muted(theme.SizeBody))
	if d.current >= 0 {
		right = giode.Stack(giode.Styles{Gap: 10, Width: -1}, d.editor,
			giode.Row(giode.Styles{Width: -1, Justify: giode.JustifyEnd}, d.delBtn))
	}
	body := giode.Row(giode.Styles{Gap: 20, Width: -1, Align: giode.AlignStart, Wrap: giode.WrapNoWrap},
		giode.Stack(giode.Styles{Width: 200, Gap: 4}, left...),
		giode.Box(giode.Styles{Width: 1, Height: 380, Background: "theme.border"}),
		giode.Box(giode.Styles{FlexGrow: 1}, right),
	)
	return components.DialogFrame("calendar", "Scheduler",
		fmt.Sprintf("%d downloads waiting for a schedule", d.waiting()), body, d.cancel, d.save)
}

// Layout draws the dialog while open.
func (d *SchedulerDialog) Layout(gtx giode.Context) giode.Dimensions { return d.dlg.Layout(gtx) }
