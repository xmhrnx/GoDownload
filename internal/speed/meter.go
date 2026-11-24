// Package speed turns growing byte counters into smoothed rates.
package speed

import (
	"math"
	"time"
)

// window is how long the smoothing remembers: about the last three seconds.
// Torrent data arrives in bursts as pieces complete, so a one-second sample
// alone jumps around even when the transfer is steady.
const window = 3 * time.Second

// Meter is the rate of a counter that only grows. The zero value is ready.
type Meter struct {
	last  int64
	at    time.Time
	rate  float64
	ready bool
}

// Update records the counter's total at now and returns the smoothed rate in
// units per second. A counter that went backwards (a new session) restarts
// the meter.
func (m *Meter) Update(total int64, now time.Time) int64 {
	if !m.ready || total < m.last {
		m.last, m.at, m.rate, m.ready = total, now, 0, true
		return 0
	}
	dt := now.Sub(m.at).Seconds()
	if dt <= 0 {
		return int64(m.rate)
	}
	sample := float64(total-m.last) / dt
	alpha := 1 - math.Exp(-dt/window.Seconds())
	m.rate += alpha * (sample - m.rate)
	m.last, m.at = total, now
	if m.rate < 1 {
		m.rate = 0
	}
	return int64(m.rate)
}

// Rate is the last computed rate.
func (m *Meter) Rate() int64 { return int64(m.rate) }
