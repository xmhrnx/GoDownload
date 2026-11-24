package state

import (
	"context"
	"os"
	"path/filepath"

	"github.com/godownload/godownload/internal/categorize"
	"github.com/godownload/godownload/internal/fileutil"
	"github.com/xmhrnx/giode"
)

// Notify shows a toast for an error, if there is one.
func (s *State) Notify(err error) {
	if err != nil {
		s.Toasts.Show(giode.ToastError, err.Error())
	}
}

// OpenFile opens a finished download in its default app. A torrent saved as
// a folder opens its main file when that's playable media, and otherwise
// shows the folder: folders can't be "opened with" an app.
func (s *State) OpenFile(id string) {
	path, err := s.M.Path(id)
	if err != nil {
		s.Notify(err)
		return
	}
	go func() {
		var err error
		if st, statErr := os.Stat(path); statErr == nil && st.IsDir() {
			if main, _ := fileutil.MainFile(path); main != "" && categorize.IsMedia(main) {
				err = s.App.OpenWith(context.Background(), main)
			} else {
				err = s.App.OpenFolder(context.Background(), path)
			}
		} else {
			err = s.App.OpenWith(context.Background(), path)
		}
		s.App.Post(func() { s.Notify(err) })
	}()
}

// OpenFolder shows the folder a download is in.
func (s *State) OpenFolder(id string) {
	path, err := s.M.Path(id)
	if err != nil {
		s.Notify(err)
		return
	}
	dir := filepath.Dir(path)
	if st, err := os.Stat(path); err == nil && st.IsDir() {
		dir = path // a torrent's folder
	}
	go func() {
		err := s.App.OpenFolder(context.Background(), dir)
		s.App.Post(func() { s.Notify(err) })
	}()
}

// CopyLink puts a download's link on the clipboard.
func (s *State) CopyLink(id string) {
	if it, ok := s.Item(id); ok {
		link := it.URL
		if it.Torrent != nil && it.Torrent.Magnet != "" {
			link = it.Torrent.Magnet
		} else if it.Torrent != nil {
			link = "magnet:?xt=urn:btih:" + it.Torrent.InfoHash
		}
		s.App.SetClipboard(link)
		s.Toasts.Show(giode.ToastInfo, "Link copied")
	}
}
