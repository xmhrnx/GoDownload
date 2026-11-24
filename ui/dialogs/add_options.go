package dialogs

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/manager"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// AddOptions are the add dialog's choices: where to save, the category, how
// many connections, a checksum, when to start and whether to seed.
type AddOptions struct {
	s        *state.State
	folder   *components.FolderField
	category *giode.SelectElement
	conns    *giode.NumberInputElement
	checksum *giode.InputElement
	start    *giode.SegmentedElement
	seed     *giode.SwitchElement
	// picked is set once the user chooses a folder, which then stays put.
	picked bool
	// auto is the category guessed from the links.
	auto model.Category
}

// NewAddOptions creates the controls.
func NewAddOptions(s *state.State) *AddOptions {
	o := &AddOptions{s: s}
	o.folder = components.NewFolderField(s, func(string) { o.picked = true })
	opts := []string{"Automatic"}
	for _, c := range model.Categories {
		opts = append(opts, c.Title())
	}
	o.category = giode.Input.Select(theme.Field, opts...).OnChange(func(int) { o.suggestFolder() })
	o.conns = giode.Input.Number(8, giode.Styles{Width: 130}).Range(1, 32).Step(1).Decimals(0)
	st := theme.Field
	st.Padding = giode.SymmetricInset(10, 8)
	o.checksum = giode.Input.Text("Optional: SHA-256, SHA-1 or MD5 to verify the file", st)
	o.start = giode.Input.Segmented("Start now", "Add paused", "On schedule").Styles(giode.Styles{FontSize: theme.SizeSmall})
	o.seed = giode.Input.Switch("Seed after downloading", giode.Styles{FontSize: theme.SizeBody, Color: "theme.text"})
	return o
}

// Reset prepares the options for a new set of links.
func (o *AddOptions) Reset() {
	st := o.s.M.Settings()
	o.picked = false
	o.auto = ""
	o.category.SetSelected(0)
	o.conns.SetValue(float64(st.Connections))
	o.checksum.SetText("")
	o.start.SetSelected(0)
	o.seed.SetChecked(st.SeedByDefault)
	o.folder.SetPath(st.BaseDir)
}

// SetAuto records the category the links look like and updates the folder.
func (o *AddOptions) SetAuto(c model.Category) {
	o.auto = c
	o.suggestFolder()
}

// Category is the chosen category, or the guessed one on Automatic.
func (o *AddOptions) Category(guess model.Category) model.Category {
	if i := o.category.Selected(); i > 0 {
		return model.Categories[i-1]
	}
	return guess
}

func (o *AddOptions) suggestFolder() {
	if o.picked {
		return
	}
	c := o.Category(o.auto)
	if c == "" {
		o.folder.SetPath(o.s.M.Settings().BaseDir)
		return
	}
	o.folder.SetPath(o.s.M.DirFor(c))
}

// Dir is where a download of category c goes: the picked folder, or that
// category's folder when several kinds of files are added at once.
func (o *AddOptions) Dir(c model.Category) string {
	if o.picked || o.auto != "" || o.category.Selected() > 0 {
		return o.folder.Path()
	}
	return o.s.M.DirFor(c)
}

// StartMode is when the downloads begin.
func (o *AddOptions) StartMode() manager.StartMode { return manager.StartMode(o.start.Selected()) }

// View lays out the options; checksum only for a single file, connections
// only for web downloads, seeding only for torrents.
func (o *AddOptions) View(single, hasHTTP, hasTorrent bool) giode.Element {
	row := []giode.Element{giode.Box(giode.Styles{Width: 190}, components.Labeled("Category", o.category))}
	if hasHTTP {
		row = append(row, components.Labeled("Connections", o.conns))
	}
	if hasTorrent {
		row = append(row, giode.Box(giode.Styles{Width: 240, Padding: giode.Inset{Top: 18}}, o.seed))
	}
	kids := []giode.Element{
		components.Labeled("Save to", o.folder),
		giode.Row(giode.Styles{Gap: 16, Align: giode.AlignStart, Width: -1}, row...),
	}
	if single && hasHTTP {
		kids = append(kids, components.Labeled("Checksum", o.checksum))
	}
	kids = append(kids, components.Labeled("When", o.start))
	return giode.Stack(giode.Styles{Gap: 14, Width: -1}, kids...)
}
