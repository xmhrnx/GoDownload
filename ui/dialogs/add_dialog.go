package dialogs

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/links"
	"github.com/godownload/godownload/internal/manager"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// AddDialog adds downloads: it takes pasted links (web, magnet, info hash)
// or a .torrent file, checks each, and lets the user choose names, files,
// folder and when to start.
type AddDialog struct {
	s       *state.State
	open    bool
	dlg     *giode.DialogElement
	text    *giode.InputElement
	name    *giode.InputElement
	entries []*addEntry
	// autoName is the name last filled in, so a name the user typed
	// isn't overwritten when a check finishes.
	autoName string
	opts     *AddOptions
	list     *giode.ScrollElement
	ctx      context.Context
	stop     context.CancelFunc

	cancel  *giode.ButtonElement
	next    *giode.ButtonElement
	torrent *giode.ButtonElement
	add     *giode.ButtonElement
}

// NewAddDialog creates the dialog.
func NewAddDialog(s *state.State) *AddDialog {
	d := &AddDialog{s: s, opts: NewAddOptions(s)}
	st := theme.Field
	st.Padding = giode.UniformInset(12)
	d.text = giode.Input.Text("Paste links, magnet links or info hashes — one per line", st).Lines(5, 10)
	nst := theme.Field
	nst.Padding = giode.SymmetricInset(10, 6)
	d.name = giode.Input.Text("File name", nst)
	d.list = giode.Scroll(d.entryCards, giode.Styles{ScrollBar: giode.ScrollBar{AutoHide: true, Width: 5}}).FitContent(true)
	d.cancel = giode.Button("Cancel", theme.QuietButton).OnClick(d.close)
	d.next = giode.Button("Continue", theme.PrimaryButton).OnClick(d.continueWithText)
	d.torrent = giode.Button("Open .torrent…", theme.QuietButton).Icon(theme.Icon("magnet", 15, "")).OnClick(d.PickTorrent)
	d.add = giode.Button("Download", theme.PrimaryButton).Icon(theme.Icon("download", 15, "#FFFFFF")).OnClick(d.submit)
	d.dlg = giode.Dialog(&d.open, d.content, components.DialogStyles(600)).Top(70)
	return d
}

// Open shows the dialog with text (links) filled in; with links found it goes
// straight to reviewing them.
func (d *AddDialog) Open(text string) {
	found := links.Find(text)
	if len(found) == 0 {
		d.reset()
		d.text.SetText(text)
		d.open = true
		d.text.Focus()
		return
	}
	in := make([]Incoming, len(found))
	for i, l := range found {
		in[i] = Incoming{URL: l}
	}
	d.Review(in)
}

// Review shows links (from the browser, a paste or a .torrent file) for the
// user to confirm.
func (d *AddDialog) Review(in []Incoming) {
	d.reset()
	d.open = true
	for _, i := range in {
		e := newAddEntry(i)
		d.entries = append(d.entries, e)
		e.probe(d.ctx, d.s, d.probed)
	}
	if len(d.entries) == 1 {
		d.autoName = d.entries[0].name()
		d.name.SetText(d.autoName)
	}
}

// PickTorrent opens a .torrent file from disk.
func (d *AddDialog) PickTorrent() {
	go func() {
		files, err := d.s.App.PickFiles(context.Background(), giode.PickOptions{Types: []string{"torrent", "application/x-bittorrent"}, Multiple: true})
		var in []Incoming
		for _, f := range files {
			data, rerr := io.ReadAll(io.LimitReader(f, 32<<20))
			f.Close()
			if rerr != nil {
				err = rerr
				break
			}
			in = append(in, Incoming{URL: f.Name, MetaInfo: data})
		}
		d.s.App.Post(func() {
			if errors.Is(err, giode.ErrCanceled) {
				return
			}
			if err != nil {
				d.s.Notify(err)
				return
			}
			if len(in) > 0 {
				d.Review(in)
			}
		})
	}()
}

func (d *AddDialog) reset() {
	if d.stop != nil {
		d.stop()
	}
	d.ctx, d.stop = context.WithCancel(context.Background())
	d.entries = nil
	d.autoName = ""
	d.opts.Reset()
}

func (d *AddDialog) close() {
	d.open = false
	if d.stop != nil {
		d.stop()
	}
}

func (d *AddDialog) continueWithText() {
	if len(links.Find(d.text.Text())) == 0 {
		d.text.Invalid("No links found. Paste a web address, a magnet link or an info hash.")
		return
	}
	d.text.Invalid("")
	d.Open(d.text.Text())
}

// probed runs when an entry's check finishes: once all agree on a category,
// the folder follows it.
func (d *AddDialog) probed() {
	if len(d.entries) == 1 && d.name.Text() == d.autoName {
		d.autoName = d.entries[0].name()
		d.name.SetText(d.autoName)
	}
	var cat model.Category
	for _, e := range d.entries {
		if e.probing {
			return
		}
		c := e.category()
		if cat != "" && c != cat {
			d.opts.SetAuto("")
			return
		}
		cat = c
	}
	d.opts.SetAuto(cat)
}

func (d *AddDialog) submit() {
	mode := d.opts.StartMode()
	settings := d.s.M.Settings()
	added, dup := 0, 0
	for _, e := range d.entries {
		cat := d.opts.Category(e.category())
		r := manager.AddRequest{
			Name: e.name(), Category: cat, Dir: d.opts.Dir(cat), Size: e.size(), Start: mode,
		}
		if len(d.entries) == 1 && d.name.Text() != "" {
			r.Name = d.name.Text()
		}
		if e.torrent != nil {
			r.Kind, r.Torrent = model.KindTorrent, e.torrent
			r.Seed, r.SeedRatio = d.opts.seed.Checked(), settings.SeedRatio
			if e.meta.Name != "" {
				r.Files = e.files.Priorities()
			}
		} else {
			r.Kind, r.URL, r.Headers = model.KindHTTP, e.in.URL, e.in.Headers
			if e.err == nil {
				r.Kind, r.URL, r.Resumable = e.probed.Kind, e.probed.URL, e.probed.Resumable
			}
			r.Connections = int(d.opts.conns.Value())
			if len(d.entries) == 1 {
				sum, err := manager.ParseChecksum(d.opts.checksum.Text())
				if err != nil {
					d.opts.checksum.Invalid(err.Error())
					return
				}
				r.Checksum = sum
			}
		}
		_, err := d.s.M.Add(r)
		switch {
		case errors.Is(err, manager.ErrDuplicate):
			dup++
		case err != nil:
			d.s.Notify(err)
		default:
			added++
		}
	}
	d.close()
	if dup > 0 {
		d.s.Toasts.Show(giode.ToastWarning, fmt.Sprintf("%d already in the list", dup))
	}
	if added > 0 && mode == manager.StartScheduled {
		d.s.Toasts.Show(giode.ToastInfo, "Added to the schedule. Set times in the Scheduler.")
	}
}

func (d *AddDialog) entryCards() giode.Element {
	kids := make([]giode.Element, 0, len(d.entries))
	for _, e := range d.entries {
		var nameField giode.Element
		if len(d.entries) == 1 {
			nameField = d.name
		}
		kids = append(kids, addEntryCard(e, nameField))
	}
	return giode.Stack(giode.Styles{Gap: 8, Width: -1}, kids...)
}

func (d *AddDialog) content() giode.Element {
	if len(d.entries) == 0 {
		body := giode.Stack(giode.Styles{Gap: 10, Width: -1},
			d.text,
			giode.Text("Tip: copy a link anywhere and press Ctrl+V in GoDownload, or let the browser extension send downloads here.",
				theme.Muted(theme.SizeSmall)),
		)
		return components.DialogFrame("plus", "New download", "", body, d.torrent, d.cancel, d.next)
	}
	hasHTTP, hasTorrent := false, false
	var torrentFiles giode.Element
	for _, e := range d.entries {
		if e.torrent != nil {
			hasTorrent = true
			if len(d.entries) == 1 && e.meta.Name != "" {
				torrentFiles = e.files
			}
		} else {
			hasHTTP = true
		}
	}
	kids := []giode.Element{giode.Box(giode.Styles{Width: -1, MaxHeight: 240}, d.list)}
	if torrentFiles != nil {
		kids = append(kids, torrentFiles)
	}
	kids = append(kids, giode.Divider(giode.Styles{Color: "theme.border"}), d.opts.View(len(d.entries) == 1, hasHTTP, hasTorrent))
	subtitle := ""
	if len(d.entries) > 1 {
		subtitle = fmt.Sprintf("%d downloads", len(d.entries))
	}
	return components.DialogFrame("download", "New download", subtitle,
		giode.Stack(giode.Styles{Gap: 14, Width: -1}, kids...), d.cancel, d.add)
}

// Layout draws the dialog while open.
func (d *AddDialog) Layout(gtx giode.Context) giode.Dimensions { return d.dlg.Layout(gtx) }
