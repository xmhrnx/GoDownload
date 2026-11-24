package dialogs

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/theme"
)

// addEntryCard shows one link in the add dialog: its file, size, whether it
// can resume, or what went wrong. nameField replaces the name when given.
func addEntryCard(e *addEntry, nameField giode.Element) giode.Element {
	var title giode.Element = giode.Text(e.name(), theme.Ellipsis(theme.Strong(theme.SizeBody)))
	if nameField != nil {
		title = nameField
	}
	var info []giode.Element
	switch {
	case e.probing && e.torrent != nil:
		info = append(info, giode.Spinner(giode.Styles{Width: 12, Height: 12, Color: "theme.primary"}),
			giode.Text("Asking peers for the torrent's file list…", theme.Muted(theme.SizeSmall)))
	case e.probing:
		info = append(info, giode.Spinner(giode.Styles{Width: 12, Height: 12, Color: "theme.primary"}),
			giode.Text("Checking the link…", theme.Muted(theme.SizeSmall)))
	case e.err != nil:
		info = append(info, theme.Icon("alert", 13, "theme.danger"),
			giode.Text(e.err.Error(), theme.Ellipsis(giode.Styles{FontSize: theme.SizeSmall, Color: "theme.danger"})))
	default:
		size := "Unknown size"
		if s := e.size(); s >= 0 {
			size = format.Bytes(s)
		}
		info = append(info, giode.Text(size, theme.Muted(theme.SizeSmall)))
		switch {
		case e.torrent != nil:
			info = append(info, chip("magnet", "Torrent", "theme.primary"))
		case e.probed.Resumable:
			info = append(info, chip("check", "Resumable · multi-connection", "theme.success"))
		default:
			info = append(info, chip("alert", "Can't resume", "theme.warning"))
		}
	}
	return giode.Row(giode.Styles{Gap: 12, Align: giode.AlignCenter, Width: -1, Wrap: giode.WrapNoWrap,
		Padding: giode.UniformInset(12), Background: theme.Raised, BorderRadius: 10},
		components.FileBadge(e.category(), e.torrent != nil),
		giode.Stack(giode.Styles{FlexGrow: 1, Gap: 6},
			title,
			giode.Row(giode.Styles{Gap: 8, Align: giode.AlignCenter, Wrap: giode.WrapNoWrap}, info...),
		),
	)
}
