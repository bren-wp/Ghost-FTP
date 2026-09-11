//go:build linux

package desktop

import (
	"testing"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/itemlist"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestLinuxFileSortDefaultsMatchSharedNameOrdering(t *testing.T) {
	u := &linuxDesktop{language: "en", selectedLocal: -1, selectedRemote: -1}
	items := []model.Item{
		{Name: "zeta.txt", Size: 30},
		{Name: "folder", IsDirectory: true},
		{Name: "Alpha.txt", Size: 10},
	}
	u.acceptLinuxFileFilterSnapshot(false, items)
	if len(u.localItems) != 3 {
		t.Fatalf("local item count = %d, want 3", len(u.localItems))
	}
	if got := []string{u.localItems[0].Name, u.localItems[1].Name, u.localItems[2].Name}; got[0] != "folder" || got[1] != "Alpha.txt" || got[2] != "zeta.txt" {
		t.Fatalf("default Linux ordering = %v", got)
	}
}

func TestLinuxFileSortCyclePreservesSelection(t *testing.T) {
	u := &linuxDesktop{language: "en", selectedLocal: -1, selectedRemote: -1}
	items := []model.Item{
		{Name: "zeta.txt", Size: 30},
		{Name: "folder", IsDirectory: true},
		{Name: "alpha.txt", Size: 10},
	}
	u.acceptLinuxFileFilterSnapshot(false, items)
	u.selectedLocal = restoreLinuxSelection(u.localItems, "alpha.txt")
	u.cycleLinuxFileSort(false)

	if spec := u.linuxFileSortSpec(false); spec.Field != itemlist.FieldName || !spec.Descending {
		t.Fatalf("sort spec after first cycle = %+v, want name descending", spec)
	}
	if got := []string{u.localItems[0].Name, u.localItems[1].Name, u.localItems[2].Name}; got[0] != "folder" || got[1] != "zeta.txt" || got[2] != "alpha.txt" {
		t.Fatalf("descending Linux ordering = %v", got)
	}
	if u.selectedLocal != 2 || u.localItems[u.selectedLocal].Name != "alpha.txt" {
		t.Fatalf("selection was not restored after sorting: index=%d items=%v", u.selectedLocal, u.localItems)
	}
}

func TestLinuxRemoteSortIncludesPermissionsAndUnknownMetadataLast(t *testing.T) {
	u := &linuxDesktop{language: "en", connected: true, selectedLocal: -1, selectedRemote: -1}
	state := linuxFileSortStateFor(u)
	state.remoteIndex = 8
	items := []model.Item{
		{Name: "unknown.txt", Permissions: ""},
		{Name: "private.txt", Permissions: "0600"},
		{Name: "public.txt", Permissions: "0644"},
		{Name: "folder", IsDirectory: true, Permissions: "0755"},
	}
	u.acceptLinuxFileFilterSnapshot(true, items)
	if spec := u.linuxFileSortSpec(true); spec.Field != itemlist.FieldPermissions || spec.Descending {
		t.Fatalf("remote permission spec = %+v", spec)
	}
	if got := []string{u.remoteItems[0].Name, u.remoteItems[1].Name, u.remoteItems[2].Name, u.remoteItems[3].Name}; got[0] != "folder" || got[1] != "private.txt" || got[2] != "public.txt" || got[3] != "unknown.txt" {
		t.Fatalf("permission ordering = %v", got)
	}
}

func TestLinuxSortSupportsModifiedAndFilterComposition(t *testing.T) {
	u := &linuxDesktop{language: "en", selectedLocal: -1, selectedRemote: -1}
	state := linuxFileSortStateFor(u)
	state.localIndex = 6
	old := time.Unix(100, 0)
	newer := time.Unix(200, 0)
	u.acceptLinuxFileFilterSnapshot(false, []model.Item{
		{Name: "new.log", Modified: newer},
		{Name: "other.txt", Modified: time.Unix(50, 0)},
		{Name: "old.log", Modified: old},
	})
	u.applyLinuxFileFilter(false, ".log")
	if len(u.localItems) != 2 || u.localItems[0].Name != "old.log" || u.localItems[1].Name != "new.log" {
		t.Fatalf("filtered modified ordering = %#v", u.localItems)
	}
}

func TestLinuxSortControlDoesNotOverlapRecursiveSearchControl(t *testing.T) {
	u := &linuxDesktop{layout: buildLinuxDesktopLayout(premiumStartWidth, premiumStartHeight)}
	for _, remote := range []bool{false, true} {
		sortRect := u.fileSortControlRect(remote)
		searchRect := u.recursiveSearchButtonRect(remote)
		if sortRect.right > searchRect.left {
			t.Fatalf("sort rect overlaps search rect: sort=%+v search=%+v", sortRect, searchRect)
		}
		filterRect := u.fileFilterPromptControlRect(remote)
		if filterRect.right > sortRect.left {
			t.Fatalf("filter rect overlaps sort rect: filter=%+v sort=%+v", filterRect, sortRect)
		}
	}
}
