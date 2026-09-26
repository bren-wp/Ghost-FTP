//go:build windows

package main

import (
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	procEnumWindows     = user32.NewProc("EnumWindows")
	procGetWindowTextW  = user32.NewProc("GetWindowTextW")
	procIsWindowVisible = user32.NewProc("IsWindowVisible")
	procShowWindow      = user32.NewProc("ShowWindow")
	procPostMessageW    = user32.NewProc("PostMessageW")
	procGetWindowLongW  = user32.NewProc("GetWindowLongW")
	procSetWindowLongW  = user32.NewProc("SetWindowLongW")
	procSetWindowPos    = user32.NewProc("SetWindowPos")
	procReleaseCapture  = user32.NewProc("ReleaseCapture")
	procSendMessageW    = user32.NewProc("SendMessageW")
)

const (
	gwlStyle        = ^uintptr(15) // -16
	wsCaption       = 0x00C00000
	wsThickFrame    = 0x00040000
	swMinimize      = 6
	swMaximize      = 3
	swRestore       = 9
	wmClose         = 0x0010
	wmNcLButtonDown = 0x00A1
	htCaption       = 2
	swpNoMove       = 0x0002
	swpNoSize       = 0x0001
	swpNoZOrder     = 0x0004
	swpFrameChanged = 0x0020
)

func findGhostWindow() uintptr {
	var found uintptr
	cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible == 0 {
			return 1
		}
		buf := make([]uint16, 256)
		n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n == 0 {
			return 1
		}
		title := syscall.UTF16ToString(buf)
		if title == "Ghost FTP" || strings.HasPrefix(title, "Ghost FTP -") {
			found = hwnd
			return 0
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	return found
}

func stripNativeCaptionSoon() {
	go func() {
		for i := 0; i < 40; i++ {
			time.Sleep(100 * time.Millisecond)
			hwnd := findGhostWindow()
			if hwnd == 0 {
				continue
			}
			style, _, _ := procGetWindowLongW.Call(hwnd, ^uintptr(15))
			style &^= wsCaption
			// Keep resize border but remove OS caption/titlebar.
			style |= wsThickFrame
			procSetWindowLongW.Call(hwnd, ^uintptr(15), style)
			procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpFrameChanged)
			return
		}
	}()
}

func handleWindowAction(action string) bool {
	hwnd := findGhostWindow()
	if hwnd == 0 {
		return false
	}
	switch action {
	case "minimize":
		procShowWindow.Call(hwnd, swMinimize)
	case "maximize":
		procShowWindow.Call(hwnd, swMaximize)
	case "restore":
		procShowWindow.Call(hwnd, swRestore)
	case "close":
		procPostMessageW.Call(hwnd, wmClose, 0, 0)
	case "drag":
		procReleaseCapture.Call()
		procSendMessageW.Call(hwnd, wmNcLButtonDown, htCaption, 0)
	default:
		return false
	}
	return true
}
