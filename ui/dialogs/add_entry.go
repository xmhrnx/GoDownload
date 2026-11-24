package dialogs

import (
	"context"

	"github.com/godownload/godownload/internal/bt"
	"github.com/godownload/godownload/internal/categorize"
	"github.com/godownload/godownload/internal/fileutil"
	"github.com/godownload/godownload/internal/links"
	"github.com/godownload/godownload/internal/manager"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/state"
)

// Incoming is a link to add, with what the browser knew about it.
type Incoming struct {
	URL      string
	Filename string
	Headers  model.RequestHeaders
	// MetaInfo is a .torrent file's contents, for torrents picked from disk.
	MetaInfo []byte
}

// addEntry is one link in the add dialog, probed in the background.
type addEntry struct {
	in      Incoming
	torrent *bt.Source
	probing bool
	err     error
	probed  manager.Probed
	meta    bt.Meta
	files   *components.TorrentFileList
}

func newAddEntry(in Incoming) *addEntry {
	e := &addEntry{in: in, probing: true}
	switch {
	case len(in.MetaInfo) > 0:
		e.torrent = &bt.Source{MetaInfo: in.MetaInfo}
	case links.IsTorrent(in.URL):
		src, _ := bt.ParseSource(in.URL)
		e.torrent = &src
	}
	if e.torrent != nil {
		e.files = components.NewTorrentFileList(false, nil)
	}
	return e
}

// probe asks the server or the swarm about the entry; done runs on the UI
// goroutine with the result applied.
func (e *addEntry) probe(ctx context.Context, s *state.State, done func()) {
	go func() {
		if e.torrent != nil {
			meta, err := s.M.InspectTorrent(ctx, *e.torrent)
			s.App.Post(func() {
				e.probing, e.meta, e.err = false, meta, err
				if err == nil {
					e.files.SetFiles(meta.Files)
				}
				done()
			})
			return
		}
		p, err := s.M.Probe(ctx, e.in.URL, e.in.Headers)
		s.App.Post(func() {
			e.probing, e.probed, e.err = false, p, err
			if e.in.Filename != "" && err == nil {
				e.probed.Name = fileutil.Sanitize(e.in.Filename)
				e.probed.Category = categorize.ByName(e.probed.Name, "")
			}
			done()
		})
	}()
}

// name is the file or torrent name to show.
func (e *addEntry) name() string {
	switch {
	case e.torrent != nil && e.meta.Name != "":
		return e.meta.Name
	case e.torrent != nil:
		return "Torrent"
	case e.probed.Name != "":
		return e.probed.Name
	case e.in.Filename != "":
		return e.in.Filename
	}
	return fileutil.NameFromURL(e.in.URL)
}

// category is the entry's file type, once known.
func (e *addEntry) category() model.Category {
	if e.torrent != nil {
		names := make([]string, len(e.meta.Files))
		sizes := make([]int64, len(e.meta.Files))
		for i, f := range e.meta.Files {
			names[i], sizes[i] = f.Path, f.Size
		}
		return categorize.Largest(names, sizes)
	}
	if e.probed.Category != "" {
		return e.probed.Category
	}
	return categorize.ByName(e.name(), "")
}

func (e *addEntry) size() int64 {
	if e.torrent != nil {
		if e.meta.Name == "" {
			return -1
		}
		return e.meta.Size
	}
	if e.err != nil || e.probing {
		return -1
	}
	return e.probed.Size
}
