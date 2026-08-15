//go:build windows

package desktop

import (
	"brendigo.com/byftp/internal/model"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

func (a *app) goSafe(fn func()) {
	go func() {
		defer func() {
			if recover() != nil {
				a.dispatch(func() {
					a.setStatus("Radnja nije dovršena. Pokušajte ponovno.")
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
					a.setStatus("Radnja nije dovršena. Pokušajte ponovno.")
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

func clearList(list uintptr) { sendMessageW.Call(list, lvmDeleteAllItems, 0, 0) }

func fillItems(list uintptr, items []model.Item) {
	clearList(list)
	for i, it := range items {
		insertListRowWithImage(list, i, []string{it.Name, formatItemType(it), formatSize(it.Size, it.IsDirectory), formatTime(it.Modified)}, systemIconIndex(it.Name, it.IsDirectory))
	}
}

func fillTransfers(list uintptr, jobs []model.TransferJob) {
	clearList(list)
	for i, j := range jobs {
		dir := "Preuzimanje"
		if j.Direction == "upload" {
			dir = "Slanje"
		}
		status := transferStatusLabel(j.Status)
		if j.Status == "running" && j.Attempts > 1 {
			status += fmt.Sprintf(" — pokušaj %d", j.Attempts)
		}
		if j.Error != "" {
			status += ": " + j.Error
		}
		progress := j.Progress
		if progress > 0 && progress <= 1 {
			progress *= 100
		}
		if progress < 0 {
			progress = 0
		}
		if progress > 100 {
			progress = 100
		}
		insertListRow(list, i, []string{dir, j.LocalPath, j.RemotePath, status, fmt.Sprintf("%.0f%%", progress)})
	}
}

func transferStatusLabel(status string) string {
	switch status {
	case "queued":
		return "Na čekanju"
	case "running":
		return "U tijeku"
	case "done":
		return "Završeno"
	case "failed":
		return "Greška"
	case "cancelled":
		return "Otkazano"
	case "skipped":
		return "Preskočeno"
	default:
		return status
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

func formatItemType(it model.Item) string {
	if it.IsSymlink {
		return "Veza"
	}
	if it.IsDirectory {
		return "Mapa"
	}
	ext := strings.TrimPrefix(strings.ToUpper(filepath.Ext(it.Name)), ".")
	if ext == "" {
		return "Datoteka"
	}
	if len(ext) > 8 {
		ext = ext[:8]
	}
	return ext
}

func formatSize(n int64, dir bool) string {
	if dir {
		return "Mapa"
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
