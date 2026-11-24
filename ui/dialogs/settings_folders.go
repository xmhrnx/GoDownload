package dialogs

import (
	"path/filepath"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// folderSettings: the downloads folder and each category's folder.
type folderSettings struct {
	base *components.FolderField
	cats map[model.Category]*components.FolderField
}

func newFolderSettings(s *state.State) *folderSettings {
	f := &folderSettings{base: components.NewFolderField(s, nil), cats: map[model.Category]*components.FolderField{}}
	for _, c := range model.Categories {
		f.cats[c] = components.NewFolderField(s, nil)
	}
	return f
}

func (f *folderSettings) Title() string { return "Folders" }
func (f *folderSettings) Icon() string  { return "folder" }

func (f *folderSettings) Load(st model.Settings) {
	f.base.SetPath(st.BaseDir)
	for c, field := range f.cats {
		p := st.CategoryDirs[c]
		if p == "" {
			p = filepath.Join(st.BaseDir, c.Title())
		}
		field.SetPath(p)
	}
}

func (f *folderSettings) Apply(st *model.Settings) error {
	st.BaseDir = f.base.Path()
	st.CategoryDirs = map[model.Category]string{}
	for c, field := range f.cats {
		// Only remember folders that differ from the default, so moving the
		// base folder moves the category folders with it.
		if p := field.Path(); p != "" && p != filepath.Join(st.BaseDir, c.Title()) {
			st.CategoryDirs[c] = p
		}
	}
	return nil
}

func (f *folderSettings) View() giode.Element {
	kids := []giode.Element{
		components.Labeled("Downloads folder", f.base),
		giode.Text("Each kind of file gets its own folder inside it:", theme.Muted(theme.SizeSmall)),
	}
	for _, c := range model.Categories {
		kids = append(kids, giode.Row(giode.Styles{Gap: 10, Align: giode.AlignCenter, Width: -1, Wrap: giode.WrapNoWrap},
			giode.Row(giode.Styles{Gap: 8, Align: giode.AlignCenter, Width: 120, Wrap: giode.WrapNoWrap},
				theme.Icon(theme.CategoryIcon(c), 15, theme.CategoryColor(c)),
				giode.Text(c.Title(), giode.Styles{FontSize: theme.SizeBody, Color: "theme.text"})),
			giode.Box(giode.Styles{FlexGrow: 1}, f.cats[c]),
		))
	}
	return giode.Stack(giode.Styles{Gap: 10, Width: -1}, kids...)
}
