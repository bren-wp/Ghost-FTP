//go:build linux

package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestLinuxMasterRailReservesStableContentColumn(t *testing.T) {
	layout := buildLinuxMasterRailLayout(1280, 820)
	if layout.content.left != linuxMasterContentLeft {
		t.Fatalf("content left = %d, want %d", layout.content.left, linuxMasterContentLeft)
	}
	if layout.files.left != linuxMasterRailX || layout.files.right-layout.files.left != linuxMasterRailWidth {
		t.Fatalf("files rail geometry = %+v", layout.files)
	}
	if layout.content.left <= layout.files.right {
		t.Fatalf("content overlaps rail: content=%+v files=%+v", layout.content, layout.files)
	}
}

func TestLinuxMasterRailPrimaryActionsAreOrderedAndSeparated(t *testing.T) {
	layout := buildLinuxMasterRailLayout(1280, 820)
	primary := []linuxRect{layout.files, layout.connections, layout.transfers, layout.settings}
	for index := 1; index < len(primary); index++ {
		previous := primary[index-1]
		current := primary[index]
		if current.top-previous.bottom != linuxMasterRailCardGap {
			t.Fatalf("primary gap %d = %d, want %d", index, current.top-previous.bottom, linuxMasterRailCardGap)
		}
	}
}

func TestLinuxMasterRailUtilitiesStayInsideWindow(t *testing.T) {
	layout := buildLinuxMasterRailLayout(premiumMinWidth, premiumMinHeight)
	for name, r := range map[string]linuxRect{
		"bookmarks": layout.bookmarks,
		"diagnostics": layout.diagnostics,
		"about": layout.about,
		"language": layout.language,
	} {
		if r.left < 0 || r.top < 0 || r.right > premiumMinWidth || r.bottom > premiumMinHeight {
			t.Fatalf("%s outside minimum window: %+v", name, r)
		}
	}
}

func TestLinuxMasterTransferBadgeUsesActionableQueueState(t *testing.T) {
	jobs := []model.TransferJob{
		{Status: "queued"},
		{Status: "running"},
		{Status: "failed"},
		{Status: "cancelled"},
		{Status: "done"},
		{Status: "skipped"},
	}
	if got := linuxMasterTransferBadge(jobs); got != 4 {
		t.Fatalf("badge count = %d, want 4", got)
	}
}

func TestLinuxMasterRailUsesSharedLocalizedNavigationContract(t *testing.T) {
	english := navigationLabelsForLanguage("en")
	croatian := navigationLabelsForLanguage("hr")
	if english.Files != "Files" || english.Connections != "Connections" || english.TransferQueue != "Transfer Queue" {
		t.Fatalf("unexpected English navigation labels: %+v", english)
	}
	if croatian.Files != "Datoteke" || croatian.Connections != "Veze" || croatian.TransferQueue != "Red prijenosa" {
		t.Fatalf("unexpected Croatian navigation labels: %+v", croatian)
	}
}
