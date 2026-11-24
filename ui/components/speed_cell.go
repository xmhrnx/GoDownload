package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/theme"
)

// speedCell shows the download speed, and the upload speed when seeding.
func speedCell(it model.Item) giode.Element {
	kids := []giode.Element{giode.Text(format.Speed(it.DownSpeed), giode.Styles{FontSize: theme.SizeSmall, Color: "theme.text"})}
	if it.UpSpeed > 0 {
		kids = append(kids, giode.Text("↑ "+format.Speed(it.UpSpeed), giode.Styles{FontSize: theme.SizeCaption, Color: theme.Seed}))
	}
	return giode.Stack(giode.Styles{Gap: 2}, kids...)
}
