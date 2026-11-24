package components

import (
	"fmt"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// StatusBar is the strip along the bottom: total speeds, what's running,
// the speed limit and the "when done" action.
func StatusBar(s *state.State) giode.Element {
	t := s.Totals
	settings := s.M.Settings()
	parts := []string{fmt.Sprintf("%d downloading", t.Active)}
	if t.Queued > 0 {
		parts = append(parts, fmt.Sprintf("%d queued", t.Queued))
	}
	if t.Seeding > 0 {
		parts = append(parts, fmt.Sprintf("%d seeding", t.Seeding))
	}
	summary := ""
	for i, p := range parts {
		if i > 0 {
			summary += " · "
		}
		summary += p
	}
	limit := "No speed limit"
	if settings.DownloadLimit > 0 {
		limit = "Limit " + format.Speed(settings.DownloadLimit)
	}
	kids := []giode.Element{
		theme.Icon("arrow-down", 12, "theme.primary"),
		giode.Text(speedOrIdle(t.Down), giode.Styles{FontSize: theme.SizeSmall, Color: "theme.text"}),
		theme.Icon("arrow-up", 12, theme.Seed),
		giode.Text(speedOrIdle(t.Up), giode.Styles{FontSize: theme.SizeSmall, Color: "theme.text"}),
		giode.Box(giode.Styles{Width: 8}),
		giode.Text(summary, theme.Muted(theme.SizeSmall)),
		giode.Box(giode.Styles{FlexGrow: 1}),
	}
	if settings.WhenDone != model.PowerNone {
		kids = append(kids, statusChip("power", "When done: "+settings.WhenDone.Title(), "theme.warning"))
	}
	kids = append(kids, statusChip("gauge", limit, "theme.muted"))
	return giode.Row(giode.Styles{
		Align: giode.AlignCenter, Gap: 6, Width: -1, Wrap: giode.WrapNoWrap,
		Padding: giode.SymmetricInset(16, 6), Background: theme.Sidebar,
	}, kids...)
}
