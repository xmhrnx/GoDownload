package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/listmodel"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// Sidebar is the navigation column: the add button, categories, status
// filters and the live speed graph.
type Sidebar struct {
	s          *state.State
	add        *giode.ButtonElement
	all        *SidebarItem
	categories []*SidebarItem
	statuses   []*SidebarItem
	scroll     *giode.ScrollElement
}

// NewSidebar builds the sidebar's entries once.
func NewSidebar(s *state.State) *Sidebar {
	sb := &Sidebar{s: s}
	sb.add = giode.Button("New download", theme.PrimaryButton).
		Icon(theme.Icon("plus", 16, "#FFFFFF")).
		OnClick(func() { s.Actions.Add("") })
	sb.all = NewSidebarItem("layers", "theme.muted", "All downloads",
		func() int { return s.Counts.All },
		func() bool { return s.Filter.Category == "" && s.Filter.Status == listmodel.ShowAll },
		func() { sb.choose("", listmodel.ShowAll) })
	for _, c := range model.Categories {
		sb.categories = append(sb.categories, NewSidebarItem(theme.CategoryIcon(c), theme.CategoryColor(c), c.Title(),
			func() int { return s.Counts.Category[c] },
			func() bool { return s.Filter.Category == c },
			func() { sb.choose(c, listmodel.ShowAll) }))
	}
	icons := map[listmodel.StatusFilter]string{
		listmodel.ShowActive: "download", listmodel.ShowPaused: "pause-circle",
		listmodel.ShowCompleted: "check-circle", listmodel.ShowSeeding: "upload", listmodel.ShowFailed: "alert",
	}
	for _, f := range listmodel.StatusFilters {
		sb.statuses = append(sb.statuses, NewSidebarItem(icons[f], "theme.muted", f.Title(),
			func() int { return s.Counts.Status[f] },
			func() bool { return s.Filter.Status == f },
			func() { sb.choose("", f) }))
	}
	sb.scroll = giode.Scroll(sb.entries, giode.Styles{ScrollBar: giode.ScrollBar{AutoHide: true, Width: 4}})
	return sb
}

func (sb *Sidebar) choose(c model.Category, f listmodel.StatusFilter) {
	sb.s.Filter.Category, sb.s.Filter.Status = c, f
	sb.s.Relist()
}

func (sb *Sidebar) entries() giode.Element {
	kids := []giode.Element{sb.all, SectionLabel("Categories")}
	for _, it := range sb.categories {
		kids = append(kids, it)
	}
	kids = append(kids, SectionLabel("Status"))
	for _, it := range sb.statuses {
		kids = append(kids, it)
	}
	return giode.Stack(giode.Styles{Gap: 2, Padding: giode.SymmetricInset(10, 0), Width: -1}, kids...)
}

// Layout draws the sidebar.
func (sb *Sidebar) Layout(gtx giode.Context) giode.Dimensions {
	t := sb.s.Totals
	return giode.Stack(giode.Styles{
		Width: 228, Height: -1, Background: theme.Sidebar,
		BorderColor: "theme.border", Padding: giode.Inset{Top: 12, Bottom: 12},
	},
		giode.Box(giode.Styles{Padding: giode.SymmetricInset(12, 0), Width: -1},
			giode.Box(giode.Styles{Width: -1}, sb.add)),
		giode.Box(giode.Styles{FlexGrow: 1, Width: -1, Padding: giode.Inset{Top: 10}}, sb.scroll),
		giode.Box(giode.Styles{Padding: giode.SymmetricInset(12, 0), Width: -1},
			SpeedGraph(sb.s.Speeds, t.Down, t.Up)),
	).Layout(gtx)
}
