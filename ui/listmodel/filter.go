// Package listmodel filters, sorts and selects the download list. It has no
// UI code, so it is tested on its own.
package listmodel

import (
	"strings"

	"github.com/godownload/godownload/internal/model"
)

// StatusFilter narrows the list by state.
type StatusFilter int

const (
	ShowAll StatusFilter = iota
	ShowActive
	ShowPaused
	ShowCompleted
	ShowSeeding
	ShowFailed
)

// StatusFilters lists the filters in sidebar order.
var StatusFilters = []StatusFilter{ShowActive, ShowPaused, ShowCompleted, ShowSeeding, ShowFailed}

// Title is the filter's sidebar label.
func (s StatusFilter) Title() string {
	switch s {
	case ShowActive:
		return "Downloading"
	case ShowPaused:
		return "Paused"
	case ShowCompleted:
		return "Completed"
	case ShowSeeding:
		return "Seeding"
	case ShowFailed:
		return "Failed"
	default:
		return "All downloads"
	}
}

// Filter is what the list shows.
type Filter struct {
	Category model.Category // "" for every category
	Status   StatusFilter
	Query    string
}

// Match reports whether it passes the filter.
func (f Filter) Match(it model.Item) bool {
	if f.Category != "" && it.Category != f.Category {
		return false
	}
	if !f.Status.match(it.Status) {
		return false
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		q = strings.ToLower(q)
		return strings.Contains(strings.ToLower(it.Name), q) || strings.Contains(strings.ToLower(it.URL), q)
	}
	return true
}

func (s StatusFilter) match(st model.Status) bool {
	switch s {
	case ShowActive:
		return st.Active() || st == model.StatusQueued
	case ShowPaused:
		return st == model.StatusPaused
	case ShowCompleted:
		return st == model.StatusCompleted || st == model.StatusSeeding
	case ShowSeeding:
		return st == model.StatusSeeding
	case ShowFailed:
		return st == model.StatusFailed
	default:
		return true
	}
}

// Counts is how many downloads each sidebar entry would show.
type Counts struct {
	All      int
	Category map[model.Category]int
	Status   map[StatusFilter]int
}

// Count tallies items for the sidebar.
func Count(items []model.Item) Counts {
	c := Counts{All: len(items), Category: map[model.Category]int{}, Status: map[StatusFilter]int{}}
	for _, it := range items {
		c.Category[it.Category]++
		for _, s := range StatusFilters {
			if s.match(it.Status) {
				c.Status[s]++
			}
		}
	}
	return c
}
