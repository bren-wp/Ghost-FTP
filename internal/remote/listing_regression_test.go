package remote

import (
	"fmt"
	"testing"

	"github.com/bren-wp/by-ftp/internal/model"
)

func TestParseListLineKeepsArrowInRegularFilename(t *testing.T) {
	item, ok := parseListLine("-rw-r--r-- 1 user group 42 Aug 18 11:30 izvjestaj -> final.txt")
	if !ok {
		t.Fatal("regular file listing was not recognized")
	}
	if item.IsSymlink {
		t.Fatal("regular file was incorrectly classified as symlink")
	}
	if item.Name != "izvjestaj -> final.txt" {
		t.Fatalf("name=%q want %q", item.Name, "izvjestaj -> final.txt")
	}
	if item.Size != 42 {
		t.Fatalf("size=%d want 42", item.Size)
	}
}

func TestParseListLineTrimsOnlySymlinkTarget(t *testing.T) {
	item, ok := parseListLine("lrwxrwxrwx 1 user group 12 Aug 18 11:30 link name -> target.txt")
	if !ok || !item.IsSymlink {
		t.Fatalf("symlink listing was not recognized: %#v ok=%v", item, ok)
	}
	if item.Name != "link name" {
		t.Fatalf("symlink name=%q want %q", item.Name, "link name")
	}
}

func TestParseListLineRejectsOverflowingSizeWithoutWraparound(t *testing.T) {
	item, ok := parseListLine("-rw-r--r-- 1 user group 999999999999999999999999999999 Aug 18 11:30 huge.bin")
	if !ok {
		t.Fatal("listing was not recognized")
	}
	if item.Size != 0 {
		t.Fatalf("overflowing size wrapped or leaked through: %d", item.Size)
	}
}

func TestParseDOSListLineReadsFileSize(t *testing.T) {
	item, ok := parseListLine("08-18-26  11:30AM  123456 report final.zip")
	if !ok || item.IsDirectory {
		t.Fatalf("DOS file listing was not recognized: %#v ok=%v", item, ok)
	}
	if item.Name != "report final.zip" || item.Size != 123456 {
		t.Fatalf("unexpected DOS file parse: %#v", item)
	}
}

func TestSortItemsUsesSharedStableOrderingAtDirectoryLimit(t *testing.T) {
	items := make([]model.Item, 0, maxDirectoryItems)
	for i := maxDirectoryItems - 1; i >= 0; i-- {
		items = append(items, model.Item{Name: fmt.Sprintf("file-%05d.txt", i)})
	}
	items[len(items)-1].IsDirectory = true
	items[len(items)-1].Name = "Mapa"
	sortItems(items)
	if len(items) != maxDirectoryItems {
		t.Fatalf("item count changed: %d", len(items))
	}
	if !items[0].IsDirectory || items[0].Name != "Mapa" {
		t.Fatalf("directory was not sorted first: %#v", items[0])
	}
	if items[1].Name != "file-00001.txt" || items[len(items)-1].Name != "file-49999.txt" {
		t.Fatalf("unexpected stable ordering: first=%q last=%q", items[1].Name, items[len(items)-1].Name)
	}
}
