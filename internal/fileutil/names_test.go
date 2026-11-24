package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNameFromResponse(t *testing.T) {
	cases := []struct{ cd, url, want string }{
		{`attachment; filename="report.pdf"`, "https://x/y", "report.pdf"},
		{`attachment; filename*=UTF-8''r%C3%A9sum%C3%A9.pdf`, "https://x/y", "résumé.pdf"},
		{"", "https://example.com/files/My%20Movie.mkv?token=1", "My Movie.mkv"},
		{"", "https://example.com/", "example.com"},
		{`attachment; filename="../../etc/passwd"`, "https://x/y", "passwd"},
	}
	for _, c := range cases {
		if got := NameFromResponse(c.cd, c.url); got != c.want {
			t.Errorf("NameFromResponse(%q, %q) = %q, want %q", c.cd, c.url, got, c.want)
		}
	}
}

func TestSanitize(t *testing.T) {
	if got := Sanitize(`a<b>:c?.txt`); got != "a_b__c_.txt" {
		t.Errorf("Sanitize = %q", got)
	}
	if got := Sanitize("CON.txt"); got != "_CON.txt" {
		t.Errorf("Sanitize reserved = %q", got)
	}
}

func TestUniquePath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.zip"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got := UniquePath(dir, "a.zip", func(p string) bool { return p == filepath.Join(dir, "a (2).zip") })
	if want := filepath.Join(dir, "a (3).zip"); got != want {
		t.Errorf("UniquePath = %q, want %q", got, want)
	}
}

func TestWithExtension(t *testing.T) {
	if got := WithExtension("video", "video/mp4"); filepath.Ext(got) == "" {
		t.Errorf("WithExtension gave %q", got)
	}
	if got := WithExtension("a.bin", "video/mp4"); got != "a.bin" {
		t.Errorf("WithExtension changed %q", got)
	}
}

func TestMainFile(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "Sub"), 0o755)
	os.WriteFile(filepath.Join(dir, "sample.mkv"), make([]byte, 10), 0o644)
	os.WriteFile(filepath.Join(dir, "Sub", "movie.mkv"), make([]byte, 100), 0o644)
	if got, size := MainFile(dir); got != filepath.Join(dir, "Sub", "movie.mkv") || size != 100 {
		t.Fatalf("MainFile = %s, %d", got, size)
	}
}
