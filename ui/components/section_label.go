package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/theme"
)

// SectionLabel is a small uppercase heading above a group.
func SectionLabel(text string) giode.Element {
	return giode.Box(giode.Styles{Padding: giode.Inset{Left: 10, Top: 14, Bottom: 6}},
		giode.Text(text, giode.Styles{
			Color: "theme.muted", FontSize: theme.SizeCaption, FontWeight: giode.FontWeightSemiBold,
			TextTransform: giode.TextTransformUppercase,
		}))
}
