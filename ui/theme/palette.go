// Package theme holds GoDownload's look: palette, fonts, icons and the
// shared styles components build on.
package theme

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
)

// Custom color names, resolved per mode as "theme.<name>".
const (
	Raised   = "theme.raised"   // cards and hovered rows
	Sidebar  = "theme.sidebar"  // the navigation column
	Selected = "theme.selected" // selected rows and sidebar items
	Track    = "theme.track"    // empty part of progress bars
	Seed     = "theme.seed"     // seeding/upload color
)

var dark = giode.Palette{
	Primary:    "#7C6CFA",
	Secondary:  "#8A90A6",
	Background: "#0E1016",
	Surface:    "#151821",
	Text:       "#E9EBF3",
	Border:     "#252A38",
	OnPrimary:  "#FFFFFF",
	Muted:      "#8A90A6",
	Accent:     "#22C3E6",
	Success:    "#34D399",
	Warning:    "#FBBF24",
	Danger:     "#F87171",
	Info:       "#60A5FA",
	Custom: map[string]giode.Color{
		"raised":   "#1C2030",
		"sidebar":  "#11131A",
		"selected": "#272347",
		"track":    "#262B3B",
		"seed":     "#34D399",
	},
}

var light = giode.Palette{
	Primary:    "#6550F0",
	Secondary:  "#6B7186",
	Background: "#F5F6FA",
	Surface:    "#FFFFFF",
	Text:       "#151821",
	Border:     "#E3E5EE",
	OnPrimary:  "#FFFFFF",
	Muted:      "#6B7186",
	Accent:     "#0AA5C8",
	Success:    "#10B981",
	Warning:    "#D97706",
	Danger:     "#E5484D",
	Info:       "#3B82F6",
	Custom: map[string]giode.Color{
		"raised":   "#F0F1F7",
		"sidebar":  "#EEEFF5",
		"selected": "#E8E5FF",
		"track":    "#E4E6EF",
		"seed":     "#10B981",
	},
}

// Setup installs the palettes, fonts and icons. Call it before the window opens.
func Setup() error {
	giode.SetPalette(giode.ModeDark, dark)
	giode.SetPalette(giode.ModeLight, light)
	if err := registerFonts(); err != nil {
		return err
	}
	return registerIcons()
}

// Apply switches between light, dark and the system's choice.
func Apply(t model.Theme) {
	switch t {
	case model.ThemeLight:
		giode.SetMode(giode.ModeLight)
	case model.ThemeDark:
		giode.SetMode(giode.ModeDark)
	default:
		if dark, err := giode.SystemDarkMode(); err != nil || dark {
			giode.SetMode(giode.ModeDark)
		} else {
			giode.SetMode(giode.ModeLight)
		}
	}
}
