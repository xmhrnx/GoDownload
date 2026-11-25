package ui

import (
	"context"

	"github.com/godownload/godownload/internal/bt"
	"github.com/godownload/godownload/internal/links"
	"github.com/godownload/godownload/internal/manager"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/internal/server"
	"github.com/godownload/godownload/ui/dialogs"
)

// FromBrowser takes links the extension sent. Runs on the UI goroutine.
func (r *Root) FromBrowser(in []server.Link, batch bool) {
	r.s.App.ShowWindow()
	items := make([]dialogs.Incoming, len(in))
	for i, l := range in {
		items[i] = dialogs.Incoming{
			URL: l.URL, Filename: l.Filename,
			Headers: model.RequestHeaders{Referrer: l.Referrer, Cookies: l.Cookies, UserAgent: l.UserAgent},
		}
	}
	if batch {
		title := ""
		if len(in) > 0 {
			title = in[0].PageTitle
		}
		r.dialogs.Batch.Open(items, title)
		return
	}
	settings := r.s.M.Settings()
	if len(items) == 1 && in[0].Silent && !settings.AskWhereToSave {
		r.addQuietly(items[0])
		return
	}
	r.dialogs.Add.Review(items)
}

// addQuietly adds a link to its category folder without asking.
func (r *Root) addQuietly(in dialogs.Incoming) {
	s := r.s
	if src, ok := bt.ParseSource(in.URL); ok {
		_, err := s.M.Add(manager.AddRequest{Kind: model.KindTorrent, Torrent: &src, Seed: s.M.Settings().SeedByDefault})
		s.Notify(err)
		return
	}
	go func() {
		p, err := s.M.Probe(context.Background(), in.URL, in.Headers)
		s.App.Post(func() {
			if err != nil || p.Torrent {
				// Let the user see what failed, or pick a torrent's files.
				r.dialogs.Add.Review([]dialogs.Incoming{in})
				return
			}
			name := p.Name
			if in.Filename != "" {
				name = in.Filename
			}
			_, err := s.M.Add(manager.AddRequest{Kind: p.Kind, URL: p.URL, Headers: in.Headers, Name: name,
				Category: p.Category, Dir: p.Dir, Size: p.Size, Resumable: p.Resumable})
			s.Notify(err)
		})
	}()
}

// OpenArgs handles what the app was launched with: links, magnet links and
// .torrent files (from the file manager or a second launch).
func (r *Root) OpenArgs(args []string) {
	var items []dialogs.Incoming
	for _, a := range args {
		if l := links.Find(a); len(l) > 0 {
			items = append(items, dialogs.Incoming{URL: l[0]})
			continue
		}
		if in, ok := torrentFile(a); ok {
			items = append(items, in)
		}
	}
	if len(items) > 0 {
		r.s.App.ShowWindow()
		r.dialogs.Add.Review(items)
	}
}
