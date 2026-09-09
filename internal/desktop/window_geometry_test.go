package desktop

import "testing"

func TestResponsiveWindowBoundsUsesCanonicalStartSizeOnLargeWorkArea(t *testing.T) {
	got := responsiveWindowBoundsForWorkArea(0, 0, 1920, 1080)
	if got.Width != premiumStartWidth || got.Height != premiumStartHeight {
		t.Fatalf("unexpected preferred size: %+v", got)
	}
	if got.X != (1920-premiumStartWidth)/2 || got.Y != (1080-premiumStartHeight)/2 {
		t.Fatalf("window was not centered in work area: %+v", got)
	}
}

func TestResponsiveWindowBoundsNeverExceedsConstrainedWorkArea(t *testing.T) {
	for _, tc := range []struct {
		name          string
		x, y, w, h    int
	}{
		{name: "small laptop", w: 800, h: 600},
		{name: "short desktop", w: 1024, h: 600},
		{name: "offset monitor", x: -1280, y: 40, w: 900, h: 650},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := responsiveWindowBoundsForWorkArea(tc.x, tc.y, tc.w, tc.h)
			if got.Width > tc.w || got.Height > tc.h {
				t.Fatalf("window exceeds work area: work=%dx%d got=%+v", tc.w, tc.h, got)
			}
			if got.X < tc.x || got.Y < tc.y || got.X+got.Width > tc.x+tc.w || got.Y+got.Height > tc.y+tc.h {
				t.Fatalf("window is outside work area: work=(%d,%d %dx%d) got=%+v", tc.x, tc.y, tc.w, tc.h, got)
			}
		})
	}
}

func TestResponsiveWindowBoundsPreservesMinimumWhenWorkAreaCanFitIt(t *testing.T) {
	got := responsiveWindowBoundsForWorkArea(0, 0, premiumMinWidth+20, premiumMinHeight+20)
	if got.Width < premiumMinWidth || got.Height < premiumMinHeight {
		t.Fatalf("canonical minimum was discarded despite available space: %+v", got)
	}
}

func TestResponsiveMinimumTrackSizeClampsToWorkArea(t *testing.T) {
	w, h := responsiveMinimumTrackSize(800, 600)
	if w != 800 || h != 600 {
		t.Fatalf("constrained minimum mismatch: got %dx%d", w, h)
	}
	w, h = responsiveMinimumTrackSize(1920, 1080)
	if w != premiumMinWidth || h != premiumMinHeight {
		t.Fatalf("normal minimum mismatch: got %dx%d", w, h)
	}
}
