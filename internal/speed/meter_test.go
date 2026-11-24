package speed

import (
	"testing"
	"time"
)

func TestSteadyRate(t *testing.T) {
	var m Meter
	start := time.Unix(0, 0)
	var got int64
	for i := 0; i <= 20; i++ {
		got = m.Update(int64(i)*1000, start.Add(time.Duration(i)*time.Second))
	}
	if got < 990 || got > 1000 {
		t.Fatalf("steady 1000/s measured as %d", got)
	}
}

func TestBurstsAreSmoothed(t *testing.T) {
	var m Meter
	start := time.Unix(0, 0)
	total := int64(0)
	lo, hi := int64(1<<62), int64(0)
	for i := 0; i <= 40; i++ {
		if i%2 == 0 {
			total += 2000 // 2000 every other second: 1000/s on average
		}
		r := m.Update(total, start.Add(time.Duration(i)*time.Second))
		if i > 20 {
			lo, hi = min(lo, r), max(hi, r)
		}
	}
	if lo < 600 || hi > 1400 {
		t.Fatalf("bursty 1000/s swings between %d and %d", lo, hi)
	}
}

func TestRestart(t *testing.T) {
	var m Meter
	now := time.Unix(0, 0)
	m.Update(5000, now)
	if r := m.Update(10, now.Add(time.Second)); r != 0 {
		t.Fatalf("counter reset gave %d", r)
	}
}
