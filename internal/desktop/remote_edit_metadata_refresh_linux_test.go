//go:build linux

package desktop

import (
	"errors"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestRemoteEditMetadataRefreshPreservesSelectionAndStatus(t *testing.T) {
	u := &linuxDesktop{
		remoteCurrent:  "/site",
		remoteItems:    []model.Item{{Name: "index.php", Size: 10}, {Name: "other.txt", Size: 5}},
		selectedRemote: 0,
		status:         "Saved and verified.",
	}

	result := linuxUIResult{
		action:    linuxActionRemoteEditMetadataRefresh,
		localBase: "/site",
		remoteItems: []model.Item{
			{Name: "other.txt", Size: 5},
			{Name: "index.php", Size: 42},
		},
	}
	if !u.handleRemoteEditResult(result) {
		t.Fatal("metadata refresh result was not consumed")
	}
	if u.status != "Saved and verified." {
		t.Fatalf("metadata refresh changed successful save status: %q", u.status)
	}
	if u.selectedRemote != 1 {
		t.Fatalf("edited file selection was not restored: got %d", u.selectedRemote)
	}
	if got := u.remoteItems[u.selectedRemote].Size; got != 42 {
		t.Fatalf("remote metadata was not refreshed: got size %d", got)
	}
}

func TestRemoteEditMetadataRefreshFailureDoesNotRewriteSaveOutcome(t *testing.T) {
	u := &linuxDesktop{
		remoteCurrent:  "/site",
		remoteItems:    []model.Item{{Name: "index.php", Size: 10}},
		selectedRemote: 0,
		status:         "Saved and verified.",
	}

	result := linuxUIResult{
		action: linuxActionRemoteEditMetadataRefresh,
		err:    errors.New("listing unavailable"),
	}
	if !u.handleRemoteEditResult(result) {
		t.Fatal("failed metadata refresh result was not consumed")
	}
	if u.status != "Saved and verified." {
		t.Fatalf("listing failure replaced successful save status: %q", u.status)
	}
	if len(u.remoteItems) != 1 || u.remoteItems[0].Size != 10 {
		t.Fatalf("failed refresh replaced the existing remote list: %#v", u.remoteItems)
	}
}
