package listmodel

import "slices"

// Selection is the set of selected downloads, with an anchor for Shift-click
// range selection.
type Selection struct {
	ids    map[string]bool
	anchor string
	// Focus is the last clicked row, whose details the panel shows.
	Focus string
}

// NewSelection returns an empty selection.
func NewSelection() *Selection { return &Selection{ids: map[string]bool{}} }

// Has reports whether id is selected.
func (s *Selection) Has(id string) bool { return s.ids[id] }

// IDs lists the selected IDs in the order of visible.
func (s *Selection) IDs(visible []string) []string {
	var out []string
	for _, id := range visible {
		if s.ids[id] {
			out = append(out, id)
		}
	}
	return out
}

// Len is the number selected.
func (s *Selection) Len() int { return len(s.ids) }

// Click applies a click on id: plain selects only it, ctrl toggles it, shift
// selects the range from the anchor in visible order.
func (s *Selection) Click(id string, ctrl, shift bool, visible []string) {
	switch {
	case shift && s.anchor != "":
		a, b := slices.Index(visible, s.anchor), slices.Index(visible, id)
		if a < 0 || b < 0 {
			s.only(id)
			return
		}
		if a > b {
			a, b = b, a
		}
		if !ctrl {
			clear(s.ids)
		}
		for _, v := range visible[a : b+1] {
			s.ids[v] = true
		}
	case ctrl:
		if s.ids[id] {
			delete(s.ids, id)
		} else {
			s.ids[id] = true
		}
		s.anchor = id
	default:
		s.only(id)
	}
	s.Focus = id
}

// Ensure selects id alone unless it is already selected, as a right-click does.
func (s *Selection) Ensure(id string) {
	if !s.ids[id] {
		s.only(id)
	}
	s.Focus = id
}

func (s *Selection) only(id string) {
	clear(s.ids)
	s.ids[id] = true
	s.anchor = id
}

// SelectAll selects every visible row.
func (s *Selection) SelectAll(visible []string) {
	for _, id := range visible {
		s.ids[id] = true
	}
}

// Prune drops IDs no longer in the list.
func (s *Selection) Prune(exists func(string) bool) {
	for id := range s.ids {
		if !exists(id) {
			delete(s.ids, id)
		}
	}
	if s.Focus != "" && !exists(s.Focus) {
		s.Focus = ""
	}
}

// Clear selects nothing.
func (s *Selection) Clear() {
	clear(s.ids)
	s.anchor, s.Focus = "", ""
}
