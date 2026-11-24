// Package dialogs holds GoDownload's dialogs, one per file.
package dialogs

import (
	"fmt"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// RemoveDialog asks whether to take downloads off the list only, or to
// delete their files from disk too.
type RemoveDialog struct {
	s      *state.State
	open   bool
	ids    []string
	dlg    *giode.DialogElement
	cancel *giode.ButtonElement
	remove *giode.ButtonElement
	delete *giode.ButtonElement
}

// NewRemoveDialog creates the dialog.
func NewRemoveDialog(s *state.State) *RemoveDialog {
	d := &RemoveDialog{s: s}
	d.cancel = giode.Button("Cancel", theme.QuietButton).OnClick(d.close)
	d.remove = giode.Button("Remove from list", theme.QuietButton).OnClick(func() { d.run(false) })
	d.delete = giode.Button("Delete files", theme.DangerButton).Icon(theme.Icon("trash", 15, "#FFFFFF")).OnClick(func() { d.run(true) })
	d.dlg = giode.Dialog(&d.open, d.content, components.DialogStyles(460))
	return d
}

// Open asks about ids.
func (d *RemoveDialog) Open(ids []string) {
	d.ids = ids
	d.open = true
}

func (d *RemoveDialog) close() { d.open = false }

func (d *RemoveDialog) run(deleteFiles bool) {
	ids := d.ids
	d.open = false
	go func() {
		err := d.s.M.Remove(deleteFiles, ids...)
		d.s.App.Post(func() {
			d.s.Notify(err)
			if err == nil && deleteFiles {
				d.s.Toasts.Show(giode.ToastSuccess, "Deleted")
			}
		})
	}()
}

func (d *RemoveDialog) content() giode.Element {
	title := "Remove this download?"
	name := ""
	if len(d.ids) == 1 {
		if it, ok := d.s.Item(d.ids[0]); ok {
			name = it.Name
		}
	} else {
		title = fmt.Sprintf("Remove %d downloads?", len(d.ids))
	}
	body := giode.Stack(giode.Styles{Gap: 8, Width: -1},
		giode.Text(name, theme.Ellipsis(theme.Strong(theme.SizeBody))),
		giode.Text("Remove from list keeps the files on disk. Delete files also deletes what was downloaded, finished or not.",
			giode.Styles{FontSize: theme.SizeBody, Color: "theme.muted", LineHeight: 19}),
	)
	return components.DialogFrame("trash", title, "", body, d.cancel, d.remove, d.delete)
}

// Layout draws the dialog while open.
func (d *RemoveDialog) Layout(gtx giode.Context) giode.Dimensions { return d.dlg.Layout(gtx) }
