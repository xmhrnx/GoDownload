package components

import (
	"time"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

var (
	rowHover    = giode.Mix("theme.text", "theme.background", 0.035)
	rowSelected = theme.Selected
)

// DownloadRow is one download in the list. It is kept per download (see
// rowFor), so its press state and buttons survive between frames.
type DownloadRow struct {
	s       *state.State
	it      model.Item
	press   *giode.PressableElement
	primary *giode.IconButtonElement
	more    *giode.PressableElement
}

func newDownloadRow(s *state.State, menu *RowMenu, id string) *DownloadRow {
	r := &DownloadRow{s: s}
	r.press = giode.Pressable(r.content).
		OnPrimaryClick(func(e giode.ClickEvent) {
			s.Sel.Click(id, e.Ctrl(), e.Shift(), s.VisibleIDs())
			s.Focus(id)
		}).
		OnDoubleClick(func(giode.ClickEvent) { r.open() }).
		OnSecondaryClick(func(e giode.ClickEvent) {
			s.Sel.Ensure(id)
			s.Focus(id)
			menu.OpenAt(e)
		})
	r.primary = giode.IconButton(theme.Icon("pause", 16, ""), giode.Styles{
		Color: "theme.text", Background: "theme.text", BorderRadius: 8, Width: 30, Height: 30,
	}).Variant(giode.IconButtonGhost).OnClick(r.togglePrimary)
	r.more = giode.Pressable(func(p giode.PressState) giode.Element {
		bg := giode.Color("transparent")
		if p.Hovered {
			bg = rowHover
		}
		return giode.Box(giode.Styles{Width: 30, Height: 30, BorderRadius: 8, Background: bg,
			Align: giode.AlignCenter, Justify: giode.JustifyCenter}, theme.Icon("more", 16, "theme.muted"))
	}).Label("More").OnClick(func(e giode.ClickEvent) {
		s.Sel.Ensure(id)
		s.Focus(id)
		menu.OpenAt(e)
	})
	return r
}

// open does what a double-click means: open a finished file, play media
// while it downloads, or resume a paused one.
func (r *DownloadRow) open() {
	switch {
	case r.it.Status == model.StatusCompleted || r.it.Status == model.StatusSeeding:
		r.s.OpenFile(r.it.ID)
	case r.it.Category == model.CategoryVideo || r.it.Category == model.CategoryMusic:
		r.s.Actions.Play(r.it.ID)
	case !r.it.Status.Active():
		r.s.M.Resume(r.it.ID)
	}
}

func (r *DownloadRow) togglePrimary() {
	switch {
	case r.it.Status.Finished():
		r.s.OpenFile(r.it.ID)
	case r.it.Status.Active() || r.it.Status == model.StatusQueued && !r.it.Stopped:
		r.s.M.Pause(r.it.ID)
	default:
		r.s.M.Resume(r.it.ID)
	}
}

func (r *DownloadRow) primaryIcon() (string, string) {
	switch {
	case r.it.Status.Finished():
		return "external", "Open"
	case r.it.Status.Active() || r.it.Status == model.StatusQueued && !r.it.Stopped:
		return "pause", "Pause"
	default:
		return "play", "Resume"
	}
}

func (r *DownloadRow) content(p giode.PressState) giode.Element {
	it := r.it
	bg := giode.Color("transparent")
	switch {
	case r.s.Sel.Has(it.ID):
		bg = rowSelected
	case p.Hovered:
		bg = rowHover
	}
	fraction := 0.0
	if it.Size > 0 {
		fraction = float64(it.Done) / float64(it.Size)
	}
	if it.Status.Finished() {
		fraction = 1
	}
	return giode.Row(giode.Styles{
		Align: giode.AlignCenter, Width: -1, Wrap: giode.WrapNoWrap, Gap: 0,
		Padding: giode.SymmetricInset(rowInset, 9), Background: bg, BorderRadius: 10,
	},
		giode.Box(giode.Styles{Padding: giode.Inset{Right: 12}}, FileBadge(it.Category, it.Kind == model.KindTorrent)),
		giode.Stack(giode.Styles{FlexGrow: 1, Gap: 4, Padding: giode.Inset{Right: 12}},
			giode.Text(it.Name, theme.Ellipsis(giode.Styles{FontSize: theme.SizeBody, FontWeight: giode.FontWeightMedium, Color: "theme.text"})),
			StatusLine(it),
		),
		cell(colSize, giode.Text(sizeOrUnknown(it.Size), theme.Muted(theme.SizeSmall))),
		cell(colProgress, giode.Stack(giode.Styles{Gap: 5, Width: -1},
			ProgressBar(fraction, it.Blocks, theme.StatusColor(it.Status), 6),
			giode.Text(format.Percent(it.Done, it.Size), theme.Muted(theme.SizeCaption)),
		)),
		cell(colSpeed, speedCell(it)),
		cell(colETA, giode.Text(etaText(it), theme.Muted(theme.SizeSmall))),
		cell(colAdded, giode.Text(format.Ago(it.AddedAt, time.Now()), theme.Muted(theme.SizeSmall))),
		giode.Row(giode.Styles{Width: colActions, Gap: 2, Justify: giode.JustifyEnd, Wrap: giode.WrapNoWrap}, r.primary, r.more),
	)
}

func etaText(it model.Item) string {
	if !it.Status.Active() || it.DownSpeed <= 0 {
		return ""
	}
	return format.ETA(it.ETA)
}

// Layout draws the row.
func (r *DownloadRow) Layout(gtx giode.Context) giode.Dimensions {
	icon, label := r.primaryIcon()
	r.primary.Icon(theme.Icon(icon, 16, "")).Label(label)
	r.press.Label(r.it.Name)
	return giode.Box(giode.Styles{Padding: giode.SymmetricInset(8, 1), Width: -1}, r.press).Layout(gtx)
}
