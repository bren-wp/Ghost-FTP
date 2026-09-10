//go:build linux

package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestLinuxQueuePriorityRectsFollowClearQueueWithoutOverlap(t *testing.T) {
	u := &linuxDesktop{layout: linuxDesktopLayout{clearQueue: linuxRectWH(440, 700, 110, 28)}}
	top, up, down, bottom := u.queuePriorityRects()

	for name, rect := range map[string]linuxRect{"top": top, "up": up, "down": down, "bottom": bottom} {
		if rect.top != u.layout.clearQueue.top || rect.bottom != u.layout.clearQueue.bottom {
			t.Fatalf("%s priority control must stay aligned with queue toolbar: clear=%+v rect=%+v", name, u.layout.clearQueue, rect)
		}
	}
	if top.left <= u.layout.clearQueue.right || up.left <= top.right || down.left <= up.right || bottom.left <= down.right {
		t.Fatalf("priority controls overlap existing controls: clear=%+v top=%+v up=%+v down=%+v bottom=%+v", u.layout.clearQueue, top, up, down, bottom)
	}
}

func TestLinuxQueuePriorityStateUsesSharedQueuedOnlyPolicy(t *testing.T) {
	u := &linuxDesktop{
		selectedTransfer: 2,
		transferJobs: []model.TransferJob{
			{ID: "running", Status: "running"},
			{ID: "first", Status: "queued"},
			{ID: "selected", Status: "queued"},
			{ID: "done", Status: "done"},
			{ID: "last", Status: "queued"},
		},
	}

	state := u.selectedQueuePriorityState()
	if !state.MoveTop || !state.MoveUp || !state.MoveDown || !state.MoveBottom {
		t.Fatalf("selected queued job should expose all priority directions: %+v", state)
	}

	u.transferJobs[u.selectedTransfer].Status = "running"
	state = u.selectedQueuePriorityState()
	if state.MoveTop || state.MoveUp || state.MoveDown || state.MoveBottom {
		t.Fatalf("running transfer must not expose priority controls: %+v", state)
	}
}

func TestLinuxQueuePrioritySelectionRestoresByTransferID(t *testing.T) {
	u := &linuxDesktop{
		selectedTransfer: 0,
		transferJobs: []model.TransferJob{
			{ID: "a", Status: "queued"},
			{ID: "c", Status: "queued"},
			{ID: "b", Status: "queued"},
		},
	}

	u.restoreQueuePrioritySelection("b")
	if u.selectedTransfer != 2 {
		t.Fatalf("selection must follow transfer identity after reorder, got index %d", u.selectedTransfer)
	}

	u.restoreQueuePrioritySelection("missing")
	if u.selectedTransfer != -1 {
		t.Fatalf("missing transfer identity must clear stale selection, got index %d", u.selectedTransfer)
	}
}
