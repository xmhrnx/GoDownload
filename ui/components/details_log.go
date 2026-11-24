package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/theme"
)

// DetailsLog shows a download's event log, newest last.
func DetailsLog(lines []string) giode.Element {
	if len(lines) == 0 {
		return giode.Text("Nothing logged yet.", theme.Muted(theme.SizeBody))
	}
	kids := make([]giode.Element, len(lines))
	for i, l := range lines {
		kids[i] = giode.Text(l, giode.Styles{FontSize: theme.SizeSmall, Color: "theme.text", FontFamily: "Go Mono"})
	}
	return giode.SelectionArea("download-log", giode.Stack(giode.Styles{Gap: 4, Width: -1}, kids...))
}
