package categorize

import (
	"testing"

	"github.com/godownload/godownload/internal/model"
)

func TestByName(t *testing.T) {
	cases := map[string]model.Category{
		"Movie.2024.1080p.MKV": model.CategoryVideo,
		"setup.exe":            model.CategoryPrograms,
		"photo.jpeg":           model.CategoryImages,
		"thing":                model.CategoryOther,
		"album.flac":           model.CategoryMusic,
	}
	for name, want := range cases {
		if got := ByName(name, ""); got != want {
			t.Errorf("ByName(%q) = %v, want %v", name, got, want)
		}
	}
	if got := ByName("watch", "video/mp4"); got != model.CategoryVideo {
		t.Errorf("MIME fallback = %v", got)
	}
}

func TestLargest(t *testing.T) {
	got := Largest([]string{"a.mkv", "b.nfo", "c.jpg"}, []int64{1 << 30, 100, 1 << 20})
	if got != model.CategoryVideo {
		t.Errorf("Largest = %v", got)
	}
}
