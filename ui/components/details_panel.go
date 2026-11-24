package components

import (
	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/styles/properties"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// Details tabs.
const (
	tabGeneral = iota
	tabFiles
	tabPeers
	tabConnections
	tabLog
)

// DetailsPanel shows the focused download under the list.
type DetailsPanel struct {
	s       *state.State
	tabs    *giode.SegmentedElement
	tabIDs  []int
	torrent bool
	files   *TorrentFileList
	options *DetailsOptions
	scroll  *giode.ScrollElement
	close   *ToolButton
	resize  *ResizeHandle
	lastID  string
	height  int
}

// Details panel heights in dp. The list keeps at least listMin.
const (
	detailsDefault = 280
	detailsMin     = 150
	listMin        = 140
)

// NewDetailsPanel creates the panel.
func NewDetailsPanel(s *state.State) *DetailsPanel {
	d := &DetailsPanel{s: s, options: NewDetailsOptions(s)}
	d.files = NewTorrentFileList(true, func(p []model.FilePriority) {
		s.Notify(s.M.SetFilePriorities(s.Detail.ID, p))
	})
	d.scroll = giode.Scroll(d.body, giode.Styles{ScrollBar: giode.ScrollBar{AutoHide: true, Width: 5}})
	d.close = NewToolButton("x", "Hide details", func() { s.DetailsOpen = false })
	d.height = s.M.Settings().DetailsHeight
	if d.height <= 0 {
		d.height = detailsDefault
	}
	d.resize = NewResizeHandle(
		func() int { return d.height },
		func(h int) { d.height = h },
		func() {
			st := s.M.Settings()
			st.DetailsHeight = d.height
			s.Notify(s.M.UpdateSettings(st))
		},
		detailsDefault,
	)
	d.setTabs(false)
	return d
}

// setTabs offers the tabs that fit the download kind.
func (d *DetailsPanel) setTabs(torrent bool) {
	d.torrent = torrent
	if torrent {
		d.tabs = giode.Input.Segmented("General", "Files", "Peers", "Log")
		d.tabIDs = []int{tabGeneral, tabFiles, tabPeers, tabLog}
	} else {
		d.tabs = giode.Input.Segmented("General", "Connections", "Log")
		d.tabIDs = []int{tabGeneral, tabConnections, tabLog}
	}
	d.tabs.Styles(giode.Styles{FontSize: theme.SizeSmall})
}

func (d *DetailsPanel) tab() int { return d.tabIDs[d.tabs.Selected()] }

func (d *DetailsPanel) body() giode.Element {
	it := d.s.Detail
	switch d.tab() {
	case tabFiles:
		return d.files
	case tabPeers:
		return DetailsPeers(it.PeerList)
	case tabConnections:
		return DetailsConnections(it)
	case tabLog:
		return DetailsLog(d.s.DetailLog)
	}
	return giode.Stack(giode.Styles{Gap: 18, Width: -1}, DetailsGeneral(it), d.options.View(it))
}

// Layout draws the panel, or nothing when no download is focused.
func (d *DetailsPanel) Layout(gtx giode.Context) giode.Dimensions {
	it := d.s.Detail
	if !d.s.DetailsOpen || d.s.Sel.Focus == "" || it.ID == "" {
		return giode.Dimensions{}
	}
	if torrent := it.Kind == model.KindTorrent; torrent != d.torrent || it.ID != d.lastID {
		if torrent != d.torrent {
			d.setTabs(torrent)
		}
		d.lastID = it.ID
	}
	d.files.SetFiles(it.Files)
	d.options.Sync(it)
	// Leave the list room however tall the panel was dragged or the window is.
	room := int(giode.PxToDp(gtx.Constraints.Max.Y)) - listMin
	d.resize.Bounds(detailsMin, max(detailsMin, room))
	height := min(max(d.height, detailsMin), max(detailsMin, room))
	return giode.Stack(giode.Styles{Width: -1},
		d.resize,
		d.panel(it, height),
	).Layout(gtx)
}

func (d *DetailsPanel) panel(it model.Item, height int) giode.Element {
	return giode.Stack(giode.Styles{
		Width: -1, Height: properties.Height(height), Background: "theme.surface", BorderWidth: 1, BorderColor: "theme.border",
		BorderRadius: 12, Margin: giode.Inset{Left: 12, Right: 12, Bottom: 10},
	},
		giode.Row(giode.Styles{Align: giode.AlignCenter, Gap: 12, Width: -1, Wrap: giode.WrapNoWrap,
			Padding: giode.Inset{Left: 16, Right: 8, Top: 8, Bottom: 8}},
			FileBadge(it.Category, it.Kind == model.KindTorrent),
			giode.Box(giode.Styles{FlexGrow: 1},
				giode.Text(it.Name, theme.Ellipsis(theme.Strong(theme.SizeTitle)))),
			d.tabs,
			d.close,
		),
		giode.Divider(giode.Styles{Color: "theme.border"}),
		giode.Box(giode.Styles{FlexGrow: 1, Width: -1, Padding: giode.UniformInset(16)}, d.scroll),
	)
}
