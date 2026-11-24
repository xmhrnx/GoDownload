package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/theme"
)

// NewMacTitleBar is the macOS frame: traffic lights on the left and a
// centered title, as native Mac windows have.
func NewMacTitleBar(app *giode.App, trailing ...giode.Element) *giode.TitleBarElement {
	return giode.TitleBar(app, "GoDownload", giode.Styles{
		Background: theme.Sidebar,
		Color:      "theme.text",
		Height:     38,
		FontSize:   theme.SizeBody,
		FontWeight: giode.FontWeightSemiBold,
	}).
		Variant(giode.TitleBarMac).
		Trailing(trailing...)
}
