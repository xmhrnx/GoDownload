package format

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseRate reads a rate typed by the user, in KB/s unless a unit is given:
// "500", "500K", "1.5M", "2 MB/s". Empty or "0" is unlimited (0).
func ParseRate(s string) (int64, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, "/S")
	s = strings.TrimSuffix(s, "B")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	mult := float64(1024)
	switch {
	case strings.HasSuffix(s, "K"):
		s = s[:len(s)-1]
	case strings.HasSuffix(s, "M"):
		mult = 1024 * 1024
		s = s[:len(s)-1]
	case strings.HasSuffix(s, "G"):
		mult = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("%q is not a speed", s)
	}
	return int64(v * mult), nil
}

// ParseClock reads "HH:MM" (or "H:MM", "HHMM") as minutes after midnight.
func ParseClock(s string) (int, error) {
	s = strings.TrimSpace(s)
	var h, m int
	var err error
	if i := strings.IndexByte(s, ':'); i >= 0 {
		h, err = strconv.Atoi(s[:i])
		if err == nil {
			m, err = strconv.Atoi(s[i+1:])
		}
	} else if len(s) == 4 {
		h, err = strconv.Atoi(s[:2])
		if err == nil {
			m, err = strconv.Atoi(s[2:])
		}
	} else {
		err = fmt.Errorf("bad time")
	}
	if err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("%q is not a time (HH:MM)", s)
	}
	return h*60 + m, nil
}
