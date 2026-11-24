package player

import (
	"reflect"
	"testing"
)

func TestSplitCommand(t *testing.T) {
	got := splitCommand(`"C:\Program Files\My Player\p.exe" --title '{title}' {url}`)
	want := []string{`C:\Program Files\My Player\p.exe`, "--title", "{title}", "{url}"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitCommand = %q", got)
	}
}

func TestCustomTemplate(t *testing.T) {
	p, err := Resolve(Custom, "myplayer --name {title}")
	if err != nil {
		t.Fatal(err)
	}
	got := p.args("http://x/y", "Movie")
	if want := []string{"--name", "Movie", "http://x/y"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %q", got)
	}
}
