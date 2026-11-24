package dialogs

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/theme"
)

// chip is an outlined label, such as "Resumable", on a link in the add dialog.
func chip(icon, text string, color giode.Color) giode.Element {
	return giode.Row(giode.Styles{Gap: 4, Align: giode.AlignCenter, Padding: giode.SymmetricInset(7, 2),
		BorderRadius: 99, BorderWidth: 1, BorderColor: color, Wrap: giode.WrapNoWrap},
		theme.Icon(icon, 11, color),
		giode.Text(text, giode.Styles{FontSize: theme.SizeCaption, Color: color}),
	)
}
