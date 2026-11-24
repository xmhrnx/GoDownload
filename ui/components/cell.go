package components

import (
	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/styles/properties"
)

// cell is a fixed-width column holding child.
func cell(width int, child giode.Element) giode.Element {
	return giode.Box(giode.Styles{Width: properties.Width(width), Padding: giode.Inset{Right: 8}}, child)
}
