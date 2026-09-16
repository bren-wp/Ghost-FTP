//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

const (
	ghostFTPWindowClass        = "GhostFTP.NativeWindow"
	ghostFTPStartupTargetMagic = 0x47544650
	wmCopyData                 = 0x004A
	smtoAbortIfHung            = 0x0002
	swRestore                  = 9
)

var (
	findWindowW         = user32.NewProc("FindWindowW")
	sendMessageTimeoutW = user32.NewProc("SendMessageTimeoutW")
	setForegroundWindow = user32.NewProc("SetForegroundWindow")
)

type copyDataStruct struct {
	Data    uintptr
	Size    uint32
	Payload uintptr
}

// ForwardStartupTarget hands an already-validated custom-protocol URI to the
// existing Ghost FTP window. WM_COPYDATA keeps this local to the Windows user
// session and does not introduce a network listener, background service or
// persistent handoff file.
func ForwardStartupTarget(raw string) bool {
	if _, err := ParseStartupTarget(raw); err != nil {
		return false
	}
	className, err := syscall.UTF16PtrFromString(ghostFTPWindowClass)
	if err != nil {
		return false
	}
	hwnd, _, _ := findWindowW.Call(uintptr(unsafe.Pointer(className)), 0)
	if hwnd == 0 {
		return false
	}

	payload := []byte(raw)
	if len(payload) == 0 || len(payload) > maxStartupTargetLength {
		return false
	}
	packet := copyDataStruct{
		Data:    ghostFTPStartupTargetMagic,
		Size:    uint32(len(payload)),
		Payload: uintptr(unsafe.Pointer(&payload[0])),
	}
	var messageResult uintptr
	ok, _, _ := sendMessageTimeoutW.Call(
		hwnd,
		wmCopyData,
		0,
		uintptr(unsafe.Pointer(&packet)),
		smtoAbortIfHung,
		2000,
		uintptr(unsafe.Pointer(&messageResult)),
	)
	return ok != 0 && messageResult == 1
}

func handleStartupTargetHandoff(a *app, lParam uintptr) uintptr {
	if a == nil || lParam == 0 {
		return 0
	}
	packet := (*copyDataStruct)(unsafe.Pointer(lParam))
	if packet.Data != ghostFTPStartupTargetMagic || packet.Size == 0 || packet.Size > maxStartupTargetLength || packet.Payload == 0 {
		return 0
	}

	payload := unsafe.Slice((*byte)(unsafe.Pointer(packet.Payload)), int(packet.Size))
	target, err := ParseStartupTarget(string(payload))
	if err != nil {
		return 0
	}
	SetStartupTarget(target)
	if a.hwnd != 0 {
		showWindow.Call(a.hwnd, swRestore)
		setForegroundWindow.Call(a.hwnd)
	}
	if a.protocol != 0 {
		a.applyStartupTarget()
		a.updateProtocolControls()
		a.refineWorkspaceLayout()
	}
	return 1
}
