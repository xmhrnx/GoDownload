package ui

import (
	"time"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/links"
)

// watchClipboard offers to download links the user copies, when that
// setting is on. It checks every two seconds.
func (r *Root) watchClipboard() {
	go func() {
		for range time.Tick(2 * time.Second) {
			r.s.App.Post(func() {
				if !r.s.M.Settings().WatchClipboard {
					return
				}
				r.s.App.ReadClipboard(r.clipboardChanged)
			})
		}
	}()
}

func (r *Root) clipboardChanged(text string) {
	if text == r.lastClip {
		return
	}
	first := r.lastClip == ""
	r.lastClip = text
	found := links.Find(text)
	if first || len(found) == 0 || len(text) > 8000 {
		return // ignore what was there before startup, and big pastes
	}
	msg := "Download the copied link?"
	if len(found) > 1 {
		msg = "Download the copied links?"
	}
	r.s.Toasts.ShowAction(giode.ToastInfo, msg, "Download", func() { r.s.Actions.Add(text) })
}
