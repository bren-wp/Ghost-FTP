package desktop

import "testing"

func sumWidths(widths []int) int {
	total := 0
	for _, width := range widths {
		total += width
	}
	return total
}

func TestCompactRemoteColumnsFitActualPaneAndKeepPermissionsReadable(t *testing.T) {
	for _, paneWidth := range []int{344, 378, 408, 520} {
		widths := fileColumnWidths(paneWidth, true)
		if len(widths) != 5 {
			t.Fatalf("remote widths=%v", widths)
		}
		if got, max := sumWidths(widths), paneWidth-18; got > max {
			t.Fatalf("remote columns overflow pane %d: widths=%v total=%d max=%d", paneWidth, widths, got, max)
		}
		if paneWidth < 520 && widths[4] < 84 {
			t.Fatalf("Permissions column too narrow at pane %d: widths=%v", paneWidth, widths)
		}
		for _, width := range widths {
			if width < 1 {
				t.Fatalf("invalid remote column width at pane %d: %v", paneWidth, widths)
			}
		}
	}
}

func TestCompactLocalColumnsFitActualPane(t *testing.T) {
	for _, paneWidth := range []int{344, 378, 408, 520} {
		widths := fileColumnWidths(paneWidth, false)
		if len(widths) != 4 {
			t.Fatalf("local widths=%v", widths)
		}
		if got, max := sumWidths(widths), paneWidth-18; got > max {
			t.Fatalf("local columns overflow pane %d: widths=%v total=%d max=%d", paneWidth, widths, got, max)
		}
		for _, width := range widths {
			if width < 1 {
				t.Fatalf("invalid local column width at pane %d: %v", paneWidth, widths)
			}
		}
	}
}

func TestWidePaneRetainsDetailedColumnWidths(t *testing.T) {
	remote := fileColumnWidths(720, true)
	if remote[1] != 82 || remote[2] != 92 || remote[3] != 132 || remote[4] != 112 {
		t.Fatalf("wide remote widths drifted: %v", remote)
	}
	local := fileColumnWidths(720, false)
	if local[1] != 82 || local[2] != 92 || local[3] != 132 {
		t.Fatalf("wide local widths drifted: %v", local)
	}
}
