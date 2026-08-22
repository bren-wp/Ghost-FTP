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

	"brendigo.com/byftp/internal/model"
)

func (a *app) goSafe(fn func()) {
	go func() {
		defer func() {
			if recover() != nil {
				a.dispatch(func() {
					a.setStatus(a.tr("action.failed_retry"))
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
	queue := append([]func(){}, a.dispatchQ...)
	a.dispatchQ = nil
	a.mu.Unlock()
	for _, f := range queue {
		func() {
			defer func() {
				if recover() != nil {
					a.setStatus(a.tr("action.failed_retry"))
				}
			}()
			f()
		}()
	}
}

func (a *app) setStatus(text string) { setText(a.status, text) }

func setText(hwnd uintptr, text string) {
	if hwnd != 0 {
		setWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(wstr(text))))
	}
}

func getText(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	length, _, _ := getWindowTextLengthW.Call(hwnd)
	// Defensive cap: UI controls have stricter per-field limits, but never trust
	// a window message enough to allocate an unbounded buffer here.
	if length > 65535 {
		return ""
	}
	buf := make([]uint16, int(length)+2)
	if len(buf) == 0 {
		return ""
	}
	getWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func selectedIndex(list uintptr) int {
	result, _, _ := sendMessageW.Call(list, lvmGetNextItem, ^uintptr(0), lvniSelected)
	return int(result)
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
		result, _, _ := sendMessageW.Call(list, lvmGetNextItem, start, lvniSelected)
		next := int(result)
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

func restoreItemSelection(list uintptr, items []model.Item, selected map[string]struct{}) {
	if len(selected) == 0 {
		return
	}
	for index, item := range items {
		if _, ok := selected[item.Name]; ok {
			setListRowSelected(list, index, true)
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
	probe := wstr("byftp.file")
	handle, _, _ := shGetFileInfoW.Call(
		uintptr(unsafe.Pointer(probe)), fileAttributeNormal, uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info),
		shgfiUseFileAttributes|shgfiSysIconIndex|shgfiSmallIcon,
	)
	if handle != 0 {
		sendMessageW.Call(list, lvmSetImageList, lvsilSmall, handle)
	}
}

var systemIconCache sync.Map

func systemIconIndex(name string, directory bool) int32 {
	attr := uintptr(fileAttributeNormal)
	key := strings.ToLower(filepath.Ext(name))
	probe := "byftp" + key
	if key == "" {
		key = "<file>"
		probe = "byftp.file"
	}
	if directory {
		attr = fileAttributeDirectory
		key = "<dir>"
		probe = "ByFTP-folder"
	}
	if cached, ok := systemIconCache.Load(key); ok {
		return cached.(int32)
	}
	var info shFileInfo
	pointer := wstr(probe)
	result, _, _ := shGetFileInfoW.Call(
		uintptr(unsafe.Pointer(pointer)), attr, uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info),
		shgfiUseFileAttributes|shgfiSysIconIndex|shgfiSmallIcon,
	)
	if result == 0 {
		return -1
	}
	systemIconCache.Store(key, info.IconIndex)
	return info.IconIndex
}

func formatSize(size int64, directory bool) string {
	if directory {
		return ""
	}
	const kb = 1024
	if size < kb {
		return fmt.Sprintf("%d B", size)
	}
	if size < kb*kb {
		return fmt.Sprintf("%.1f KB", float64(size)/kb)
	}
	if size < kb*kb*kb {
		return fmt.Sprintf("%.1f MB", float64(size)/(kb*kb))
	}
	return fmt.Sprintf("%.2f GB", float64(size)/(kb*kb*kb))
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Local().Format("2006-01-02 15:04")
}
