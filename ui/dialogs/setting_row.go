package dialogs

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/theme"
)

// settingRow is a label and help text beside a control.
func settingRow(label, help string, control giode.Element) giode.Element {
	text := []giode.Element{giode.Text(label, giode.Styles{FontSize: theme.SizeBody, Color: "theme.text", FontWeight: giode.FontWeightMedium})}
	if help != "" {
		text = append(text, giode.Text(help, giode.Styles{FontSize: theme.SizeSmall, Color: "theme.muted", LineHeight: 17}))
	}
	return giode.Row(giode.Styles{Gap: 16, Align: giode.AlignCenter, Width: -1, Wrap: giode.WrapNoWrap, Padding: giode.SymmetricInset(0, 8)},
		giode.Stack(giode.Styles{Gap: 3, FlexGrow: 1}, text...),
		control,
	)
}
