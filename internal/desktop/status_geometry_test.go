package desktop

import "testing"

func TestStatusBandGeometryKeepsFooterInsideMinimumWorkspace(t *testing.T) {
	statusY, contentBottom := statusBandGeometry(premiumMinHeight)

	if statusY != 646 {
		t.Fatalf("minimum-height status y = %d, want 646", statusY)
	}
	if contentBottom != 639 {
		t.Fatalf("minimum-height content bottom = %d, want 639", contentBottom)
	}
	if got := premiumMinHeight - (statusY + statusBandHeight); got != statusBandBottomInset {
		t.Fatalf("status bottom inset = %d, want %d", got, statusBandBottomInset)
	}
	if got := statusY - contentBottom; got != statusBandContentGap {
		t.Fatalf("content/status gap = %d, want %d", got, statusBandContentGap)
	}
	if statusY <= 0 || contentBottom <= 0 {
		t.Fatalf("unexpected non-positive geometry: statusY=%d contentBottom=%d", statusY, contentBottom)
	}
}

func TestStatusBandGeometryClampsTinyHeight(t *testing.T) {
	statusY, contentBottom := statusBandGeometry(1)
	if statusY != 0 || contentBottom != 0 {
		t.Fatalf("tiny-height geometry = (%d, %d), want (0, 0)", statusY, contentBottom)
	}
}
