package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// DownloadList is the header and the scrolling rows. Only rows on screen are
// built, so long histories cost nothing.
type DownloadList struct {
	s      *state.State
	header *ListHeader
	menu   *RowMenu
	list   *giode.LazyListElement
	add    *giode.ButtonElement
}

// NewDownloadList creates the list.
func NewDownloadList(s *state.State) *DownloadList {
	l := &DownloadList{s: s, header: NewListHeader(s), menu: NewRowMenu(s)}
	l.list = giode.LazyList(
		func() int { return len(s.Visible) },
		func(i int) any { return s.Visible[i].ID },
		l.row,
		giode.Styles{ScrollBar: giode.ScrollBar{AutoHide: true, Width: 6, Inset: 2}},
	)
	l.add = giode.Button("Add a download", theme.PrimaryButton).
		Icon(theme.Icon("plus", 16, "#FFFFFF")).OnClick(func() { s.Actions.Add("") })
	return l
}

func (l *DownloadList) row(i int) giode.Element {
	it := l.s.Visible[i]
	r := giode.Keyed("download-row", it.ID, func() *DownloadRow { return newDownloadRow(l.s, l.menu, it.ID) })
	r.it = it
	return r
}

// Layout draws the list.
func (l *DownloadList) Layout(gtx giode.Context) giode.Dimensions {
	if len(l.s.Visible) == 0 {
		searching := l.s.Filter.Query != "" || len(l.s.Items) > 0
		return giode.Box(giode.Styles{Width: -1, Height: -1}, EmptyState(searching, l.add), l.menu).Layout(gtx)
	}
	return giode.Stack(giode.Styles{Width: -1, Height: -1},
		l.header,
		giode.Divider(giode.Styles{Color: "theme.border"}),
		giode.Box(giode.Styles{FlexGrow: 1, Width: -1, Padding: giode.Inset{Top: 4}}, l.list),
		l.menu,
	).Layout(gtx)
}
