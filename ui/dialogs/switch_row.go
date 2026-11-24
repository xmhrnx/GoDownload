package dialogs

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/theme"
)

// switchRow is a setting that is on or off.
func switchRow(sw giode.Element, help string) giode.Element {
	kids := []giode.Element{sw}
	if help != "" {
		kids = append(kids, giode.Text(help, giode.Styles{FontSize: theme.SizeSmall, Color: "theme.muted", LineHeight: 17}))
	}
	return giode.Stack(giode.Styles{Gap: 2, Width: -1, Padding: giode.SymmetricInset(0, 6)}, kids...)
}
