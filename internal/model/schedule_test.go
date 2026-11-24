package model

import (
	"testing"
	"time"
)

func at(day time.Weekday, h, m int) time.Time {
	// 2026-11-22 is a Sunday.
	return time.Date(2026, 11, 22+int(day), h, m, 0, 0, time.Local)
}

func TestScheduleWindow(t *testing.T) {
	night := Schedule{Enabled: true, Start: 23 * 60, Stop: 7 * 60}
	night.Days[time.Monday] = true

	cases := []struct {
		t    time.Time
		want bool
	}{
		{at(time.Monday, 22, 59), false},
		{at(time.Monday, 23, 0), true},
		{at(time.Tuesday, 6, 59), true},
		{at(time.Tuesday, 7, 0), false},
		{at(time.Tuesday, 23, 30), false},
		{at(time.Sunday, 23, 30), false},
	}
	for _, c := range cases {
		if got := night.ActiveAt(c.t); got != c.want {
			t.Errorf("ActiveAt(%v) = %v, want %v", c.t, got, c.want)
		}
	}
}

func TestScheduleOpenEnded(t *testing.T) {
	s := Schedule{Enabled: true, Start: 2 * 60, Stop: -1}
	s.Days[time.Wednesday] = true
	if !s.ActiveAt(at(time.Wednesday, 23, 0)) || s.ActiveAt(at(time.Wednesday, 1, 0)) {
		t.Fatal("open-ended schedule window is wrong")
	}
	if !s.StartsAt(at(time.Wednesday, 2, 0)) {
		t.Fatal("StartsAt missed the start minute")
	}
}
