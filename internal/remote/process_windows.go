//go:build windows

package remote

import (
	"os/exec"
	"syscall"
)

func configureToolCommand(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	// External Windows networking tools are implementation details of the GUI.
	// Keep them detached from a visible console window while preserving the
	// standard handles required for stdin/stdout/stderr and AskPass.
	const createNoWindow = 0x08000000
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}
