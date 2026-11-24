package dialogs

import (
	"context"
	"fmt"
	"path"
	"slices"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/categorize"
	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/internal/player"
	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// PlayDialog streams a download to a media player while it downloads. It
// asks which file when a torrent has several, and offers alternatives when
// no player is installed.
type PlayDialog struct {
	s     *state.State
	open  bool
	dlg   *giode.DialogElement
	id    string
	files []model.TorrentFile
	// choices are the playable files' indexes; picking shows them.
	choices  []int
	picking  bool
	url      string
	title    string
	fileBtns []*giode.PressableElement
	cancel   *giode.ButtonElement
	browser  *giode.ButtonElement
	copy     *giode.ButtonElement
	goplay   *giode.ButtonElement
}

// NewPlayDialog creates the dialog.
func NewPlayDialog(s *state.State) *PlayDialog {
	d := &PlayDialog{s: s}
	d.cancel = giode.Button("Close", theme.QuietButton).OnClick(func() { d.open = false })
	d.browser = giode.Button("Open in browser", theme.QuietButton).Icon(theme.Icon("globe", 15, "")).OnClick(func() {
		d.open = false
		go func() {
			err := s.App.OpenURL(context.Background(), d.url)
			s.App.Post(func() { s.Notify(err) })
		}()
	})
	d.copy = giode.Button("Copy link", theme.QuietButton).Icon(theme.Icon("copy", 15, "")).OnClick(func() {
		s.App.SetClipboard(d.url)
		s.Toasts.Show(giode.ToastInfo, "Stream link copied. Open it in any player that plays network streams.")
	})
	d.goplay = giode.Button("Get GoPlay", theme.PrimaryButton).Icon(theme.Icon("tv", 15, "#FFFFFF")).OnClick(func() {
		go func() {
			err := s.App.OpenURL(context.Background(), player.GoPlayURL)
			s.App.Post(func() { s.Notify(err) })
		}()
	})
	d.dlg = giode.Dialog(&d.open, d.content, components.DialogStyles(640))
	return d
}

// Play starts streaming download id, asking which file if needed.
func (d *PlayDialog) Play(id string) {
	d.id = id
	go func() {
		files, err := d.s.M.StreamFiles(context.Background(), id)
		d.s.App.Post(func() {
			if err != nil {
				d.s.Notify(err)
				return
			}
			d.choose(files)
		})
	}()
	d.s.Toasts.Show(giode.ToastInfo, "Preparing stream…")
}

func (d *PlayDialog) choose(files []model.TorrentFile) {
	d.files = files
	d.choices = d.choices[:0]
	for i, f := range files {
		if categorize.IsMedia(f.Path) {
			d.choices = append(d.choices, i)
		}
	}
	if len(d.choices) == 0 {
		for i := range files {
			d.choices = append(d.choices, i)
		}
	}
	// Biggest first: usually the film, not the sample.
	slices.SortFunc(d.choices, func(a, b int) int { return int(files[b].Size - files[a].Size) })
	if len(d.choices) == 1 {
		d.launch(d.choices[0])
		return
	}
	d.fileBtns = d.fileBtns[:0]
	for _, i := range d.choices {
		f := files[i]
		d.fileBtns = append(d.fileBtns, giode.Pressable(func(p giode.PressState) giode.Element {
			bg := giode.Color(theme.Raised)
			if p.Hovered {
				bg = theme.Selected
			}
			return giode.Row(giode.Styles{Gap: 10, Align: giode.AlignCenter, Width: -1, Padding: giode.SymmetricInset(12, 9),
				Background: bg, BorderRadius: 8, Wrap: giode.WrapNoWrap},
				theme.Icon("play", 14, "theme.primary"),
				giode.Box(giode.Styles{FlexGrow: 1}, giode.Text(path.Base(f.Path), theme.Ellipsis(giode.Styles{FontSize: theme.SizeBody, Color: "theme.text"}))),
				giode.Text(format.Bytes(f.Size), theme.Muted(theme.SizeSmall)),
			)
		}).OnClick(func(giode.ClickEvent) { d.launch(i) }))
	}
	d.picking = true
	d.open = true
}

// launch opens file i in the configured player, or explains the options
// when there is none.
func (d *PlayDialog) launch(i int) {
	f := d.files[i]
	d.picking = false
	d.url = d.s.Server.StreamURL(d.id, i, f.Path)
	d.title = path.Base(f.Path)
	settings := d.s.M.Settings()
	p, err := player.Resolve(settings.Player, settings.PlayerCommand)
	if err == nil {
		err = player.Launch(p, d.url, d.title)
	}
	if err != nil {
		d.open = true // show the fallbacks
		return
	}
	d.open = false
	d.s.Toasts.Show(giode.ToastSuccess, fmt.Sprintf("Playing in %s. Seek freely: the part you jump to downloads first.", p.Name))
}

func (d *PlayDialog) content() giode.Element {
	if d.picking {
		kids := make([]giode.Element, len(d.fileBtns))
		for i, b := range d.fileBtns {
			kids[i] = b
		}
		return components.DialogFrame("tv", "Which file?", "This torrent has several media files.",
			giode.Stack(giode.Styles{Gap: 6, Width: -1}, kids...), d.cancel)
	}
	body := giode.Stack(giode.Styles{Gap: 10, Width: -1},
		giode.Text("GoDownload streams "+d.title+" while it downloads, but couldn't find a media player to open it.",
			giode.Styles{FontSize: theme.SizeBody, Color: "theme.text", LineHeight: 19}),
		giode.Text("GoPlay plays these streams and shows which parts have downloaded. VLC, mpv, IINA and others work too: choose one in Settings › Streaming, or copy the link into any player.",
			giode.Styles{FontSize: theme.SizeBody, Color: "theme.muted", LineHeight: 19}),
	)
	return components.DialogFrame("tv", "No media player found", "", body, d.cancel, d.copy, d.browser, d.goplay)
}

// Layout draws the dialog while open.
func (d *PlayDialog) Layout(gtx giode.Context) giode.Dimensions { return d.dlg.Layout(gtx) }
