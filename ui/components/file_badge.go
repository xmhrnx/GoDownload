package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/theme"
)

var badgeTints = map[model.Category]giode.Color{}

func init() {
	for _, c := range model.Categories {
		badgeTints[c] = giode.Mix(theme.CategoryColor(c), "theme.surface", 0.16)
	}
}

// FileBadge is a rounded tile with the icon of a download's category; a
// small magnet marks torrents.
func FileBadge(c model.Category, torrent bool) giode.Element {
	kids := []giode.Element{theme.Icon(theme.CategoryIcon(c), 18, theme.CategoryColor(c))}
	if torrent {
		kids = append(kids, giode.Box(giode.Styles{
			Position: giode.PositionAbsolute, Right: -4, Bottom: -4,
			Background: "theme.surface", BorderRadius: 99, Padding: giode.UniformInset(2),
			BorderWidth: 1, BorderColor: "theme.border",
		}, theme.Icon("magnet", 10, "theme.muted")))
	}
	return giode.Box(giode.Styles{
		Width: 38, Height: 38, BorderRadius: 10, Background: badgeTints[c],
		Align: giode.AlignCenter, Justify: giode.JustifyCenter,
	}, kids...)
}
