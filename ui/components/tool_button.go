package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/ui/theme"
)

// ToolButton is a square icon button with a tooltip.
type ToolButton struct {
	btn *giode.IconButtonElement
	tip *giode.TooltipElement
}

// NewToolButton creates a button showing icon that runs onClick.
func NewToolButton(icon, tip string, onClick func()) *ToolButton {
	btn := giode.IconButton(theme.Icon(icon, 18, ""), giode.Styles{
		Color:        "theme.text",
		Background:   "theme.text",
		BorderRadius: 8,
		Width:        34,
		Height:       34,
	}).Variant(giode.IconButtonGhost).Label(tip).OnClick(onClick)
	return &ToolButton{btn: btn, tip: giode.Tooltip(btn, tip)}
}

// SetIcon changes the icon.
func (b *ToolButton) SetIcon(icon string) { b.btn.Icon(theme.Icon(icon, 18, "")) }

// Layout draws the button.
func (b *ToolButton) Layout(gtx giode.Context) giode.Dimensions { return b.tip.Layout(gtx) }
