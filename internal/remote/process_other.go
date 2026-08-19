//go:build linux || darwin

package remote

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func configureToolCommand(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	// Every external networking/helper command receives its own process group.
	// CommandContext normally kills only the direct child; OpenSSH may create an
	// AskPass descendant, so cancellation must terminate the whole group.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	originalCancel := cmd.Cancel
	cmd.Cancel = func() error {
		if cmd.Process != nil && cmd.Process.Pid > 0 {
			err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			if err == nil || errors.Is(err, syscall.ESRCH) {
				return nil
			}
			if originalCancel == nil {
				return err
			}
			directErr := originalCancel()
			if directErr == nil || errors.Is(directErr, os.ErrProcessDone) {
				return nil
			}
			return errors.Join(err, directErr)
		}
		if originalCancel != nil {
			return originalCancel()
		}
		return os.ErrProcessDone
	}
}
