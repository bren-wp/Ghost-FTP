//go:build linux

package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestLinuxQueuePriorityRectsFollowClearQueueWithoutOverlap(t *testing.T) {
	u := &linuxDesktop{layout: linuxDesktopLayout{clearQueue: linuxRectWH(440, 700, 110, 28)}}
	up, down := u.queuePriorityRects()

	if up.top != u.layout.clearQueue.top || down.top != u.layout.clearQueue.top {
		t.Fatalf("priority controls must stay aligned with the queue toolbar: clear=%+v up=%+v down=%+v", u.layout.clearQueue, up, down)
	}
	if up.left <= u.layout.clearQueue.right || down.left <= up.right {
		t.Fatalf("priority controls overlap existing queue controls: clear=%+v up=%+v down=%+v", u.layout.clearQueue, up, down)
	}
	if up.bottom != u.layout.clearQueue.bottom || down.bottom != u.layout.clearQueue.bottom {
		t.Fatalf("priority controls must preserve the queue toolbar height: clear=%+v up=%+v down=%+v", u.layout.clearQueue, up, down)
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
	if !state.MoveUp || !state.MoveDown {
		t.Fatalf("selected queued job should move across nearest queued neighbors: %+v", state)
	}

	u.transferJobs[u.selectedTransfer].Status = "running"
	state = u.selectedQueuePriorityState()
	if state.MoveUp || state.MoveDown {
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
