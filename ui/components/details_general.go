package components

import (
	"fmt"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/theme"
)

// DetailsGeneral is the overview tab: the block map and the numbers.
func DetailsGeneral(it model.Item) giode.Element {
	stats := []giode.Element{
		StatItem("Status", theme.StatusLabel(it.Status)),
		StatItem("Size", sizeOrUnknown(it.Size)),
		StatItem("Downloaded", format.Bytes(it.Done)+"  "+format.Percent(it.Done, it.Size)),
		StatItem("Speed", format.Speed(it.DownSpeed)),
		StatItem("Time left", etaText(it)),
		StatItem("Added", it.AddedAt.Format("Jan 2, 2006 15:04")),
	}
	if !it.CompletedAt.IsZero() {
		stats = append(stats, StatItem("Finished", it.CompletedAt.Format("Jan 2, 2006 15:04")))
	}
	if it.Kind == model.KindTorrent && it.Torrent != nil {
		ratio := 0.0
		if it.Size > 0 {
			ratio = float64(it.Torrent.Uploaded) / float64(it.Size)
		}
		stats = append(stats,
			StatItem("Peers", fmt.Sprintf("%d (%d seeds)", it.Peers, it.Seeders)),
			StatItem("Uploaded", fmt.Sprintf("%s · ratio %.2f", format.Bytes(it.Torrent.Uploaded), ratio)),
			StatItem("Upload speed", format.Speed(it.UpSpeed)),
			StatItem("Info hash", it.Torrent.InfoHash),
		)
	} else {
		resumable := "No — restarts if interrupted"
		if it.Resumable || it.Status == model.StatusCompleted {
			resumable = "Yes"
		}
		if it.Size < 0 && it.Done == 0 {
			resumable = "Unknown until it starts"
		}
		stats = append(stats,
			StatItem("Connections", fmt.Sprint(it.Connections)),
			StatItem("Resumable", resumable),
		)
		if it.Checksum != "" {
			stats = append(stats, StatItem("Checksum", it.Checksum))
		}
	}
	stats = append(stats, StatItem("Saved to", it.Dir))
	if it.URL != "" {
		stats = append(stats, StatItem("Address", it.URL))
	}
	return giode.Stack(giode.Styles{Gap: 16, Width: -1},
		BlockMap(it),
		giode.Grid(giode.Styles{GridColumns: "repeat(auto-fill, minmax(170, 1fr))", Gap: 16, RowGap: 14, Width: -1}, stats...),
	)
}
