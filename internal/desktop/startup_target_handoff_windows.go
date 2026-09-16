//go:build windows

package desktop

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

const (
	ghostFTPStartupReceiverClass = "GhostFTP.StartupTargetReceiver"
	ghostFTPStartupTargetMagic   = 0x47544650
	wmCopyData                   = 0x004A
	smtoAbortIfHung              = 0x0002
	swRestore                    = 9
)

var (
	findWindowW                  = user32.NewProc("FindWindowW")
	sendMessageTimeoutW          = user32.NewProc("SendMessageTimeoutW")
	setForegroundWindow          = user32.NewProc("SetForegroundWindow")
	startupTargetReceiverProcPtr = syscall.NewCallback(startupTargetReceiverProc)
)

type copyDataStruct struct {
	Data    uintptr
	Size    uint32
	Payload uintptr
}

// StartStartupTargetReceiver creates a hidden same-session Windows message
// receiver in the primary Ghost FTP process. It carries only the allowlisted
// non-secret ghostftp: payload and creates no socket, service or persistent IPC
// state.
func StartStartupTargetReceiver() func() {
	ready := make(chan uintptr, 1)
	done := make(chan struct{})

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(done)

		hinst, _, _ := getModuleHandleW.Call(0)
		className := wstr(ghostFTPStartupReceiverClass)
		wc := wndClassEx{
			CbSize:    uint32(unsafe.Sizeof(wndClassEx{})),
			WndProc:   startupTargetReceiverProcPtr,
			Instance:  hinst,
			ClassName: className,
		}
		if registered, _, _ := registerClassExW.Call(uintptr(unsafe.Pointer(&wc))); registered == 0 {
			ready <- 0
			return
		}

		hwnd, _, _ := createWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(wstr("Ghost FTP startup handoff"))),
			0,
			0, 0, 0, 0,
			0, 0, hinst, 0,
		)
		ready <- hwnd
		if hwnd == 0 {
			return
		}

		var message msg
		for {
			result, _, _ := getMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
			if int32(result) <= 0 {
				return
			}
			translateMessage.Call(uintptr(unsafe.Pointer(&message)))
			dispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
		}
	}()

	hwnd := <-ready
	if hwnd == 0 {
		<-done
		return func() {}
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			postMessageW.Call(hwnd, wmClose, 0, 0)
			<-done
		})
	}
}

// ForwardStartupTarget hands an already-validated custom-protocol URI to the
// primary Ghost FTP process. WM_COPYDATA keeps this local to the Windows user
// session and does not introduce a network listener, background service or
// persistent handoff file.
func ForwardStartupTarget(raw string) bool {
	if _, err := ParseStartupTarget(raw); err != nil {
		return false
	}
	className, err := syscall.UTF16PtrFromString(ghostFTPStartupReceiverClass)
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

func startupTargetReceiverProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case wmCopyData:
		return receiveStartupTargetCopyData(lParam)
	case wmClose:
		destroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		postQuitMessage.Call(0)
		return 0
	}
	result, _, _ := defWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func receiveStartupTargetCopyData(lParam uintptr) uintptr {
	if lParam == 0 {
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
	dispatchStartupTargetToOpenWindow()
	return 1
}

func dispatchStartupTargetToOpenWindow() {
	apps.Range(func(_, value any) bool {
		a, ok := value.(*app)
		if !ok || a == nil || a.hwnd == 0 {
			return true
		}
		a.dispatch(func() {
			showWindow.Call(a.hwnd, swRestore)
			setForegroundWindow.Call(a.hwnd)
			if a.applyStartupTarget() {
				a.updateProtocolControls()
				a.refineWorkspaceLayout()
			}
		})
		return false
	})
}
