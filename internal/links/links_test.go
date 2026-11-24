package links

import (
	"reflect"
	"testing"
)

func TestFind(t *testing.T) {
	text := `Get it here: https://example.com/a.zip, or (https://cdn.example.com/b.iso).
magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=x
https://example.com/a.zip again, and 89abcdef0123456789abcdef0123456789abcdef`
	want := []string{
		"https://example.com/a.zip",
		"https://cdn.example.com/b.iso",
		"magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=x",
		"89abcdef0123456789abcdef0123456789abcdef",
	}
	if got := Find(text); !reflect.DeepEqual(got, want) {
		t.Fatalf("Find =\n%q\nwant\n%q", got, want)
	}
	if !IsTorrent(want[2]) || IsTorrent(want[0]) {
		t.Fatal("IsTorrent wrong")
	}
}
