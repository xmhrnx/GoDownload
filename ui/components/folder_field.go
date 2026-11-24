package components

import (
	"context"
	"errors"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// FolderField is a folder path with a Browse button that opens the system's
// folder picker.
type FolderField struct {
	in     *giode.InputElement
	browse *giode.ButtonElement
}

// NewFolderField creates the field; onPick runs when a folder is picked.
func NewFolderField(s *state.State, onPick func(string)) *FolderField {
	f := &FolderField{}
	st := theme.Field
	st.Padding = giode.SymmetricInset(10, 8)
	f.in = giode.Input.Text("Folder", st)
	f.browse = giode.Button("Browse…", theme.QuietButton).Icon(theme.Icon("folder", 15, "")).OnClick(func() {
		start := f.in.Text()
		go func() {
			dir, err := s.App.PickFolder(context.Background(), giode.FolderOptions{Title: "Save downloads to", Directory: start})
			s.App.Post(func() {
				if errors.Is(err, giode.ErrCanceled) {
					return
				}
				if err != nil {
					s.Notify(err)
					return
				}
				f.in.SetText(dir)
				if onPick != nil {
					onPick(dir)
				}
			})
		}()
	})
	return f
}

// Path is the folder typed or picked.
func (f *FolderField) Path() string { return f.in.Text() }

// SetPath shows a folder.
func (f *FolderField) SetPath(p string) { f.in.SetText(p) }

// Layout draws the field and button.
func (f *FolderField) Layout(gtx giode.Context) giode.Dimensions {
	return giode.Row(giode.Styles{Gap: 8, Align: giode.AlignCenter, Width: -1, Wrap: giode.WrapNoWrap},
		giode.Box(giode.Styles{FlexGrow: 1}, f.in), f.browse,
	).Layout(gtx)
}
