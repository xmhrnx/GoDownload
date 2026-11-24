package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/theme"
)

// NewWindowsTitleBar is the frame for Windows and Linux: logo and name on the
// left, caption buttons on the right with a red close hover.
func NewWindowsTitleBar(app *giode.App, trailing ...giode.Element) *giode.TitleBarElement {
	return giode.TitleBar(app, "GoDownload", giode.Styles{
		Background: theme.Sidebar,
		Color:      "theme.text",
		Height:     40,
		FontSize:   theme.SizeBody,
		FontWeight: giode.FontWeightSemiBold,
		Padding:    giode.Inset{Left: 12},
	}).
		Variant(giode.TitleBarWindows).
		Icon(Logo(18)).
		Trailing(trailing...)
}
