//go:build windows

package desktop

import (
	"sync"
	"syscall"
)

const swRestore = 9

type modalAwareEnableWindowProc struct {
	enable     *syscall.LazyProc
	isIconic   *syscall.LazyProc
	getParent  *syscall.LazyProc
	showWindow *syscall.LazyProc

	mu        sync.Mutex
	wasIconic map[uintptr]bool
}

func newModalAwareEnableWindowProc(user32 *syscall.LazyDLL) *modalAwareEnableWindowProc {
	return &modalAwareEnableWindowProc{
		enable:     user32.NewProc("EnableWindow"),
		isIconic:   user32.NewProc("IsIconic"),
		getParent:  user32.NewProc("GetParent"),
		showWindow: user32.NewProc("ShowWindow"),
		wasIconic:  make(map[uintptr]bool),
	}
}

func (p *modalAwareEnableWindowProc) Call(hwnd, enabled uintptr) (uintptr, uintptr, error) {
	if p == nil || p.enable == nil {
		return 0, 0, syscall.EINVAL
	}

	trackTopLevel := false
	if hwnd != 0 {
		parent, _, _ := p.getParent.Call(hwnd)
		trackTopLevel = parent == 0
	}

	if trackTopLevel && enabled == 0 {
		iconic, _, _ := p.isIconic.Call(hwnd)
		p.mu.Lock()
		if _, exists := p.wasIconic[hwnd]; !exists {
			p.wasIconic[hwnd] = iconic != 0
		}
		p.mu.Unlock()
	}

	r1, r2, err := p.enable.Call(hwnd, enabled)

	if trackTopLevel && enabled != 0 {
		p.mu.Lock()
		wasIconic, tracked := p.wasIconic[hwnd]
		delete(p.wasIconic, hwnd)
		p.mu.Unlock()
		if tracked && !wasIconic {
			iconic, _, _ := p.isIconic.Call(hwnd)
			if iconic != 0 {
				p.showWindow.Call(hwnd, swRestore)
			}
		}
	}

	return r1, r2, err
}
