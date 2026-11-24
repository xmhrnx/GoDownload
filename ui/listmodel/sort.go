package listmodel

import (
	"cmp"
	"slices"
	"strings"

	"github.com/godownload/godownload/internal/model"
)

// SortKey is the column the list is sorted by.
type SortKey int

const (
	ByAdded SortKey = iota
	ByName
	BySize
	ByProgress
	BySpeed
	ByETA
	ByStatus
)

// Sort is a column and a direction.
type Sort struct {
	Key  SortKey
	Desc bool
}

// DefaultSort is newest first by time added, so rows never move on their own
// while downloads progress.
var DefaultSort = Sort{Key: ByAdded, Desc: true}

// Toggle returns the sort after clicking column k: a new column sorts
// ascending (time added: newest first), the same column flips direction.
func (s Sort) Toggle(k SortKey) Sort {
	if s.Key == k {
		return Sort{Key: k, Desc: !s.Desc}
	}
	return Sort{Key: k, Desc: k == ByAdded}
}

// Apply returns the items that pass f, sorted by s. Ties keep time-added
// order, so equal rows don't swap places between refreshes.
func Apply(items []model.Item, f Filter, s Sort) []model.Item {
	out := make([]model.Item, 0, len(items))
	for _, it := range items {
		if f.Match(it) {
			out = append(out, it)
		}
	}
	slices.SortStableFunc(out, func(a, b model.Item) int {
		c := compare(a, b, s.Key)
		if s.Desc {
			c = -c
		}
		if c == 0 {
			c = a.AddedAt.Compare(b.AddedAt)
			if s.Desc && s.Key == ByAdded {
				c = -c
			}
		}
		return c
	})
	return out
}

func progress(it model.Item) float64 {
	if it.Size <= 0 {
		return 0
	}
	return float64(it.Done) / float64(it.Size)
}

func eta(it model.Item) int64 {
	if it.ETA < 0 {
		return 1 << 62
	}
	return it.ETA
}

func compare(a, b model.Item, k SortKey) int {
	switch k {
	case ByName:
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	case BySize:
		return cmp.Compare(a.Size, b.Size)
	case ByProgress:
		return cmp.Compare(progress(a), progress(b))
	case BySpeed:
		return cmp.Compare(a.DownSpeed, b.DownSpeed)
	case ByETA:
		return cmp.Compare(eta(a), eta(b))
	case ByStatus:
		return strings.Compare(string(a.Status), string(b.Status))
	default:
		return a.AddedAt.Compare(b.AddedAt)
	}
}
