package dialogs

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/theme"
)

// settingsSection is one page of the settings dialog.
type settingsSection interface {
	Title() string
	Icon() string
	Load(model.Settings)
	Apply(*model.Settings) error
	View() giode.Element
}

func newSwitch(label string) *giode.SwitchElement {
	return giode.Input.Switch(label, giode.Styles{FontSize: theme.SizeBody, Color: "theme.text"})
}

func newNumber(min, max, step float64) *giode.NumberInputElement {
	return giode.Input.Number(min, giode.Styles{Width: 140}).Range(min, max).Step(step).Decimals(0)
}
