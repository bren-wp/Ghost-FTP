//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var u32 = syscall.NewLazyDLL("user32.dll")
var enumW = u32.NewProc("EnumWindows")
var getText = u32.NewProc("GetWindowTextW")
var isVis = u32.NewProc("IsWindowVisible")
var showW = u32.NewProc("ShowWindow")
var postW = u32.NewProc("PostMessageW")
var getLong = u32.NewProc("GetWindowLongW")
var setLong = u32.NewProc("SetWindowLongW")
var setPos = u32.NewProc("SetWindowPos")
var releaseCapture = u32.NewProc("ReleaseCapture")
var sendMessage = u32.NewProc("SendMessageW")

func findSetupWindow() uintptr {
	var found uintptr
	cb := syscall.NewCallback(func(hwnd, l uintptr) uintptr {
		v, _, _ := isVis.Call(hwnd)
		if v == 0 {
			return 1
		}
		buf := make([]uint16, 256)
		n, _, _ := getText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if n == 0 {
			return 1
		}
		title := syscall.UTF16ToString(buf)
		if title == "Ghost FTP Setup" || strings.HasPrefix(title, "Ghost FTP Setup -") {
			found = hwnd
			return 0
		}
		return 1
	})
	enumW.Call(cb, 0)
	return found
}
func stripNativeCaptionSoon() {
	go func() {
		for i := 0; i < 40; i++ {
			time.Sleep(100 * time.Millisecond)
			h := findSetupWindow()
			if h == 0 {
				continue
			}
			style, _, _ := getLong.Call(h, ^uintptr(15))
			style &^= 0x00C00000
			style |= 0x00040000
			setLong.Call(h, ^uintptr(15), style)
			setPos.Call(h, 0, 0, 0, 0, 0, 0x0002|0x0001|0x0004|0x0020)
			return
		}
	}()
}
func handleWindowAction(a string) bool {
	h := findSetupWindow()
	if h == 0 {
		return false
	}
	switch a {
	case "minimize":
		showW.Call(h, 6)
	case "maximize":
		showW.Call(h, 3)
	case "restore":
		showW.Call(h, 9)
	case "close":
		postW.Call(h, 0x0010, 0, 0)
	case "drag":
		releaseCapture.Call()
		sendMessage.Call(h, 0x00A1, 2, 0)
	default:
		return false
	}
	return true
}

func chooseInstallFolder() (string, error) {
	start := os.ExpandEnv(`%LOCALAPPDATA%\Programs\Ghost FTP`)
	esc := strings.ReplaceAll(start, "'", "''")
	ps := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms;$d=New-Object System.Windows.Forms.FolderBrowserDialog;$d.Description='Choose the Ghost FTP installation folder';$d.SelectedPath='%s';$d.ShowNewFolderButton=$true;if($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK){[Console]::Write($d.SelectedPath)}`, esc)
	out, err := exec.Command("powershell", "-NoProfile", "-STA", "-WindowStyle", "Hidden", "-Command", ps).Output()
	if err != nil {
		return "", err
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", fmt.Errorf("folder selection was cancelled")
	}
	return path, nil
}
