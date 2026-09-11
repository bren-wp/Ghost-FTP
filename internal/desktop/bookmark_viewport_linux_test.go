//go:build linux

package desktop

import (
	"fmt"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func linuxBookmarkTestItems(count int) []model.Bookmark {
	items := make([]model.Bookmark, count)
	for i := range items {
		items[i] = model.Bookmark{ID: fmt.Sprintf("bookmark-%d", i), Name: fmt.Sprintf("Bookmark %d", i)}
	}
	return items
}

func TestLinuxBookmarkViewportKeepsKeyboardSelectionVisible(t *testing.T) {
	state := &linuxBookmarkUIState{
		items:        linuxBookmarkTestItems(20),
		selected:     12,
		firstVisible: 0,
		visibleRows:  5,
	}
	state.ensureSelectionVisible()
	if state.firstVisible != 8 {
		t.Fatalf("firstVisible = %d, want 8 for selected row 12 in a five-row viewport", state.firstVisible)
	}

	state.selected = 2
	state.ensureSelectionVisible()
	if state.firstVisible != 2 {
		t.Fatalf("viewport did not follow upward keyboard selection: got %d, want 2", state.firstVisible)
	}
}

func TestLinuxBookmarkViewportMouseScrollReachesLaterRows(t *testing.T) {
	state := &linuxBookmarkUIState{
		items:        linuxBookmarkTestItems(20),
		selected:     0,
		firstVisible: 0,
		visibleRows:  5,
	}
	state.scrollViewport(3)
	if state.firstVisible != 3 || state.selected != 3 {
		t.Fatalf("scroll down did not advance selectable viewport: first=%d selected=%d", state.firstVisible, state.selected)
	}

	state.scrollViewport(100)
	if state.firstVisible != 15 || state.selected != 15 {
		t.Fatalf("viewport did not clamp to final page: first=%d selected=%d", state.firstVisible, state.selected)
	}

	state.selected = 19
	state.ensureSelectionVisible()
	if state.firstVisible != 15 {
		t.Fatalf("last bookmark is not visible on final page: first=%d", state.firstVisible)
	}
}

func TestLinuxBookmarkViewportClampsAfterCollectionShrinks(t *testing.T) {
	state := &linuxBookmarkUIState{
		items:        linuxBookmarkTestItems(4),
		selected:     9,
		firstVisible: 8,
		visibleRows:  5,
	}
	state.ensureSelectionVisible()
	if state.selected != 3 || state.firstVisible != 0 {
		t.Fatalf("shrunk collection was not clamped: first=%d selected=%d", state.firstVisible, state.selected)
	}
}

func TestLinuxBookmarkRowHitTestRejectsPaddingAndUsesViewportOffset(t *testing.T) {
	u := &linuxDesktop{}
	state := &linuxBookmarkUIState{
		items:        linuxBookmarkTestItems(20),
		selected:     7,
		firstVisible: 7,
		visibleRows:  5,
		rows:         linuxRectWH(100, 200, 400, 168),
	}
	linuxBookmarkUIStates.Store(u, state)
	t.Cleanup(func() { linuxBookmarkUIStates.Delete(u) })

	if got := u.bookmarkRowAt(110, 202); got != -1 {
		t.Fatalf("top padding selected bookmark %d, want no row", got)
	}
	if got := u.bookmarkRowAt(110, 204); got != 7 {
		t.Fatalf("first visible row mapped to %d, want bookmark 7", got)
	}
	if got := u.bookmarkRowAt(110, 236); got != 8 {
		t.Fatalf("second visible row mapped to %d, want bookmark 8", got)
	}
}
