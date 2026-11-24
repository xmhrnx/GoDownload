package components

import (
	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/ui/theme"
)

// RateField is a text field for a speed: "500" is KB/s, "2M" MB/s, empty is
// unlimited. onSet runs on Enter or when the field loses focus.
type RateField struct {
	in    *giode.InputElement
	onSet func(bytesPerSec int64)
	last  string
}

// NewRateField creates the field.
func NewRateField(width int, onSet func(int64)) *RateField {
	f := &RateField{onSet: onSet}
	st := theme.Field
	st.Width = giodeWidth(width)
	st.Padding = giode.SymmetricInset(10, 7)
	f.in = giode.Input.Text("Unlimited", st).Keyboard(giode.KeyboardNumeric)
	f.in.Submit(func(string) { f.commit() }).OnBlur(f.commit)
	return f
}

func (f *RateField) commit() {
	text := f.in.Text()
	if text == f.last {
		return
	}
	v, err := format.ParseRate(text)
	if err != nil {
		f.in.Invalid("Use a number of KB/s, or 2M for MB/s")
		return
	}
	f.in.Invalid("")
	f.last = text
	f.onSet(v)
}

// Set shows a rate without calling onSet.
func (f *RateField) Set(bytesPerSec int64) {
	f.last = ""
	if bytesPerSec > 0 {
		if bytesPerSec%(1<<20) == 0 {
			f.last = formatInt(bytesPerSec>>20) + "M"
		} else {
			f.last = formatInt(bytesPerSec >> 10)
		}
	}
	f.in.SetText(f.last)
}

// Layout draws the field.
func (f *RateField) Layout(gtx giode.Context) giode.Dimensions { return f.in.Layout(gtx) }
