//go:build windows

package desktop

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func (a *app) goSafe(fn func()) {
	go func() {
		defer func() {
			if recover() != nil {
				a.dispatch(func() {
					a.setStatus(a.tr("error.generic"))
				})
			}
		}()
		fn()
	}()
}

func (a *app) dispatch(f func()) {
	a.mu.Lock()
	if a.closing {
		a.mu.Unlock()
		return
	}
	a.dispatchQ = append(a.dispatchQ, f)
	a.mu.Unlock()
	postMessageW.Call(a.hwnd, wmAppDispatch, 0, 0)
}

func (a *app) runDispatch() {
	a.mu.Lock()
	q := append([]func(){}, a.dispatchQ...)
	a.dispatchQ = nil
	a.mu.Unlock()
	for _, f := range q {
		func() {
			defer func() {
				if recover() != nil {
					a.setStatus(a.tr("error.generic"))
				}
			}()
			f()
		}()
	}
}

func (a *app) setStatus(text string) {
	if a == nil || a.status == 0 {
		return
	}
	text = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", " "), "\n", " "))
	if text == "" {
		return
	}
	const maxStatusChars = 800
	if len(text) > maxStatusChars {
		text = text[:maxStatusChars] + "…"
	}
	setText(a.status, text)
}

func setText(hwnd uintptr, text string) {
	if hwnd != 0 {
		setWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(wstr(text))))
	}
}

func getText(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	n, _, _ := getWindowTextLengthW.Call(hwnd)
	// Defensive cap: UI controls have stricter per-field limits, but never trust
	// a window message enough to allocate an unbounded buffer here.
	if n > 65535 {
		return ""
	}
	buf := make([]uint16, int(n)+2)
	if len(buf) == 0 {
		return ""
	}
	getWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func selectedIndex(list uintptr) int {
	r, _, _ := sendMessageW.Call(list, lvmGetNextItem, ^uintptr(0), lvniSelected)
	return int(r)
}

func selectedIndices(list uintptr) []int {
	if list == 0 {
		return nil
	}
	out := make([]int, 0, 8)
	idx := -1
	for {
		var start uintptr
		if idx < 0 {
			start = ^uintptr(0)
		} else {
			start = uintptr(idx)
		}
		r, _, _ := sendMessageW.Call(list, lvmGetNextItem, start, lvniSelected)
		next := int(r)
		if next < 0 || next == idx {
			break
		}
		out = append(out, next)
		idx = next
		if len(out) >= 5000 {
			break
		}
	}
	return out
}

func selectedItemNames(list uintptr, items []model.Item) map[string]struct{} {
	selected := make(map[string]struct{})
	for _, index := range selectedIndices(list) {
		if index >= 0 && index < len(items) {
			selected[items[index].Name] = struct{}{}
		}
	}
	return selected
}

func setListRowSelected(list uintptr, row int, selected bool) {
	if list == 0 || row < 0 {
		return
	}
	state := uint32(0)
	if selected {
		state = lvisSelected
	}
	item := lvItem{State: state, StateMask: lvisSelected}
	sendMessageW.Call(list, lvmSetItemState, uintptr(row), uintptr(unsafe.Pointer(&item)))
}

func ownerForItemList(list uintptr) *app {
	var owner *app
	apps.Range(func(_, value any) bool {
		candidate, ok := value.(*app)
		if ok && (candidate.localList == list || candidate.remoteList == list) {
			owner = candidate
			return false
		}
		return true
	})
	return owner
}

func effectiveItemsForList(list uintptr, fallback []model.Item) []model.Item {
	owner := ownerForItemList(list)
	if owner == nil {
		return fallback
	}
	if list == owner.remoteList {
		return owner.remoteItems
	}
	if list == owner.localList {
		return owner.localItems
	}
	return fallback
}

func restoreItemSelection(list uintptr, items []model.Item, selected map[string]struct{}) {
	if len(selected) == 0 {
		return
	}
	items = effectiveItemsForList(list, items)
	for i, item := range items {
		if _, ok := selected[item.Name]; ok {
			setListRowSelected(list, i, true)
		}
	}
}

func clearList(list uintptr) { sendMessageW.Call(list, lvmDeleteAllItems, 0, 0) }

func setListRedraw(list uintptr, enabled bool) {
	if list == 0 {
		return
	}
	value := uintptr(0)
	if enabled {
		value = 1
	}
	sendMessageW.Call(list, wmSetRedraw, value, 0)
	if enabled {
		invalidateRect.Call(list, 0, 1)
	}
}

// fillItems is the narrow compatibility bridge used by asynchronous navigation
// callbacks. Rendering is centralized in app.fillItemList so locale handling
// and row construction have one authoritative implementation.
func fillItems(list uintptr, items []model.Item) {
	owner := ownerForItemList(list)
	if owner == nil {
		return
	}
	owner.fillItemList(list, items)
}

func insertListRow(list uintptr, row int, cols []string) {
	insertListRowWithImage(list, row, cols, -1)
}

func insertListRowWithImage(list uintptr, row int, cols []string, image int32) {
	if len(cols) == 0 {
		return
	}
	first := syscall.StringToUTF16(cols[0])
	mask := uint32(lvifText)
	if image >= 0 {
		mask |= lvifImage
	}
	item := lvItem{Mask: mask, Item: int32(row), Text: &first[0], Image: image}
	sendMessageW.Call(list, lvmInsertItemW, 0, uintptr(unsafe.Pointer(&item)))
	for col := 1; col < len(cols); col++ {
		text := syscall.StringToUTF16(cols[col])
		sub := lvItem{SubItem: int32(col), Text: &text[0]}
		sendMessageW.Call(list, lvmSetItemTextW, uintptr(row), uintptr(unsafe.Pointer(&sub)))
	}
}

func attachSystemImageList(list uintptr) {
	if list == 0 {
		return
	}
	var info shFileInfo
	probe := wstr("GhostFTP.file")
	h, _, _ := shGetFileInfoW.Call(
		uintptr(unsafe.Pointer(probe)), fileAttributeNormal, uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info),
		shgfiUseFileAttributes|shgfiSysIconIndex|shgfiSmallIcon,
	)
	if h != 0 {
		sendMessageW.Call(list, lvmSetImageList, lvsilSmall, h)
	}
}

var systemIconCache sync.Map

func systemIconIndex(name string, directory bool) int32 {
	attr := uintptr(fileAttributeNormal)
	key := strings.ToLower(filepath.Ext(name))
	probe := "GhostFTP" + key
	if key == "" {
		key = "<file>"
		probe = "GhostFTP.file"
	}
	if directory {
		attr = fileAttributeDirectory
		key = "<dir>"
		probe = "GhostFTP-folder"
	}
	if cached, ok := systemIconCache.Load(key); ok {
		return cached.(int32)
	}
	var info shFileInfo
	p := wstr(probe)
	r, _, _ := shGetFileInfoW.Call(
		uintptr(unsafe.Pointer(p)), attr, uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info),
		shgfiUseFileAttributes|shgfiSysIconIndex|shgfiSmallIcon,
	)
	if r == 0 {
		return -1
	}
	systemIconCache.Store(key, info.IconIndex)
	return info.IconIndex
}

func formatSize(n int64, dir bool) string {
	if dir {
		return "—"
	}
	const kb = 1024
	if n < kb {
		return fmt.Sprintf("%d B", n)
	}
	if n < kb*kb {
		return fmt.Sprintf("%.1f KB", float64(n)/kb)
	}
	if n < kb*kb*kb {
		return fmt.Sprintf("%.1f MB", float64(n)/(kb*kb))
	}
	return fmt.Sprintf("%.2f GB", float64(n)/(kb*kb*kb))
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("02.01.2006 15:04")
}
