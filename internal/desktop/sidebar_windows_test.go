//go:build windows

package desktop

import "testing"

func TestCleanupSidebarControlsRemovesPerWindowState(t *testing.T) {
	a := &app{hwnd: uintptr(0x1234), buttons: make(map[uintptr]buttonVisual)}
	stores := []*struct {
		name  string
		store interface {
			Store(any, any)
			Load(any) (any, bool)
		}
		hwnd uintptr
	}{
		{name: "files", store: &sidebarFiles, hwnd: 0x2001},
		{name: "transfers", store: &sidebarTransfers, hwnd: 0x2002},
		{name: "bookmarks", store: &sidebarBookmarks, hwnd: 0x2003},
		{name: "diagnostics", store: &sidebarDiagnostics, hwnd: 0x2004},
	}

	for _, item := range stores {
		item.store.Store(a.hwnd, item.hwnd)
		a.buttons[item.hwnd] = buttonVisual{Label: item.name}
	}

	a.cleanupSidebarControls()

	for _, item := range stores {
		if _, ok := item.store.Load(a.hwnd); ok {
			t.Fatalf("%s sidebar handle survived cleanup", item.name)
		}
		if _, ok := a.buttons[item.hwnd]; ok {
			t.Fatalf("%s button metadata survived cleanup", item.name)
		}
	}
}
