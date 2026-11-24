package components

import (
	"github.com/xmhrnx/giode"
)

// ResizeHandle is a horizontal grip that changes the height of the panel
// below it when dragged up or down. Double-click restores the default height.
type ResizeHandle struct {
	press    *giode.PressableElement
	get      func() int
	set      func(int)
	min, max int
	start    int
}

// NewResizeHandle creates a grip for a panel whose height get and set read
// and write; done runs when a drag ends, to save the height.
func NewResizeHandle(get func() int, set func(int), done func(), defaultHeight int) *ResizeHandle {
	r := &ResizeHandle{get: get, set: set, min: 1, max: 1 << 20}
	r.press = giode.Pressable(r.content, giode.Styles{Cursor: giode.CursorNSResize, Width: -1}).
		Label("Resize details").
		DragThreshold(0).
		OnDragStart(func(giode.DragEvent) { r.start = r.get() }).
		OnDragMove(func(e giode.DragEvent) {
			// Dragging up makes the panel below taller.
			r.set(r.clamp(r.start - int(e.DeltaDp.Y)))
		}).
		OnDragEnd(func(giode.DragEvent) { done() }).
		OnDoubleClick(func(giode.ClickEvent) {
			r.set(r.clamp(defaultHeight))
			done()
		})
	return r
}

// Bounds limits the height the handle sets.
func (r *ResizeHandle) Bounds(min, max int) { r.min, r.max = min, max }

func (r *ResizeHandle) clamp(h int) int { return min(max(h, r.min), r.max) }

func (r *ResizeHandle) content(p giode.PressState) giode.Element {
	grip := giode.Color("theme.border")
	if p.Hovered || p.Dragging {
		grip = "theme.primary"
	}
	return giode.Box(giode.Styles{Width: -1, Height: 10, Align: giode.AlignCenter, Justify: giode.JustifyCenter},
		giode.Box(giode.Styles{Width: 44, Height: 4, BorderRadius: 2, Background: grip}))
}

// Layout draws the grip.
func (r *ResizeHandle) Layout(gtx giode.Context) giode.Dimensions { return r.press.Layout(gtx) }
