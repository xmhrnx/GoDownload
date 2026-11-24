package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/theme"
)

// EmptyState fills the list when it has nothing to show.
func EmptyState(searching bool, add *giode.ButtonElement) giode.Element {
	title, body := "No downloads yet", "Paste a link, drop a .torrent file here, or let the browser extension send downloads."
	if searching {
		title, body = "Nothing matches", "Try another search or filter."
	}
	kids := []giode.Element{
		giode.Box(giode.Styles{
			Width: 72, Height: 72, BorderRadius: 20, Background: theme.Raised,
			Align: giode.AlignCenter, Justify: giode.JustifyCenter,
		}, theme.Icon("inbox", 32, "theme.muted")),
		giode.Text(title, theme.Strong(theme.SizeTitle)),
		giode.Text(body, giode.Styles{Color: "theme.muted", FontSize: theme.SizeBody, MaxWidth: 340, TextAlign: giode.TextAlignCenter}),
	}
	if !searching {
		kids = append(kids, giode.Box(giode.Styles{Padding: giode.Inset{Top: 6}}, add))
	}
	return giode.Stack(giode.Styles{Width: -1, Height: -1, Align: giode.AlignCenter, Justify: giode.JustifyCenter, Gap: 10}, kids...)
}
