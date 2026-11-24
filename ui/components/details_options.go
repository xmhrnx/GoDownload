package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/state"
	"github.com/godownload/godownload/ui/theme"
)

// DetailsOptions are the settings of one download: its speed limit and
// connections, or for a torrent, seeding and its upload limit.
type DetailsOptions struct {
	s       *state.State
	id      string
	limit   *RateField
	conns   *giode.NumberInputElement
	seed    *giode.SwitchElement
	upLimit *RateField
	ratio   *giode.NumberInputElement
}

// NewDetailsOptions creates the controls once; Sync loads a download into them.
func NewDetailsOptions(s *state.State) *DetailsOptions {
	o := &DetailsOptions{s: s}
	o.limit = NewRateField(140, func(v int64) { s.Notify(s.M.SetSpeedLimit(o.id, v)) })
	o.conns = giode.Input.Number(8, giode.Styles{Width: 130}).Range(1, 32).Step(1).Decimals(0).
		OnChange(func(v float64) { s.Notify(s.M.SetConnections(o.id, int(v))) })
	o.seed = giode.Input.Switch("Seed after downloading", giode.Styles{FontSize: theme.SizeBody, Color: "theme.text"}).
		OnChange(func(on bool) { s.Notify(s.M.SetSeed(o.id, on)) })
	o.upLimit = NewRateField(140, func(v int64) { s.Notify(s.M.SetUploadLimit(o.id, v)) })
	o.ratio = giode.Input.Number(0, giode.Styles{Width: 130}).Range(0, 100).Step(0.5).Decimals(1).
		OnChange(func(v float64) {
			if it, ok := s.Item(o.id); ok && it.Torrent != nil {
				s.Notify(s.M.SetSeedGoal(o.id, v, it.Torrent.SeedMinutes))
			}
		})
	return o
}

// Sync shows it's settings when the focused download changes.
func (o *DetailsOptions) Sync(it model.Item) {
	if it.ID == o.id {
		return
	}
	o.id = it.ID
	o.limit.Set(it.SpeedLimit)
	conns := it.MaxConnections
	if conns <= 0 {
		conns = o.s.M.Settings().Connections
	}
	o.conns.SetValue(float64(conns))
	if t := it.Torrent; t != nil {
		o.seed.SetChecked(t.Seed)
		o.upLimit.Set(t.UploadLimit)
		o.ratio.SetValue(t.SeedRatio)
	}
}

// View lays out the controls that apply to it.
func (o *DetailsOptions) View(it model.Item) giode.Element {
	if it.Kind == model.KindTorrent {
		return giode.Row(giode.Styles{Gap: 20, Align: giode.AlignEnd, Width: -1},
			Labeled("Upload limit (KB/s)", o.upLimit),
			Labeled("Stop seeding at ratio (0 = never)", o.ratio),
			giode.Box(giode.Styles{Width: 240, Padding: giode.Inset{Bottom: 6}}, o.seed),
		)
	}
	return giode.Row(giode.Styles{Gap: 20, Align: giode.AlignEnd, Width: -1},
		Labeled("Speed limit (KB/s)", o.limit),
		Labeled("Connections", o.conns),
	)
}
