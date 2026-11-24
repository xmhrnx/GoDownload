package components

import (
	"fmt"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/theme"
)

// DetailsPeers lists a torrent's connected peers.
func DetailsPeers(peers []model.Peer) giode.Element {
	if len(peers) == 0 {
		return giode.Text("Not connected to any peers.", theme.Muted(theme.SizeBody))
	}
	head := func(s string) giode.Element { return giode.Text(s, theme.Muted(theme.SizeCaption)) }
	kids := []giode.Element{head("Address"), head("Client"), head("Has"), head("Down"), head("Up")}
	for _, p := range peers {
		client := p.Client
		if client == "" {
			client = "—"
		}
		kids = append(kids,
			giode.Text(p.Addr, theme.Ellipsis(giode.Styles{FontSize: theme.SizeSmall, Color: "theme.text"})),
			giode.Text(client, theme.Ellipsis(theme.Muted(theme.SizeSmall))),
			giode.Text(fmt.Sprintf("%.0f%%", p.Progress*100), theme.Muted(theme.SizeSmall)),
			giode.Text(format.Speed(p.DownSpeed), giode.Styles{FontSize: theme.SizeSmall, Color: "theme.text"}),
			giode.Text(format.Speed(p.UpSpeed), giode.Styles{FontSize: theme.SizeSmall, Color: theme.Seed}),
		)
	}
	return giode.Grid(giode.Styles{GridColumns: "2fr 1.5fr 60 90 90", Gap: 12, RowGap: 8, Width: -1}, kids...)
}
