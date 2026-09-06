package itemlist

import (
	"testing"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func itemNames(items []model.Item) []string {
	out := make([]string, len(items))
	for i := range items {
		out[i] = items[i].Name
	}
	return out
}

func assertNames(t *testing.T, items []model.Item, want []string) {
	t.Helper()
	got := itemNames(items)
	if len(got) != len(want) {
		t.Fatalf("got %d items want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("item %d=%q want %q; all=%v", i, got[i], want[i], got)
		}
	}
}

func TestSortDirectoriesFirstAndCaseInsensitiveStable(t *testing.T) {
	items := []model.Item{
		{Name: "zeta.txt"},
		{Name: "beta", IsDirectory: true},
		{Name: "Alpha.txt"},
		{Name: "alpha.txt"},
		{Name: "Alpha", IsDirectory: true},
	}
	Sort(items)
	assertNames(t, items, []string{"Alpha", "beta", "Alpha.txt", "alpha.txt", "zeta.txt"})
}

func TestSortBySizeDescendingKeepsDirectoriesFirst(t *testing.T) {
	items := []model.Item{
		{Name: "small.bin", Size: 10},
		{Name: "folder-a", IsDirectory: true, Size: 1},
		{Name: "large.bin", Size: 100},
		{Name: "folder-b", IsDirectory: true, Size: 2},
	}
	SortBy(items, Spec{Field: FieldSize, Descending: true})
	assertNames(t, items, []string{"folder-b", "folder-a", "large.bin", "small.bin"})
}

func TestSortByModifiedKeepsUnknownMetadataLast(t *testing.T) {
	older := time.Date(2025, time.January, 2, 3, 4, 0, 0, time.UTC)
	newer := older.Add(24 * time.Hour)
	base := []model.Item{
		{Name: "unknown.txt"},
		{Name: "newer.txt", Modified: newer},
		{Name: "older.txt", Modified: older},
	}

	ascending := append([]model.Item(nil), base...)
	SortBy(ascending, Spec{Field: FieldModified})
	assertNames(t, ascending, []string{"older.txt", "newer.txt", "unknown.txt"})

	descending := append([]model.Item(nil), base...)
	SortBy(descending, Spec{Field: FieldModified, Descending: true})
	assertNames(t, descending, []string{"newer.txt", "older.txt", "unknown.txt"})
}

func TestSortByPermissionsKeepsMissingMetadataLast(t *testing.T) {
	items := []model.Item{
		{Name: "none", Permissions: ""},
		{Name: "strict", Permissions: "0600"},
		{Name: "public", Permissions: "0755"},
	}
	SortBy(items, Spec{Field: FieldPermissions, Descending: true})
	assertNames(t, items, []string{"public", "strict", "none"})
}

func TestSortByTypeUsesCaseInsensitiveExtension(t *testing.T) {
	items := []model.Item{
		{Name: "z.TXT"},
		{Name: "b.jpg"},
		{Name: "plain"},
		{Name: "shortcut", IsSymlink: true},
	}
	SortBy(items, Spec{Field: FieldType})
	assertNames(t, items, []string{"plain", "b.jpg", "shortcut", "z.TXT"})
}

func TestSortInvalidFieldFallsBackToName(t *testing.T) {
	items := []model.Item{{Name: "z"}, {Name: "a"}}
	SortBy(items, Spec{Field: Field(255)})
	assertNames(t, items, []string{"a", "z"})
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
