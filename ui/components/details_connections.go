package components

import (
	"fmt"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/theme"
)

// DetailsConnections shows what each HTTP connection is fetching.
func DetailsConnections(it model.Item) giode.Element {
	if len(it.Segments) == 0 {
		return giode.Text("No open connections.", theme.Muted(theme.SizeBody))
	}
	kids := []giode.Element{}
	for i, s := range it.Segments {
		frac := 0.0
		if s.End > s.Start {
			frac = float64(s.Pos-s.Start) / float64(s.End-s.Start)
		}
		kids = append(kids, giode.Row(giode.Styles{Gap: 12, Align: giode.AlignCenter, Width: -1, Wrap: giode.WrapNoWrap},
			giode.Text(fmt.Sprintf("#%d", i+1), giode.Styles{FontSize: theme.SizeSmall, Color: "theme.muted", Width: 30}),
			giode.Box(giode.Styles{FlexGrow: 1}, ProgressBar(frac, nil, "theme.accent", 6)),
			giode.Text(fmt.Sprintf("%s – %s", format.Bytes(s.Pos), format.Bytes(s.End)),
				giode.Styles{FontSize: theme.SizeSmall, Color: "theme.text", Width: 170, TextAlign: giode.TextAlignEnd}),
		))
	}
	return giode.Stack(giode.Styles{Gap: 10, Width: -1}, kids...)
}
