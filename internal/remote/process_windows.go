//go:build windows

package remote

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

var (
	processKernel32          = syscall.NewLazyDLL("kernel32.dll")
	createJobObjectW         = processKernel32.NewProc("CreateJobObjectW")
	assignProcessToJobObject = processKernel32.NewProc("AssignProcessToJobObject")
	terminateJobObject       = processKernel32.NewProc("TerminateJobObject")
	openProcessForJob        = processKernel32.NewProc("OpenProcess")
	closeProcessHandle       = processKernel32.NewProc("CloseHandle")
)

const (
	createNoWindow     = 0x08000000
	processTerminate   = 0x0001
	processSetQuota    = 0x0100
	toolProcessAccess  = processTerminate | processSetQuota
)

func configureToolCommand(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	// External Windows networking tools are implementation details of the GUI.
	// Keep them detached from a visible console window while preserving the
	// standard handles required for stdin/stdout/stderr and AskPass.
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

func callWindowsProcessAPI(ok uintptr, callErr error, fallback string) error {
	if ok != 0 {
		return nil
	}
	if callErr != nil && callErr != syscall.Errno(0) {
		return callErr
	}
	return errors.New(fallback)
}

func createToolJob() (syscall.Handle, error) {
	h, _, callErr := createJobObjectW.Call(0, 0)
	if h == 0 {
		return 0, callWindowsProcessAPI(h, callErr, "Windows Job Object nije dostupan")
	}
	return syscall.Handle(h), nil
}

func closeToolHandle(h syscall.Handle) {
	if h != 0 {
		_, _, _ = closeProcessHandle.Call(uintptr(h))
	}
}

func openToolProcess(pid int) (syscall.Handle, error) {
	if pid <= 0 {
		return 0, errors.New("vanjski proces nema ispravan PID")
	}
	h, _, callErr := openProcessForJob.Call(toolProcessAccess, 0, uintptr(uint32(pid)))
	if h == 0 {
		return 0, callWindowsProcessAPI(h, callErr, "vanjski proces nije moguće otvoriti za lifecycle zaštitu")
	}
	return syscall.Handle(h), nil
}

func assignToolProcess(job, process syscall.Handle) error {
	r, _, callErr := assignProcessToJobObject.Call(uintptr(job), uintptr(process))
	return callWindowsProcessAPI(r, callErr, "vanjski proces nije moguće vezati uz Windows Job Object")
}

func terminateToolJob(job syscall.Handle) error {
	if job == 0 {
		return os.ErrProcessDone
	}
	r, _, callErr := terminateJobObject.Call(uintptr(job), 1)
	return callWindowsProcessAPI(r, callErr, "Windows Job Object nije moguće prekinuti")
}

func runToolCommand(cmd *exec.Cmd) error {
	if cmd == nil {
		return errors.New("vanjski proces nije postavljen")
	}
	job, err := createToolJob()
	if err != nil {
		return fmt.Errorf("siguran lifecycle vanjskog procesa nije dostupan: %w", err)
	}
	defer closeToolHandle(job)

	originalCancel := cmd.Cancel
	cmd.Cancel = func() error {
		jobErr := terminateToolJob(job)
		var directErr error
		if originalCancel != nil {
			directErr = originalCancel()
		} else if cmd.Process != nil {
			directErr = cmd.Process.Kill()
		}
		// Either path is sufficient to wake Wait. The Job Object is preferred
		// because it owns descendants; direct kill closes the tiny Start->Assign
		// race before the process has been attached to the job.
		if jobErr == nil || directErr == nil || errors.Is(directErr, os.ErrProcessDone) {
			return nil
		}
		return errors.Join(jobErr, directErr)
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	process, openErr := openToolProcess(cmd.Process.Pid)
	if openErr != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("vanjski proces nije pokrenut bez lifecycle zaštite: %w", openErr)
	}
	assignErr := assignToolProcess(job, process)
	closeToolHandle(process)
	if assignErr != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("vanjski proces nije pokrenut bez process-tree zaštite: %w", assignErr)
	}

	waitErr := cmd.Wait()
	// TerminateJobObject is also called after a normal parent exit. This removes
	// any helper that failed to exit with curl/OpenSSH before secrets and temp
	// session files are released by the caller.
	cleanupErr := terminateToolJob(job)
	return errors.Join(waitErr, cleanupErr)
}
