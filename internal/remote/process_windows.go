//go:build windows

package remote

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	processKernel32              = syscall.NewLazyDLL("kernel32.dll")
	createToolhelp32SnapshotProc = processKernel32.NewProc("CreateToolhelp32Snapshot")
	process32FirstWProc          = processKernel32.NewProc("Process32FirstW")
	process32NextWProc           = processKernel32.NewProc("Process32NextW")
	openProcessForTerminateProc  = processKernel32.NewProc("OpenProcess")
	terminateProcessProc         = processKernel32.NewProc("TerminateProcess")
	closeProcessHandleProc       = processKernel32.NewProc("CloseHandle")
)

const (
	createNoWindow      = 0x08000000
	th32csSnapProcess   = 0x00000002
	processTerminate    = 0x0001
	windowsNoMoreFiles  = syscall.Errno(18)
	windowsInvalidParam = syscall.Errno(87)
)

type processEntry32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

func configureToolCommand(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	// External Windows networking tools are implementation details of the GUI.
	// Keep them detached from a visible console window while preserving standard
	// handles required for stdin/stdout/stderr and SSH_ASKPASS.
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}

	originalCancel := cmd.Cancel
	cmd.Cancel = func() error {
		if cmd.Process == nil || cmd.Process.Pid <= 0 {
			if originalCancel != nil {
				return originalCancel()
			}
			return os.ErrProcessDone
		}
		rootPID := uint32(cmd.Process.Pid)

		// Snapshot before killing the parent closes the case where an AskPass child
		// already exists. Kill the direct process immediately afterwards so it can
		// no longer create additional descendants while cleanup proceeds.
		parents, snapshotErr := snapshotProcessTree()
		directErr := error(nil)
		if originalCancel != nil {
			directErr = originalCancel()
		} else {
			directErr = cmd.Process.Kill()
		}
		if errors.Is(directErr, os.ErrProcessDone) {
			directErr = nil
		}

		firstTreeErr := terminateDescendantsFromSnapshot(rootPID, parents)
		// A second fresh snapshot catches a descendant created in the tiny interval
		// between the first snapshot and termination of the trusted parent process.
		secondTreeErr := terminateProcessDescendants(rootPID)
		return errors.Join(snapshotErr, directErr, firstTreeErr, secondTreeErr)
	}
}

func windowsProcessAPIError(callErr error, fallback error) error {
	if callErr != nil && callErr != syscall.Errno(0) {
		return callErr
	}
	return fallback
}

func closeWindowsProcessHandle(h uintptr) {
	if h != 0 && h != ^uintptr(0) {
		_, _, _ = closeProcessHandleProc.Call(h)
	}
}

func snapshotProcessTree() (map[uint32][]uint32, error) {
	h, _, callErr := createToolhelp32SnapshotProc.Call(th32csSnapProcess, 0)
	if h == ^uintptr(0) {
		return nil, windowsProcessAPIError(callErr, errors.New("Windows popis procesa nije dostupan"))
	}
	defer closeWindowsProcessHandle(h)

	parents := make(map[uint32][]uint32)
	entry := processEntry32{Size: uint32(unsafe.Sizeof(processEntry32{}))}
	r, _, firstErr := process32FirstWProc.Call(h, uintptr(unsafe.Pointer(&entry)))
	if r == 0 {
		if errno, ok := firstErr.(syscall.Errno); ok && errno == windowsNoMoreFiles {
			return parents, nil
		}
		return nil, windowsProcessAPIError(firstErr, errors.New("Windows popis procesa nije moguće pročitati"))
	}
	for {
		if entry.ProcessID != 0 && entry.ProcessID != entry.ParentProcessID {
			parents[entry.ParentProcessID] = append(parents[entry.ParentProcessID], entry.ProcessID)
		}
		entry = processEntry32{Size: uint32(unsafe.Sizeof(processEntry32{}))}
		r, _, nextErr := process32NextWProc.Call(h, uintptr(unsafe.Pointer(&entry)))
		if r != 0 {
			continue
		}
		if errno, ok := nextErr.(syscall.Errno); ok && errno == windowsNoMoreFiles {
			break
		}
		return nil, windowsProcessAPIError(nextErr, errors.New("Windows popis procesa nije moguće dovršiti"))
	}
	return parents, nil
}

func terminateProcessPID(pid uint32) error {
	if pid == 0 {
		return nil
	}
	h, _, openErr := openProcessForTerminateProc.Call(processTerminate, 0, uintptr(pid))
	if h == 0 {
		if errno, ok := openErr.(syscall.Errno); ok && errno == windowsInvalidParam {
			return nil
		}
		return windowsProcessAPIError(openErr, errors.New("child proces nije moguće otvoriti za prekid"))
	}
	defer closeWindowsProcessHandle(h)
	r, _, terminateErr := terminateProcessProc.Call(h, 1)
	if r == 0 {
		if errno, ok := terminateErr.(syscall.Errno); ok && errno == windowsInvalidParam {
			return nil
		}
		return windowsProcessAPIError(terminateErr, errors.New("child proces nije moguće prekinuti"))
	}
	return nil
}

func terminateDescendantsFromSnapshot(root uint32, parents map[uint32][]uint32) error {
	if root == 0 || len(parents) == 0 {
		return nil
	}
	visited := make(map[uint32]bool)
	var errs []error
	var visit func(uint32)
	visit = func(parent uint32) {
		for _, child := range parents[parent] {
			if child == 0 || visited[child] {
				continue
			}
			visited[child] = true
			visit(child)
			if err := terminateProcessPID(child); err != nil {
				errs = append(errs, err)
			}
		}
	}
	visit(root)
	return errors.Join(errs...)
}

func terminateProcessDescendants(root uint32) error {
	parents, err := snapshotProcessTree()
	if err != nil {
		return err
	}
	return terminateDescendantsFromSnapshot(root, parents)
}
