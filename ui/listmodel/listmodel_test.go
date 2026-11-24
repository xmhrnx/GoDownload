package listmodel

import (
	"reflect"
	"testing"
	"time"

	"github.com/godownload/godownload/internal/model"
)

func item(id string, added int, st model.Status, cat model.Category, size int64) model.Item {
	return model.Item{Download: model.Download{ID: id, Name: id, Status: st, Category: cat, Size: size,
		AddedAt: time.Unix(int64(added), 0)}}
}

func ids(items []model.Item) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.ID)
	}
	return out
}

func TestApply(t *testing.T) {
	items := []model.Item{
		item("a", 1, model.StatusCompleted, model.CategoryVideo, 30),
		item("b", 2, model.StatusDownloading, model.CategoryMusic, 10),
		item("c", 3, model.StatusSeeding, model.CategoryVideo, 20),
	}
	if got := ids(Apply(items, Filter{}, DefaultSort)); !reflect.DeepEqual(got, []string{"c", "b", "a"}) {
		t.Errorf("default order = %v", got)
	}
	if got := ids(Apply(items, Filter{Category: model.CategoryVideo}, Sort{Key: BySize})); !reflect.DeepEqual(got, []string{"c", "a"}) {
		t.Errorf("video by size = %v", got)
	}
	if got := ids(Apply(items, Filter{Status: ShowCompleted}, DefaultSort)); !reflect.DeepEqual(got, []string{"c", "a"}) {
		t.Errorf("completed = %v", got)
	}
	if got := ids(Apply(items, Filter{Query: "B"}, DefaultSort)); !reflect.DeepEqual(got, []string{"b"}) {
		t.Errorf("search = %v", got)
	}
	c := Count(items)
	if c.Category[model.CategoryVideo] != 2 || c.Status[ShowSeeding] != 1 || c.Status[ShowActive] != 1 {
		t.Errorf("counts = %+v", c)
	}
}

func TestToggle(t *testing.T) {
	s := DefaultSort.Toggle(ByName)
	if s != (Sort{Key: ByName}) {
		t.Fatalf("toggle = %+v", s)
	}
	if s = s.Toggle(ByName); !s.Desc {
		t.Fatal("second click should flip")
	}
}

func TestSelection(t *testing.T) {
	vis := []string{"a", "b", "c", "d"}
	s := NewSelection()
	s.Click("b", false, false, vis)
	s.Click("d", false, true, vis)
	if got := s.IDs(vis); !reflect.DeepEqual(got, []string{"b", "c", "d"}) {
		t.Fatalf("shift range = %v", got)
	}
	s.Click("c", true, false, vis)
	if got := s.IDs(vis); !reflect.DeepEqual(got, []string{"b", "d"}) {
		t.Fatalf("ctrl toggle = %v", got)
	}
	s.Ensure("a")
	if got := s.IDs(vis); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatalf("ensure = %v", got)
	}
}
