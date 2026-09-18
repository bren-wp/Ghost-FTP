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

func TestLinuxMasterRailUtilitiesStayInsideWindowAndAboveStatusBand(t *testing.T) {
	layout := buildLinuxMasterRailLayout(premiumMinWidth, premiumMinHeight)
	for name, r := range map[string]linuxRect{
		"bookmarks":   layout.bookmarks,
		"diagnostics": layout.diagnostics,
		"about":       layout.about,
	} {
		if r.left < 0 || r.top < 0 || r.right > premiumMinWidth || r.bottom > premiumMinHeight {
			t.Fatalf("%s outside minimum window: %+v", name, r)
		}
	}
	if layout.about.bottom > premiumMinHeight-linuxMasterRailBottomInset {
		t.Fatalf("utility controls overlap reserved status band: %+v", layout.about)
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
	if got := linuxTransferBadgeLabel(100); got != "99+" {
		t.Fatalf("badge label = %q, want 99+", got)
	}
}

func TestLinuxMasterAffineTransformMovesWorkspaceOutOfRail(t *testing.T) {
	original := linuxRectWH(premiumOuterGap, 100, 300, 30)
	transformed := linuxAffineRect(original, premiumOuterGap, 1280-premiumOuterGap, linuxMasterContentLeft, 1280-premiumOuterGap)
	if transformed.left != linuxMasterContentLeft {
		t.Fatalf("transformed left = %d, want %d", transformed.left, linuxMasterContentLeft)
	}
	if transformed.right <= transformed.left {
		t.Fatalf("invalid transformed rect: %+v", transformed)
	}
	if transformed.top != original.top || transformed.bottom != original.bottom {
		t.Fatalf("horizontal transform changed vertical geometry: before=%+v after=%+v", original, transformed)
	}
}

func TestLinuxMasterLayoutTransformUsesRailSettingsHitTarget(t *testing.T) {
	u := &linuxDesktop{width: 1280, height: 820, layout: buildLinuxDesktopLayout(1280, 820)}
	u.applyLinuxMasterLayoutTransform()
	rail := buildLinuxMasterRailLayout(1280, 820)
	if u.layout.settings != rail.settings {
		t.Fatalf("settings hit target = %+v, want %+v", u.layout.settings, rail.settings)
	}
	if u.layout.localPath.left < linuxMasterContentLeft {
		t.Fatalf("local workspace overlaps rail after transform: %+v", u.layout.localPath)
	}
	if u.layout.queue.left < linuxMasterContentLeft {
		t.Fatalf("queue overlaps rail after transform: %+v", u.layout.queue)
	}
}

func TestLinuxMasterLayoutTransformIsIdempotentAcrossRenderHooks(t *testing.T) {
	u := &linuxDesktop{width: 1280, height: 820, layout: buildLinuxDesktopLayout(1280, 820)}
	u.applyLinuxMasterLayoutTransform()
	firstLocalPath := u.layout.localPath
	firstRemotePath := u.layout.remotePath
	firstQueue := u.layout.queue
	firstSettings := u.layout.settings

	u.applyLinuxMasterLayoutTransform()
	if u.layout.localPath != firstLocalPath || u.layout.remotePath != firstRemotePath || u.layout.queue != firstQueue {
		t.Fatalf("second transform compounded workspace geometry: local=%+v remote=%+v queue=%+v", u.layout.localPath, u.layout.remotePath, u.layout.queue)
	}
	if u.layout.settings != firstSettings {
		t.Fatalf("second transform moved settings rail target: before=%+v after=%+v", firstSettings, u.layout.settings)
	}
}

func TestLinuxMasterToolbarMatchesReferenceHierarchy(t *testing.T) {
	layout := buildLinuxMasterToolbarLayout(1280)
	if layout.region.left != linuxMasterContentLeft || layout.region.top != 72 {
		t.Fatalf("toolbar region = %+v", layout.region)
	}
	row := []linuxRect{
		layout.back, layout.forward, layout.refresh, layout.newFolder,
		layout.upload, layout.download, layout.bookmarks, layout.more,
	}
	for index := 1; index < len(row); index++ {
		if row[index].left <= row[index-1].right {
			t.Fatalf("toolbar controls overlap at %d: previous=%+v current=%+v", index, row[index-1], row[index])
		}
	}
	if layout.connection.right >= layout.status.left || layout.status.right >= layout.quick.left {
		t.Fatalf("connection row overlaps: connection=%+v status=%+v quick=%+v", layout.connection, layout.status, layout.quick)
	}
}

func TestLinuxWorkspaceHistoryIsBoundedAndDeduplicated(t *testing.T) {
	var history []linuxWorkspaceHistoryEntry
	for i := 0; i < linuxWorkspaceHistoryLimit+8; i++ {
		history = appendLinuxWorkspaceHistory(history, linuxWorkspaceHistoryEntry{Path: fmt.Sprintf("/tmp/%d", i)})
	}
	if len(history) != linuxWorkspaceHistoryLimit {
		t.Fatalf("history length = %d, want %d", len(history), linuxWorkspaceHistoryLimit)
	}
	last := history[len(history)-1]
	history = appendLinuxWorkspaceHistory(history, last)
	if len(history) != linuxWorkspaceHistoryLimit {
		t.Fatalf("duplicate history entry changed length: %d", len(history))
	}
	remote := normalizeLinuxWorkspaceHistoryEntry(linuxWorkspaceHistoryEntry{Remote: true, Path: "/var/www/../www/site"})
	if remote.Path != "/var/www/site" {
		t.Fatalf("remote history path = %q, want /var/www/site", remote.Path)
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
