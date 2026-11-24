// Package assets embeds the app icon.
package assets

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
)

var (
	// Icon is the 512 px app icon.
	//go:embed icon.png
	Icon []byte
	// Tray is the monochrome tray icon (a template icon on macOS).
	//go:embed tray.png
	Tray []byte
	//go:embed icon-256.png
	logo []byte
)

// Logo decodes the icon for drawing in the window.
func Logo() image.Image {
	img, err := png.Decode(bytes.NewReader(logo))
	if err != nil {
		panic("embedded logo is not a PNG: " + err.Error())
	}
	return img
}
