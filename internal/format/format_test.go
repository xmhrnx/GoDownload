package format

import "testing"

func TestBytes(t *testing.T) {
	cases := map[int64]string{
		0:                 "0 B",
		1023:              "1023 B",
		1536:              "1.50 KB",
		10 * 1024 * 1024:  "10.0 MB",
		700 * 1024 * 1024: "700 MB",
		-1:                "—",
	}
	for in, want := range cases {
		if got := Bytes(in); got != want {
			t.Errorf("Bytes(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestETA(t *testing.T) {
	cases := map[int64]string{42: "42s", 192: "3m 12s", 3900: "1h 05m", 90000: "1d 01h", -1: "∞"}
	for in, want := range cases {
		if got := ETA(in); got != want {
			t.Errorf("ETA(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestParseRate(t *testing.T) {
	cases := map[string]int64{"": 0, "500": 500 * 1024, "1.5M": 1572864, "2 MB/s": 2 << 20, "64k": 64 << 10}
	for in, want := range cases {
		got, err := ParseRate(in)
		if err != nil || got != want {
			t.Errorf("ParseRate(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	if _, err := ParseRate("fast"); err == nil {
		t.Error("ParseRate accepted garbage")
	}
}

func TestParseClock(t *testing.T) {
	if m, err := ParseClock("23:15"); err != nil || m != 23*60+15 {
		t.Fatalf("ParseClock = %d, %v", m, err)
	}
	if _, err := ParseClock("25:00"); err == nil {
		t.Fatal("accepted 25:00")
	}
}
