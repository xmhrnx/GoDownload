package dialogs

import (
	"fmt"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/ui/components"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// uploadPresets are one-click limits, in bytes per second; 0 is unlimited.
var uploadPresets = []int64{0, 100 << 10, 500 << 10, 1 << 20, 5 << 20}

// UploadLimitDialog sets how fast torrents may upload to other peers.
type UploadLimitDialog struct {
	s       *state.State
	open    bool
	ids     []string
	dlg     *giode.DialogElement
	field   *components.RateField
	value   int64
	presets []*giode.ButtonElement
	cancel  *giode.ButtonElement
	apply   *giode.ButtonElement
}

// NewUploadLimitDialog creates the dialog.
func NewUploadLimitDialog(s *state.State) *UploadLimitDialog {
	d := &UploadLimitDialog{s: s}
	d.field = components.NewRateField(160, func(v int64) { d.value = v })
	for _, p := range uploadPresets {
		label := "Unlimited"
		if p > 0 {
			label = format.Speed(p)
		}
		d.presets = append(d.presets, giode.Button(label, theme.QuietButton).OnClick(func() {
			d.value = p
			d.field.Set(p)
		}))
	}
	d.cancel = giode.Button("Cancel", theme.QuietButton).OnClick(func() { d.open = false })
	d.apply = giode.Button("Apply", theme.PrimaryButton).OnClick(d.submit)
	d.dlg = giode.Dialog(&d.open, d.content, components.DialogStyles(480))
	return d
}

// Open asks for the limit of the torrents among ids, starting from the
// first one's current limit.
func (d *UploadLimitDialog) Open(ids []string) {
	d.ids = nil
	d.value = 0
	for _, id := range ids {
		if it, ok := d.s.Item(id); ok && it.Torrent != nil {
			if len(d.ids) == 0 {
				d.value = it.Torrent.UploadLimit
			}
			d.ids = append(d.ids, id)
		}
	}
	if len(d.ids) == 0 {
		return
	}
	d.field.Set(d.value)
	d.open = true
}

func (d *UploadLimitDialog) submit() {
	for _, id := range d.ids {
		d.s.Notify(d.s.M.SetUploadLimit(id, d.value))
	}
	d.open = false
}

func (d *UploadLimitDialog) content() giode.Element {
	subtitle := ""
	if len(d.ids) == 1 {
		if it, ok := d.s.Item(d.ids[0]); ok {
			subtitle = it.Name
		}
	} else {
		subtitle = fmt.Sprintf("%d torrents", len(d.ids))
	}
	presets := make([]giode.Element, len(d.presets))
	for i, b := range d.presets {
		presets[i] = b
	}
	body := giode.Stack(giode.Styles{Gap: 12, Width: -1},
		giode.Row(giode.Styles{Gap: 6, Wrap: giode.WrapWrap}, presets...),
		components.Labeled("Limit (KB/s, or 2M for MB/s; empty for none)", d.field),
		giode.Text("A limited torrent shows a yellow badge in the list.", theme.Muted(theme.SizeSmall)),
	)
	return components.DialogFrame("gauge", "Upload speed limit", subtitle, body, d.cancel, d.apply)
}

// Layout draws the dialog while open.
func (d *UploadLimitDialog) Layout(gtx giode.Context) giode.Dimensions { return d.dlg.Layout(gtx) }
