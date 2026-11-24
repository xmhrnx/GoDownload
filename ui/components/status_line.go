package components

import (
	"fmt"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/theme"
)

// StatusLine is the small line under a download's name: a colored status and
// what's happening, or the error of a failed download.
func StatusLine(it model.Item) giode.Element {
	color := theme.StatusColor(it.Status)
	detail := statusDetail(it)
	detailStyle := theme.Muted(theme.SizeSmall)
	if it.Status == model.StatusFailed {
		detailStyle.Color = "theme.danger"
	}
	kids := []giode.Element{
		giode.Box(giode.Styles{Width: 7, Height: 7, BorderRadius: 99, Background: color}),
		giode.Text(theme.StatusLabel(it.Status), giode.Styles{Color: color, FontSize: theme.SizeSmall, FontWeight: giode.FontWeightMedium}),
	}
	if it.Torrent != nil {
		kids = append(kids, SeedBadge(it.Torrent))
	}
	kids = append(kids, giode.Box(giode.Styles{FlexGrow: 1}, giode.Text(detail, theme.Ellipsis(detailStyle))))
	return giode.Row(giode.Styles{Gap: 6, Align: giode.AlignCenter, Wrap: giode.WrapNoWrap}, kids...)
}

func statusDetail(it model.Item) string {
	switch it.Status {
	case model.StatusFailed:
		return it.Error
	case model.StatusSeeding:
		ratio := 0.0
		if it.Size > 0 && it.Torrent != nil {
			ratio = float64(it.Torrent.Uploaded) / float64(it.Size)
		}
		return fmt.Sprintf("· ratio %.2f · %d peers", ratio, it.Peers)
	case model.StatusCompleted:
		return "· " + format.Bytes(it.Size)
	case model.StatusDownloading, model.StatusConnecting:
		s := "· " + format.Bytes(it.Done) + " of " + sizeOrUnknown(it.Size)
		if it.Kind == model.KindTorrent {
			s += fmt.Sprintf(" · %d/%d peers", it.Seeders, it.Peers)
		} else if it.Connections > 0 {
			s += fmt.Sprintf(" · %d connections", it.Connections)
		}
		if it.Streaming {
			s += " · streaming"
		}
		return s
	case model.StatusVerifying:
		return "· " + format.Percent(it.Done, it.Size)
	default:
		if it.Scheduled {
			return "· waiting for schedule"
		}
		return "· " + format.Bytes(it.Done) + " of " + sizeOrUnknown(it.Size)
	}
}

func sizeOrUnknown(n int64) string {
	if n <= 0 {
		return "unknown size"
	}
	return format.Bytes(n)
}
