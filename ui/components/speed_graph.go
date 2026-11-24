package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/ui/theme"
)

// SpeedGraph is a small card with the current speeds and a graph of the
// last minute of download speed.
func SpeedGraph(samples []float64, down, up int64) giode.Element {
	return giode.Stack(giode.Styles{
		Gap: 8, Padding: giode.UniformInset(12), Background: "theme.surface",
		BorderRadius: 10, BorderWidth: 1, BorderColor: "theme.border", Width: -1,
	},
		giode.Row(giode.Styles{Align: giode.AlignCenter, Gap: 6, Wrap: giode.WrapNoWrap},
			theme.Icon("arrow-down", 13, "theme.primary"),
			giode.Text(speedOrIdle(down), theme.Strong(theme.SizeSmall)),
			giode.Box(giode.Styles{FlexGrow: 1}),
			theme.Icon("arrow-up", 13, theme.Seed),
			giode.Text(speedOrIdle(up), theme.Muted(theme.SizeSmall)),
		),
		giode.Canvas(giode.Styles{Width: -1, Height: 36}, func(p *giode.Painter) { drawSpeeds(p, samples) }),
	)
}

// speedFill is an opaque tint: translucent fills render too bright.
var speedFill = giode.Mix("theme.primary", "theme.surface", 0.2)

func speedOrIdle(bps int64) string {
	if bps <= 0 {
		return "0 B/s"
	}
	return format.Speed(bps)
}

func drawSpeeds(p *giode.Painter, samples []float64) {
	w, h := p.Width(), p.Height()
	p.Line(0, h-0.5, w, h-0.5, 1, "theme.border")
	if len(samples) < 2 {
		return
	}
	peak := 1.0
	for _, v := range samples {
		peak = max(peak, v)
	}
	step := w / float32(59)
	x0 := w - step*float32(len(samples)-1)
	area := p.Path()
	area.MoveTo(x0, h)
	line := p.Path()
	for i, v := range samples {
		x := x0 + step*float32(i)
		y := h - 2 - float32(v/peak)*(h-4)
		area.LineTo(x, y)
		if i == 0 {
			line.MoveTo(x, y)
		} else {
			line.LineTo(x, y)
		}
	}
	area.LineTo(w, h)
	area.Close()
	area.Fill(speedFill)
	line.Stroke(1.5, "theme.primary")
}
