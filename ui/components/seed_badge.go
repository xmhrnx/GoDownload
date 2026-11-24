package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/theme"
)

// SeedBadge shows a torrent's upload setting at a glance: red when seeding
// is off, yellow when its upload speed is limited, green when it seeds freely.
func SeedBadge(t *model.TorrentOptions) giode.Element {
	color, text := SeedState(t)
	return giode.Row(giode.Styles{
		Gap: 4, Align: giode.AlignCenter, Padding: giode.SymmetricInset(6, 1),
		BorderRadius: 99, BorderWidth: 1, BorderColor: color, Wrap: giode.WrapNoWrap,
	},
		theme.Icon("upload", 11, color),
		giode.Text(text, giode.Styles{FontSize: theme.SizeCaption, Color: color, FontWeight: giode.FontWeightMedium}),
	)
}

// SeedState is the color and short label of a torrent's upload setting.
func SeedState(t *model.TorrentOptions) (giode.Color, string) {
	switch {
	case !t.Seed:
		return "theme.danger", "Seeding off"
	case t.UploadLimit > 0:
		return "theme.warning", format.Speed(t.UploadLimit) + " max"
	default:
		return theme.Seed, "Seeding on"
	}
}
