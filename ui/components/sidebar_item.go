package components

import (
	"strconv"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/theme"
)

// SidebarItem is one navigation entry: an icon, a label and a count.
type SidebarItem struct {
	press    *giode.PressableElement
	icon     string
	tint     giode.Color
	label    string
	count    func() int
	selected func() bool
}

// NewSidebarItem creates an entry; count and selected are read every frame.
func NewSidebarItem(icon string, tint giode.Color, label string, count func() int, selected func() bool, onClick func()) *SidebarItem {
	it := &SidebarItem{icon: icon, tint: tint, label: label, count: count, selected: selected}
	it.press = giode.Pressable(it.content).Label(label).OnClick(func(giode.ClickEvent) { onClick() })
	return it
}

func (it *SidebarItem) content(s giode.PressState) giode.Element {
	sel := it.selected()
	bg := giode.Color("transparent")
	fg := giode.Color("theme.text")
	iconColor := it.tint
	switch {
	case sel:
		bg = theme.Selected
		iconColor = "theme.primary"
	case s.Hovered:
		bg = theme.Raised
	}
	weight := giode.FontWeightNormal
	if sel {
		weight = giode.FontWeightSemiBold
	}
	n := it.count()
	countText := ""
	if n > 0 {
		countText = strconv.Itoa(n)
	}
	return giode.Row(giode.Styles{
		Align:        giode.AlignCenter,
		Gap:          10,
		Padding:      giode.SymmetricInset(10, 7),
		Background:   bg,
		BorderRadius: 8,
		Width:        -1,
		Wrap:         giode.WrapNoWrap,
	},
		theme.Icon(it.icon, 16, iconColor),
		giode.Box(giode.Styles{FlexGrow: 1},
			giode.Text(it.label, theme.Ellipsis(giode.Styles{Color: fg, FontSize: theme.SizeBody, FontWeight: weight}))),
		giode.Text(countText, theme.Muted(theme.SizeSmall)),
	)
}

// Layout draws the entry.
func (it *SidebarItem) Layout(gtx giode.Context) giode.Dimensions { return it.press.Layout(gtx) }
