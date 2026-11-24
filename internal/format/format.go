// Package format turns sizes, rates and durations into short display strings.
package format

import (
	"fmt"
	"time"
)

var units = []string{"B", "KB", "MB", "GB", "TB", "PB"}

// Bytes formats n with binary multiples: 1536 → "1.5 KB". Negative is unknown.
func Bytes(n int64) string {
	if n < 0 {
		return "—"
	}
	v := float64(n)
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	switch {
	case i == 0:
		return fmt.Sprintf("%d B", n)
	case v >= 100:
		return fmt.Sprintf("%.0f %s", v, units[i])
	case v >= 10:
		return fmt.Sprintf("%.1f %s", v, units[i])
	default:
		return fmt.Sprintf("%.2f %s", v, units[i])
	}
}

// Speed formats a rate in bytes per second.
func Speed(bps int64) string {
	if bps <= 0 {
		return ""
	}
	return Bytes(bps) + "/s"
}

// ETA formats seconds left as "1h 05m", "3m 12s" or "42s"; negative is unknown.
func ETA(seconds int64) string {
	if seconds < 0 {
		return "∞"
	}
	d := time.Duration(seconds) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	switch {
	case h >= 24:
		return fmt.Sprintf("%dd %02dh", h/24, h%24)
	case h > 0:
		return fmt.Sprintf("%dh %02dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm %02ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

// Percent formats done/total as "42.3%".
func Percent(done, total int64) string {
	if total <= 0 {
		return ""
	}
	p := float64(done) / float64(total) * 100
	if p >= 99.95 && done < total {
		p = 99.9
	}
	return fmt.Sprintf("%.1f%%", p)
}

// Ago formats how long ago t was: "just now", "5 min ago", "Yesterday", "Mar 4".
func Ago(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case sameDay(t, now):
		return t.Format("15:04")
	case sameDay(t, now.AddDate(0, 0, -1)):
		return "Yesterday"
	case t.Year() == now.Year():
		return t.Format("Jan 2")
	default:
		return t.Format("Jan 2, 2006")
	}
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// Minutes formats minutes after midnight as "HH:MM".
func Minutes(m int) string {
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}
