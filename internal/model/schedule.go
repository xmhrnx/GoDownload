package model

import "time"

// Schedule starts the queue at a time of day and optionally stops it later.
type Schedule struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	// Start and Stop are minutes after midnight; Stop < 0 means never stop.
	Start int `json:"start"`
	Stop  int `json:"stop"`
	// Days are the weekdays it runs on, indexed by time.Weekday.
	Days [7]bool `json:"days"`
	// MaxActive overrides the number of simultaneous downloads while it runs; 0 keeps the setting.
	MaxActive int `json:"maxActive,omitempty"`
	// SpeedLimit caps all downloads while it runs, in bytes per second; 0 keeps the setting.
	SpeedLimit int64 `json:"speedLimit,omitempty"`
	// WhenDone runs once the queue empties during the window, or at Stop.
	WhenDone PowerAction `json:"whenDone,omitempty"`
}

// ActiveAt reports whether t falls inside the schedule's window.
// Windows that cross midnight belong to the day they start on.
func (s Schedule) ActiveAt(t time.Time) bool {
	if !s.Enabled {
		return false
	}
	minute := t.Hour()*60 + t.Minute()
	today := t.Weekday()
	yesterday := (today + 6) % 7
	switch {
	case s.Stop < 0:
		return s.Days[today] && minute >= s.Start
	case s.Stop > s.Start:
		return s.Days[today] && minute >= s.Start && minute < s.Stop
	default: // crosses midnight
		return (s.Days[today] && minute >= s.Start) || (s.Days[yesterday] && minute < s.Stop)
	}
}

// StartsAt reports whether t is the schedule's start minute.
func (s Schedule) StartsAt(t time.Time) bool {
	return s.Enabled && s.Days[t.Weekday()] && t.Hour()*60+t.Minute() == s.Start
}
