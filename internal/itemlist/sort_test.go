package itemlist

import (
	"testing"

	"github.com/bren-wp/by-ftp/internal/model"
)

func TestSortDirectoriesFirstAndCaseInsensitiveStable(t *testing.T) {
	items := []model.Item{
		{Name: "zeta.txt"},
		{Name: "beta", IsDirectory: true},
		{Name: "Alpha.txt"},
		{Name: "alpha.txt"},
		{Name: "Alpha", IsDirectory: true},
	}
	Sort(items)
	want := []string{"Alpha", "beta", "Alpha.txt", "alpha.txt", "zeta.txt"}
	for i := range want {
		if items[i].Name != want[i] {
			t.Fatalf("item %d=%q want %q", i, items[i].Name, want[i])
		}
	}
}

func TestSortHandlesLargeList(t *testing.T) {
	items := make([]model.Item, 50000)
	for i := range items {
		items[i].Name = string(rune('a'+(i%26))) + "-file"
		items[i].IsDirectory = i%7 == 0
	}
	Sort(items)
	seenFile := false
	for i := range items {
		if !items[i].IsDirectory {
			seenFile = true
			continue
		}
		if seenFile {
			t.Fatalf("directory found after files at index %d", i)
		}
	}
}
