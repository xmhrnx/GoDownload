package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/theme"
)

var dayNames = [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

// DayPicker toggles the days of the week, indexed by time.Weekday.
type DayPicker struct {
	Days [7]bool
	btns [7]*giode.PressableElement
}

// NewDayPicker creates the seven toggles.
func NewDayPicker() *DayPicker {
	d := &DayPicker{}
	for i := range d.btns {
		d.btns[i] = giode.Pressable(func(p giode.PressState) giode.Element {
			bg, fg := giode.Color(theme.Raised), giode.Color("theme.text")
			if d.Days[i] {
				bg, fg = "theme.primary", "theme.onprimary"
			} else if p.Hovered {
				bg = theme.Selected
			}
			return giode.Box(giode.Styles{Width: 46, Padding: giode.SymmetricInset(0, 7), BorderRadius: 8, Background: bg,
				Align: giode.AlignCenter, Justify: giode.JustifyCenter},
				giode.Text(dayNames[i], giode.Styles{FontSize: theme.SizeSmall, Color: fg, FontWeight: giode.FontWeightMedium}))
		}).Label(dayNames[i]).OnClick(func(giode.ClickEvent) { d.Days[i] = !d.Days[i] })
	}
	return d
}

// Layout draws the toggles, Monday first.
func (d *DayPicker) Layout(gtx giode.Context) giode.Dimensions {
	order := []int{1, 2, 3, 4, 5, 6, 0}
	kids := make([]giode.Element, len(order))
	for i, day := range order {
		kids[i] = d.btns[day]
	}
	return giode.Row(giode.Styles{Gap: 6, Wrap: giode.WrapNoWrap}, kids...).Layout(gtx)
}
