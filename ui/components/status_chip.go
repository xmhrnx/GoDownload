package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/theme"
)

// statusChip is a small rounded label with an icon in the status bar.
func statusChip(icon, text string, color giode.Color) giode.Element {
	return giode.Row(giode.Styles{
		Align: giode.AlignCenter, Gap: 5, Padding: giode.SymmetricInset(8, 3),
		BorderRadius: 99, Background: theme.Raised, Wrap: giode.WrapNoWrap,
	},
		theme.Icon(icon, 12, color),
		giode.Text(text, giode.Styles{FontSize: theme.SizeCaption, Color: color}),
	)
}
