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
		groupErr := killToolProcessGroup(cmd)
		if groupErr == nil {
			return nil
		}
		if originalCancel == nil {
			return groupErr
		}
		directErr := originalCancel()
		if directErr == nil || errors.Is(directErr, os.ErrProcessDone) {
			return nil
		}
		return errors.Join(groupErr, directErr)
	}
}

func killToolProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil || cmd.Process.Pid <= 0 {
		return os.ErrProcessDone
	}
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return os.ErrProcessDone
	}
	return err
}

func runToolCommand(cmd *exec.Cmd) error {
	if cmd == nil {
		return errors.New("vanjski proces nije postavljen")
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	waitErr := cmd.Wait()
	// A helper can outlive a parent that exits normally. Kill any remaining
	// members before returning so secrets/prompts cannot survive the operation.
	cleanupErr := killToolProcessGroup(cmd)
	if errors.Is(cleanupErr, os.ErrProcessDone) {
		cleanupErr = nil
	}
	return errors.Join(waitErr, cleanupErr)
}
