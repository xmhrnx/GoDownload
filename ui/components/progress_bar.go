package components

import (
	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/styles/properties"

	"github.com/godownload/godownload/ui/theme"
)

// ProgressBar draws progress as a thin bar. With a block map it shows which
// parts of the file are on disk, so parallel connections and torrent pieces
// are visible as they fill in.
func ProgressBar(fraction float64, blocks []float32, color giode.Color, height int) giode.Element {
	return giode.Canvas(giode.Styles{Width: -1, Height: properties.Height(height)}, func(p *giode.Painter) {
		w, h := p.Width(), p.Height()
		p.Rect(0, 0, w, h, h/2, theme.Track)
		if len(blocks) > 1 && fraction < 1 {
			drawBlocks(p, blocks, color)
			return
		}
		if fraction > 0 {
			p.Rect(0, 0, max(h, w*float32(min(fraction, 1))), h, h/2, color)
		}
	})
}

func drawBlocks(p *giode.Painter, blocks []float32, color giode.Color) {
	w, h := p.Width(), p.Height()
	bw := w / float32(len(blocks))
	for i := 0; i < len(blocks); {
		if blocks[i] <= 0 {
			i++
			continue
		}
		// Merge neighbours into one run so the bar draws few shapes.
		j := i
		for j < len(blocks) && blocks[j] > 0 {
			j++
		}
		p.Rect(float32(i)*bw, 0, float32(j-i)*bw, h, min(h/2, 1.5), color)
		i = j
	}
}
