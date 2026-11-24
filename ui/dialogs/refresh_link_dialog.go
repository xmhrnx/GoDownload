package dialogs

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/links"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// RefreshLinkDialog replaces an expired link with a fresh one, keeping what
// is already downloaded.
type RefreshLinkDialog struct {
	s      *state.State
	open   bool
	id     string
	dlg    *giode.DialogElement
	field  *giode.InputElement
	cancel *giode.ButtonElement
	save   *giode.ButtonElement
}

// NewRefreshLinkDialog creates the dialog.
func NewRefreshLinkDialog(s *state.State) *RefreshLinkDialog {
	d := &RefreshLinkDialog{s: s}
	st := theme.Field
	st.Padding = giode.SymmetricInset(10, 8)
	d.field = giode.Input.Text("https://…", st).Keyboard(giode.KeyboardURL).Submit(func(string) { d.apply() })
	d.cancel = giode.Button("Cancel", theme.QuietButton).OnClick(func() { d.open = false })
	d.save = giode.Button("Use this link", theme.PrimaryButton).OnClick(d.apply)
	d.dlg = giode.Dialog(&d.open, d.content, components.DialogStyles(520))
	return d
}

// Open asks for a new link for download id.
func (d *RefreshLinkDialog) Open(id string) {
	d.id = id
	d.field.SetText("")
	d.open = true
	d.field.Focus()
}

func (d *RefreshLinkDialog) apply() {
	found := links.Find(d.field.Text())
	if len(found) == 0 || links.IsTorrent(found[0]) {
		d.field.Invalid("Paste the file's new download link")
		return
	}
	d.field.Invalid("")
	d.open = false
	d.s.Notify(d.s.M.UpdateURL(d.id, found[0], model.RequestHeaders{}))
	d.s.Toasts.Show(giode.ToastInfo, "Link updated; the download continues where it left off")
}

func (d *RefreshLinkDialog) content() giode.Element {
	name := ""
	if it, ok := d.s.Item(d.id); ok {
		name = it.Name
	}
	body := giode.Stack(giode.Styles{Gap: 10, Width: -1},
		giode.Text("Links from file hosts often expire. Start the download again in your browser, copy the new link, and paste it here. What's already downloaded is kept.",
			giode.Styles{FontSize: theme.SizeBody, Color: "theme.muted", LineHeight: 19}),
		d.field,
	)
	return components.DialogFrame("link", "Refresh link", name, body, d.cancel, d.save)
}

// Layout draws the dialog while open.
func (d *RefreshLinkDialog) Layout(gtx giode.Context) giode.Dimensions { return d.dlg.Layout(gtx) }
