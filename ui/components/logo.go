// Package components holds GoDownload's reusable UI pieces, one per file.
package components

import (
	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/styles/properties"

	"github.com/godownload/godownload/assets"
)

var logoImage = assets.Logo()

// Logo is the app icon at size dp.
func Logo(size int) giode.Element {
	return giode.Image(logoImage, giode.Styles{
		Width: properties.Width(size), Height: properties.Height(size), Fit: giode.FitContain,
	})
}
