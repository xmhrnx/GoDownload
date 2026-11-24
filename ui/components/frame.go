package components

import "github.com/xmhrnx/giode"

// NewTitleBar picks the platform's frame: the Mac one on macOS, the Windows
// one elsewhere.
func NewTitleBar(app *giode.App, trailing ...giode.Element) giode.Element {
	if giode.IsPlatform(giode.PlatformMacOS) {
		return NewMacTitleBar(app, trailing...)
	}
	return NewWindowsTitleBar(app, trailing...)
}
