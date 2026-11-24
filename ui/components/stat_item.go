package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/theme"
)

// StatItem is a label over a value in the details panel.
func StatItem(label, value string) giode.Element {
	if value == "" {
		value = "—"
	}
	return giode.Stack(giode.Styles{Gap: 3},
		giode.Text(label, theme.Muted(theme.SizeCaption)),
		giode.Text(value, theme.Ellipsis(giode.Styles{FontSize: theme.SizeBody, Color: "theme.text", FontWeight: giode.FontWeightMedium})),
	)
}
