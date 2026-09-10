//go:build windows

package platform

import (
	"syscall"
	"unsafe"
)

var (
	kernel32Architecture = syscall.NewLazyDLL("kernel32.dll")
	getNativeSystemInfo  = kernel32Architecture.NewProc("GetNativeSystemInfo")
)

type windowsSystemInfo struct {
	ProcessorArchitecture uint16
	Reserved              uint16
	PageSize              uint32
	MinimumAppAddress     uintptr
	MaximumAppAddress     uintptr
	ActiveProcessorMask   uintptr
	NumberOfProcessors    uint32
	ProcessorType         uint32
	AllocationGranularity uint32
	ProcessorLevel        uint16
	ProcessorRevision     uint16
}

// NativeWindowsArchitecture returns the architecture of Windows itself rather
// than the architecture of this bootstrap process. The public bootstrap is an
// x86 PE so it can start on both supported x86 and x64 Windows systems; this
// function selects the matching native Ghost FTP payload without trusting
// mutable environment variables.
func NativeWindowsArchitecture() (string, error) {
	var info windowsSystemInfo
	getNativeSystemInfo.Call(uintptr(unsafe.Pointer(&info)))
	return windowsArchitectureFromProcessor(info.ProcessorArchitecture)
}
