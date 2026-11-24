package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/theme"
)

// DialogFrame lays out a dialog: an icon and title, the body, and the
// buttons right-aligned at the bottom.
func DialogFrame(icon, title, subtitle string, body giode.Element, buttons ...giode.Element) giode.Element {
	head := []giode.Element{
		giode.Box(giode.Styles{Width: 36, Height: 36, BorderRadius: 10, Background: theme.Selected,
			Align: giode.AlignCenter, Justify: giode.JustifyCenter}, theme.Icon(icon, 18, "theme.primary")),
	}
	titles := []giode.Element{giode.Text(title, theme.Strong(17))}
	if subtitle != "" {
		titles = append(titles, giode.Text(subtitle, theme.Muted(theme.SizeSmall)))
	}
	head = append(head, giode.Stack(giode.Styles{Gap: 2, FlexGrow: 1}, titles...))
	foot := append([]giode.Element{giode.Box(giode.Styles{FlexGrow: 1})}, buttons...)
	return giode.Stack(giode.Styles{Gap: 18, Width: -1},
		giode.Row(giode.Styles{Gap: 12, Align: giode.AlignCenter, Width: -1, Wrap: giode.WrapNoWrap}, head...),
		body,
		giode.Row(giode.Styles{Gap: 8, Align: giode.AlignCenter, Width: -1, Wrap: giode.WrapNoWrap}, foot...),
	)
}

// DialogStyles is the panel look shared by every dialog, width in dp.
func DialogStyles(width int) giode.Styles {
	return giode.Styles{
		Background: "theme.surface", BorderRadius: 16, Width: giodeWidth(width),
		BorderWidth: 1, BorderColor: "theme.border",
		BoxShadow: giode.BoxShadow{Y: 18, Blur: 48, Color: "#00000070"},
		Padding:   giode.UniformInset(22),
	}
}
