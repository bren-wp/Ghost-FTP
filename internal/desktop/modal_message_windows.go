//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

const (
	desktopGARoot    = 2
	desktopWMKeyDown = 0x0100
	desktopVKEscape  = 0x1B
	desktopVKReturn  = 0x0D
	desktopBMClick   = 0x00F5
)

type modalAwareGetMessageProc struct {
	getMessage      *syscall.LazyProc
	getAncestor     *syscall.LazyProc
	getFocus        *syscall.LazyProc
	isWindowEnabled *syscall.LazyProc
	isDialogMessage *syscall.LazyProc
	sendMessage     *syscall.LazyProc
}

func newModalAwareGetMessageProc(user32 *syscall.LazyDLL) *modalAwareGetMessageProc {
	return &modalAwareGetMessageProc{
		getMessage:      user32.NewProc("GetMessageW"),
		getAncestor:     user32.NewProc("GetAncestor"),
		getFocus:        user32.NewProc("GetFocus"),
		isWindowEnabled: user32.NewProc("IsWindowEnabled"),
		isDialogMessage: user32.NewProc("IsDialogMessageW"),
		sendMessage:     user32.NewProc("SendMessageW"),
	}
}

func (p *modalAwareGetMessageProc) Call(messagePtr, hwndFilter, messageMin, messageMax uintptr) (uintptr, uintptr, error) {
	if p == nil || p.getMessage == nil {
		return 0, 0, syscall.EINVAL
	}

	result, aux, err := p.getMessage.Call(messagePtr, hwndFilter, messageMin, messageMax)
	if int32(result) <= 0 || messagePtr == 0 {
		return result, aux, err
	}

	message := (*msg)(unsafe.Pointer(messagePtr))
	root := p.modalRoot(message.Hwnd)
	if root == 0 {
		return result, aux, err
	}

	if p.handleModalShortcut(root, message) {
		*message = msg{}
		return result, aux, err
	}

	handled, _, _ := p.isDialogMessage.Call(root, messagePtr)
	if handled != 0 {
		// The nested caller must regain control after every consumed dialog
		// message so its closed flag is observed immediately. Returning a
		// harmless WM_NULL-style thread message avoids dispatching the same
		// keyboard input twice without hiding WM_QUIT/GetMessage errors.
		*message = msg{}
	}
	return result, aux, err
}

func (p *modalAwareGetMessageProc) modalRoot(hwnd uintptr) uintptr {
	if hwnd == 0 || p.getAncestor == nil {
		return 0
	}
	root, _, _ := p.getAncestor.Call(hwnd, desktopGARoot)
	if root == 0 {
		return 0
	}
	if _, ok := siteManagerStates.Load(root); ok {
		return root
	}
	if _, ok := bookmarkManagerStates.Load(root); ok {
		return root
	}
	return 0
}

func (p *modalAwareGetMessageProc) handleModalShortcut(root uintptr, message *msg) bool {
	if message == nil || message.Message != desktopWMKeyDown {
		return false
	}

	switch message.WParam {
	case desktopVKEscape:
		p.sendMessage.Call(root, wmClose, 0, 0)
		return true
	case desktopVKReturn:
		button := p.modalEnterButton(root)
		if button != 0 && p.windowEnabled(button) {
			p.sendMessage.Call(button, desktopBMClick, 0, 0)
		}
		return true
	default:
		return false
	}
}

func (p *modalAwareGetMessageProc) modalEnterButton(root uintptr) uintptr {
	focus := uintptr(0)
	if p.getFocus != nil {
		focus, _, _ = p.getFocus.Call()
	}

	if value, ok := siteManagerStates.Load(root); ok {
		state := value.(*siteManagerState)
		buttons := []uintptr{state.duplicate, state.save, state.delete, state.connect, state.close}
		for _, button := range buttons {
			if focus != 0 && focus == button {
				return button
			}
		}
		return state.connect
	}

	if value, ok := bookmarkManagerStates.Load(root); ok {
		state := value.(*bookmarkManagerState)
		buttons := []uintptr{state.open, state.addLocal, state.addRemote, state.delete, state.close}
		for _, button := range buttons {
			if focus != 0 && focus == button {
				return button
			}
		}
		return state.open
	}
	return 0
}

func (p *modalAwareGetMessageProc) windowEnabled(hwnd uintptr) bool {
	if hwnd == 0 || p.isWindowEnabled == nil {
		return false
	}
	enabled, _, _ := p.isWindowEnabled.Call(hwnd)
	return enabled != 0
}
