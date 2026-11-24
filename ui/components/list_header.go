package components

import (
	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/styles/properties"

	"github.com/godownload/godownload/ui/listmodel"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// ListHeader is the row of sortable column titles above the list.
type ListHeader struct {
	s    *state.State
	cols map[listmodel.SortKey]*giode.PressableElement
}

// NewListHeader creates the clickable column titles.
func NewListHeader(s *state.State) *ListHeader {
	h := &ListHeader{s: s, cols: map[listmodel.SortKey]*giode.PressableElement{}}
	titles := map[listmodel.SortKey]string{
		listmodel.ByName: "Name", listmodel.BySize: "Size", listmodel.ByProgress: "Progress",
		listmodel.BySpeed: "Speed", listmodel.ByETA: "Time left", listmodel.ByAdded: "Added",
	}
	for k, title := range titles {
		h.cols[k] = giode.Pressable(func(p giode.PressState) giode.Element {
			return h.title(k, title, p.Hovered)
		}).Label("Sort by " + title).OnClick(func(giode.ClickEvent) {
			s.Sort = s.Sort.Toggle(k)
			s.Relist()
		})
	}
	return h
}

func (h *ListHeader) title(k listmodel.SortKey, title string, hovered bool) giode.Element {
	active := h.s.Sort.Key == k
	color := giode.Color("theme.muted")
	if active || hovered {
		color = "theme.text"
	}
	kids := []giode.Element{giode.Text(title, giode.Styles{
		FontSize: theme.SizeSmall, FontWeight: giode.FontWeightMedium, Color: color,
	})}
	if active {
		arrow := "chevron-up"
		if h.s.Sort.Desc {
			arrow = "chevron-down"
		}
		kids = append(kids, theme.Icon(arrow, 12, "theme.primary"))
	}
	return giode.Row(giode.Styles{Gap: 4, Align: giode.AlignCenter, Wrap: giode.WrapNoWrap}, kids...)
}

// Layout draws the header.
func (h *ListHeader) Layout(gtx giode.Context) giode.Dimensions {
	return giode.Row(giode.Styles{
		Align: giode.AlignCenter, Width: -1, Wrap: giode.WrapNoWrap,
		Padding: giode.Inset{Left: 8 + rowInset + 50, Right: 8 + rowInset, Top: 6, Bottom: 8},
	},
		giode.Box(giode.Styles{FlexGrow: 1}, h.cols[listmodel.ByName]),
		cell(colSize, h.cols[listmodel.BySize]),
		cell(colProgress, h.cols[listmodel.ByProgress]),
		cell(colSpeed, h.cols[listmodel.BySpeed]),
		cell(colETA, h.cols[listmodel.ByETA]),
		cell(colAdded, h.cols[listmodel.ByAdded]),
		giode.Box(giode.Styles{Width: properties.Width(colActions)}),
	).Layout(gtx)
}
