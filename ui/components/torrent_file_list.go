package components

import (
	"fmt"
	"path"

	"github.com/xmhrnx/giode"

	"github.com/godownload/godownload/internal/categorize"
	"github.com/godownload/godownload/internal/format"
	"github.com/godownload/godownload/internal/model"
	"github.com/godownload/godownload/ui/theme"
)

// TorrentFileList lists a torrent's files with a checkbox each, to choose
// what to download; with progress, each file also shows how much is done.
type TorrentFileList struct {
	files    []model.TorrentFile
	checks   []*giode.CheckboxElement
	progress bool
	onChange func([]model.FilePriority)
	all      *giode.ButtonElement
	none     *giode.ButtonElement
	scroll   *giode.ScrollElement
}

// NewTorrentFileList creates the list; onChange gets the new priorities.
func NewTorrentFileList(progress bool, onChange func([]model.FilePriority)) *TorrentFileList {
	l := &TorrentFileList{progress: progress, onChange: onChange}
	link := giode.Styles{Color: "theme.primary", FontSize: theme.SizeSmall, Padding: giode.SymmetricInset(6, 2)}
	l.all = giode.Button("Select all", link).Variant(giode.VariantGhost).OnClick(func() { l.setAll(true) })
	l.none = giode.Button("Select none", link).Variant(giode.VariantGhost).OnClick(func() { l.setAll(false) })
	l.scroll = giode.Scroll(l.rows, giode.Styles{ScrollBar: giode.ScrollBar{AutoHide: true, Width: 5}}).FitContent(true)
	return l
}

// SetFiles shows files. Checkboxes are kept while the file count is unchanged.
func (l *TorrentFileList) SetFiles(files []model.TorrentFile) {
	if len(files) != len(l.files) {
		l.checks = make([]*giode.CheckboxElement, len(files))
		for i, f := range files {
			cb := giode.Input.Checkbox("", giode.Styles{Background: "theme.primary"})
			cb.SetChecked(f.Priority != model.PrioritySkip)
			cb.OnChange(func(bool) { l.changed() })
			l.checks[i] = cb
		}
	}
	l.files = files
}

// Priorities is each file's priority from its checkbox.
func (l *TorrentFileList) Priorities() []model.FilePriority {
	out := make([]model.FilePriority, len(l.checks))
	for i, cb := range l.checks {
		if cb.Checked() {
			out[i] = model.PriorityNormal
			if i < len(l.files) && l.files[i].Priority == model.PriorityHigh {
				out[i] = model.PriorityHigh
			}
		}
	}
	return out
}

func (l *TorrentFileList) setAll(on bool) {
	for _, cb := range l.checks {
		cb.SetChecked(on)
	}
	l.changed()
}

func (l *TorrentFileList) changed() {
	if l.onChange != nil {
		l.onChange(l.Priorities())
	}
}

func (l *TorrentFileList) selectedSize() (n int, size int64) {
	for i, cb := range l.checks {
		if cb.Checked() {
			n++
			size += l.files[i].Size
		}
	}
	return n, size
}

func (l *TorrentFileList) rows() giode.Element {
	kids := make([]giode.Element, 0, len(l.files))
	for i, f := range l.files {
		cat := categorize.ByName(f.Path, "")
		right := []giode.Element{giode.Text(format.Bytes(f.Size), theme.Muted(theme.SizeSmall))}
		if l.progress && f.Size > 0 && f.Done >= 0 {
			right = append([]giode.Element{giode.Box(giode.Styles{Width: 90},
				ProgressBar(float64(f.Done)/float64(f.Size), nil, "theme.primary", 5))}, right...)
		}
		names := []giode.Element{giode.Text(path.Base(f.Path), theme.Ellipsis(giode.Styles{FontSize: theme.SizeBody, Color: "theme.text"}))}
		if dir := path.Dir(f.Path); dir != "." {
			names = append(names, giode.Text(dir, theme.Ellipsis(theme.Muted(theme.SizeCaption))))
		}
		kids = append(kids, giode.Row(giode.Styles{Gap: 10, Align: giode.AlignCenter, Width: -1, Wrap: giode.WrapNoWrap,
			Padding: giode.SymmetricInset(4, 5)},
			l.checks[i],
			theme.Icon(theme.CategoryIcon(cat), 15, theme.CategoryColor(cat)),
			giode.Stack(giode.Styles{FlexGrow: 1, Gap: 1}, names...),
			giode.Row(giode.Styles{Gap: 10, Align: giode.AlignCenter, Wrap: giode.WrapNoWrap}, right...),
		))
	}
	return giode.Stack(giode.Styles{Width: -1}, kids...)
}

// Layout draws the list with its select-all controls and selected total.
func (l *TorrentFileList) Layout(gtx giode.Context) giode.Dimensions {
	n, size := l.selectedSize()
	return giode.Stack(giode.Styles{Gap: 6, Width: -1},
		giode.Row(giode.Styles{Align: giode.AlignCenter, Width: -1, Wrap: giode.WrapNoWrap},
			giode.Text(fmt.Sprintf("%d of %d files · %s", n, len(l.files), format.Bytes(size)), theme.Muted(theme.SizeSmall)),
			giode.Box(giode.Styles{FlexGrow: 1}),
			l.all, l.none,
		),
		giode.Box(giode.Styles{Width: -1, MaxHeight: 260}, l.scroll),
	).Layout(gtx)
}
