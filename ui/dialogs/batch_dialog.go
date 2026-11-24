package dialogs

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/bt"
	"github.com/godownload/godownload/internal/categorize"
	"github.com/godownload/godownload/internal/fileutil"
	"github.com/godownload/godownload/internal/links"
	"github.com/godownload/godownload/internal/manager"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// BatchDialog picks which of many links to download, for "download all
// links on this page" from the browser.
type BatchDialog struct {
	s      *state.State
	open   bool
	dlg    *giode.DialogElement
	items  []Incoming
	checks []*giode.CheckboxElement
	filter *giode.InputElement
	scroll *giode.ScrollElement
	title  string
	cancel *giode.ButtonElement
	all    *giode.ButtonElement
	none   *giode.ButtonElement
	add    *giode.ButtonElement
}

// NewBatchDialog creates the dialog.
func NewBatchDialog(s *state.State) *BatchDialog {
	d := &BatchDialog{s: s}
	st := theme.Field
	st.Padding = giode.SymmetricInset(10, 7)
	d.filter = giode.Input.Text("Filter: zip mp4 pdf…", st).OnChange(func(string) {})
	d.scroll = giode.Scroll(d.rows, giode.Styles{ScrollBar: giode.ScrollBar{AutoHide: true, Width: 5}})
	link := giode.Styles{Color: "theme.primary", FontSize: theme.SizeSmall, Padding: giode.SymmetricInset(6, 2)}
	d.all = giode.Button("Select shown", link).Variant(giode.VariantGhost).OnClick(func() { d.setShown(true) })
	d.none = giode.Button("Clear", link).Variant(giode.VariantGhost).OnClick(func() { d.setShown(false) })
	d.cancel = giode.Button("Cancel", theme.QuietButton).OnClick(func() { d.open = false })
	d.add = giode.Button("Download selected", theme.PrimaryButton).OnClick(d.submit)
	d.dlg = giode.Dialog(&d.open, d.content, components.DialogStyles(640)).Top(60)
	return d
}

// Open shows links from a page titled title.
func (d *BatchDialog) Open(items []Incoming, title string) {
	d.items, d.title = items, title
	d.checks = make([]*giode.CheckboxElement, len(items))
	for i, it := range items {
		cb := giode.Input.Checkbox("", giode.Styles{Background: "theme.primary"})
		// Pre-select what looks like a file rather than a page.
		cb.SetChecked(links.IsTorrent(it.URL) || categorize.ByName(fileutil.NameFromURL(it.URL), "") != model.CategoryOther)
		d.checks[i] = cb
	}
	d.filter.SetText("")
	d.open = true
}

func (d *BatchDialog) shown(i int) bool {
	words := strings.Fields(strings.ToLower(d.filter.Text()))
	if len(words) == 0 {
		return true
	}
	u := strings.ToLower(d.items[i].URL)
	for _, w := range words {
		if strings.Contains(u, strings.TrimPrefix(w, ".")) {
			return true
		}
	}
	return false
}

func (d *BatchDialog) setShown(on bool) {
	for i, cb := range d.checks {
		if d.shown(i) {
			cb.SetChecked(on)
		}
	}
}

func (d *BatchDialog) submit() {
	n := 0
	for i, cb := range d.checks {
		if !cb.Checked() || !d.shown(i) {
			continue
		}
		it := d.items[i]
		r := manager.AddRequest{Kind: model.KindHTTP, URL: it.URL, Headers: it.Headers, Name: fileutil.NameFromURL(it.URL), Size: -1}
		if links.IsTorrent(it.URL) {
			src, _ := bt.ParseSource(it.URL)
			r = manager.AddRequest{Kind: model.KindTorrent, Torrent: &src, Seed: d.s.M.Settings().SeedByDefault}
		}
		if _, err := d.s.M.Add(r); err == nil {
			n++
		}
	}
	d.open = false
	d.s.Toasts.Show(giode.ToastSuccess, fmt.Sprintf("Added %d downloads", n))
}

func (d *BatchDialog) rows() giode.Element {
	var kids []giode.Element
	for i, it := range d.items {
		if !d.shown(i) {
			continue
		}
		name := fileutil.NameFromURL(it.URL)
		cat := categorize.ByName(name, "")
		kids = append(kids, giode.Row(giode.Styles{Gap: 10, Align: giode.AlignCenter, Width: -1, Wrap: giode.WrapNoWrap,
			Padding: giode.SymmetricInset(4, 4)},
			d.checks[i],
			theme.Icon(theme.CategoryIcon(cat), 15, theme.CategoryColor(cat)),
			giode.Stack(giode.Styles{FlexGrow: 1, Gap: 1},
				giode.Text(name, theme.Ellipsis(giode.Styles{FontSize: theme.SizeBody, Color: "theme.text"})),
				giode.Text(linkPlace(it.URL), theme.Ellipsis(theme.Muted(theme.SizeCaption))),
			),
		))
	}
	return giode.Stack(giode.Styles{Width: -1}, kids...)
}

func (d *BatchDialog) content() giode.Element {
	selected := 0
	for i, cb := range d.checks {
		if cb.Checked() && d.shown(i) {
			selected++
		}
	}
	body := giode.Stack(giode.Styles{Gap: 10, Width: -1},
		giode.Row(giode.Styles{Gap: 8, Align: giode.AlignCenter, Width: -1, Wrap: giode.WrapNoWrap},
			giode.Box(giode.Styles{FlexGrow: 1}, d.filter), d.all, d.none),
		giode.Box(giode.Styles{Width: -1, Height: 340}, d.scroll),
		giode.Text(fmt.Sprintf("%d selected · files go to their category folders", selected), theme.Muted(theme.SizeSmall)),
	)
	return components.DialogFrame("list-check", "Download links", d.title, body, d.cancel, d.add)
}

// linkPlace is where a link points: its host and folder.
func linkPlace(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	return u.Host + path.Dir(u.Path)
}

// Layout draws the dialog while open.
func (d *BatchDialog) Layout(gtx giode.Context) giode.Dimensions { return d.dlg.Layout(gtx) }
