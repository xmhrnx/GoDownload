package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/theme"
)

// Labeled puts a small label above a control.
func Labeled(label string, control giode.Element) giode.Element {
	return giode.Stack(giode.Styles{Gap: 6},
		giode.Text(label, giode.Styles{FontSize: theme.SizeSmall, Color: "theme.muted", FontWeight: giode.FontWeightMedium}),
		control,
	)
}
