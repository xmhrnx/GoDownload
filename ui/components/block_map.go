package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/theme"
)

var blockPartial = giode.Mix("theme.primary", "theme.surface", 0.45)

// BlockMap draws the file as a strip of cells, filled where data is on disk,
// with a marker where each connection is writing. It makes multi-connection
// downloading, torrent pieces and repairs visible.
func BlockMap(it model.Item) giode.Element {
	return giode.Canvas(giode.Styles{Width: -1, Height: 22}, func(p *giode.Painter) {
		w, h := p.Width(), p.Height()
		p.Rect(0, 0, w, h, 5, theme.Track)
		blocks := it.Blocks
		if it.Status.Finished() {
			blocks = []float32{1}
		}
		if len(blocks) == 0 {
			return
		}
		bw := w / float32(len(blocks))
		for i, b := range blocks {
			switch {
			case b >= 1:
				p.Rect(float32(i)*bw, 0, bw+0.5, h, 0, theme.StatusColor(it.Status))
			case b > 0:
				p.Rect(float32(i)*bw, 0, bw+0.5, h, 0, blockPartial)
			}
		}
		if it.Size > 0 {
			for _, s := range it.Segments {
				x := float32(s.Pos) / float32(it.Size) * w
				p.Rect(x-1, 0, 2, h, 0, "theme.accent")
			}
		}
	})
}
