package theme

import (
	"github.com/xmhrnx/giode"
	"github.com/xmhrnx/giode/styles/properties"
)

// Type scale.
const (
	SizeCaption = 11
	SizeSmall   = 12
	SizeBody    = 13
	SizeTitle   = 15
	SizeHeading = 20
)

// Muted is secondary text.
func Muted(size int) giode.Styles {
	return giode.Styles{Color: "theme.muted", FontSize: properties.FontSize(size)}
}

// Strong is emphasized text.
func Strong(size int) giode.Styles {
	return giode.Styles{Color: "theme.text", FontSize: properties.FontSize(size), FontWeight: giode.FontWeightSemiBold}
}

// Ellipsis keeps text on one line, ending in "…".
func Ellipsis(st giode.Styles) giode.Styles {
	st.MaxLines = 1
	st.TextOverflow = giode.TextOverflowEllipsis
	return st
}

// Card is a raised panel.
var Card = giode.Styles{
	Background:   "theme.surface",
	BorderRadius: 12,
	BorderWidth:  1,
	BorderColor:  "theme.border",
	Padding:      giode.UniformInset(16),
}

// Field is the look of text fields and selects.
var Field = giode.Styles{
	Background:   "theme.background",
	BorderColor:  "theme.border",
	BorderRadius: 8,
	FontSize:     SizeBody,
	Color:        "theme.text",
}

// PrimaryButton is the main action's button.
var PrimaryButton = giode.Styles{
	BackgroundGradient: giode.Linear(120, "#7C5CFA", "#4F6BF2"),
	Color:              "#FFFFFF",
	BorderRadius:       8,
	FontSize:           SizeBody,
	FontWeight:         giode.FontWeightSemiBold,
	Padding:            giode.SymmetricInset(16, 9),
}

// QuietButton is a secondary action's button.
var QuietButton = giode.Styles{
	Background:   "theme.raised",
	Color:        "theme.text",
	BorderRadius: 8,
	FontSize:     SizeBody,
	FontWeight:   giode.FontWeightMedium,
	Padding:      giode.SymmetricInset(14, 9),
}

// DangerButton confirms something destructive.
var DangerButton = giode.Styles{
	Background:   "theme.danger",
	Color:        "#FFFFFF",
	BorderRadius: 8,
	FontSize:     SizeBody,
	FontWeight:   giode.FontWeightSemiBold,
	Padding:      giode.SymmetricInset(16, 9),
}

// Brand is the logo gradient.
var Brand = giode.Linear(135, "#7B5CFA", "#4F6BF2", "#19C6E6")
