package theme

import (
	"embed"

	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/styles/properties"
)

//go:embed icons
var iconFS embed.FS

func registerIcons() error { return giode.RegisterIconsFS(iconFS, "icons") }

// Icon is a named icon at size dp in color c ("" inherits the text color).
func Icon(name string, size int, c giode.Color) giode.Element {
	return giode.Icon(name, giode.Styles{Width: properties.Width(size), Height: properties.Height(size), Color: c})
}

// IconSource is a registered icon for components that take an icon source
// (menu items, checkboxes).
func IconSource(name string) giode.IconSource {
	src, _ := giode.LookupIcon(name)
	return src
}
