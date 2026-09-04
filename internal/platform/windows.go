//go:build windows

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	messageBox             = user32.NewProc("MessageBoxW")
	comdlg32               = syscall.NewLazyDLL("comdlg32.dll")
	getOpenFile            = comdlg32.NewProc("GetOpenFileNameW")
	shell32                = syscall.NewLazyDLL("shell32.dll")
	browseFolder           = shell32.NewProc("SHBrowseForFolderW")
	getPathID              = shell32.NewProc("SHGetPathFromIDListW")
	getKnownPath           = shell32.NewProc("SHGetKnownFolderPath")
	ole32                  = syscall.NewLazyDLL("ole32.dll")
	coTaskFree             = ole32.NewProc("CoTaskMemFree")
	comctl32               = syscall.NewLazyDLL("comctl32.dll")
	taskDialog             = comctl32.NewProc("TaskDialog")
	kernel32System         = syscall.NewLazyDLL("kernel32.dll")
	getSystemDirectory     = kernel32System.NewProc("GetSystemDirectoryW")
	setErrorMode           = kernel32System.NewProc("SetErrorMode")
	setDllDirectoryW       = kernel32System.NewProc("SetDllDirectoryW")
	openProcess            = kernel32System.NewProc("OpenProcess")
	queryFullProcessImageW = kernel32System.NewProc("QueryFullProcessImageNameW")
)

const (
	ofnExplorer        = 0x00080000
	ofnFileMustExist   = 0x00001000
	ofnPathMustExist   = 0x00000800
	ofnNoChangeDir     = 0x00000008
	ofnDontAddToRecent = 0x02000000
	bifReturnOnlyFS    = 0x00000001
	bifNewDialogStyle  = 0x00000040
)

type openFileName struct {
	StructSize      uint32
	Owner           uintptr
	Instance        uintptr
	Filter          *uint16
	CustomFilter    *uint16
	MaxCustomFilter uint32
	FilterIndex     uint32
	File            *uint16
	MaxFile         uint32
	FileTitle       *uint16
	MaxFileTitle    uint32
	InitialDir      *uint16
	Title           *uint16
	Flags           uint32
	FileOffset      uint16
	FileExtension   uint16
	DefExt          *uint16
	CustData        uintptr
	Hook            uintptr
	TemplateName    *uint16
	ReservedPtr     uintptr
	Reserved        uint32
	FlagsEx         uint32
}

type browseInfo struct {
	Owner       uintptr
	Root        uintptr
	DisplayName *uint16
	Title       *uint16
	Flags       uint32
	Callback    uintptr
	LParam      uintptr
	Image       int32
}

var folderIDLocalAppData = guid{
	Data1: 0xF1B32785,
	Data2: 0x6FBA,
	Data3: 0x4FCF,
	Data4: [8]byte{0x9D, 0x55, 0x7B, 0x8E, 0x7F, 0x15, 0x70, 0x91},
}

// LocalAppData resolves the Windows known folder directly instead of trusting
// the LOCALAPPDATA environment variable. This keeps profile/config storage
// anchored to the signed-in user's real Windows profile location.

// HardenProcessPrivacy prevents Windows Error Reporting from being invoked for
// this process. Child processes inherit the process error mode, so FTP/SFTP
// helpers do not trigger an external crash-reporting flow either.
func HardenProcessPrivacy() {
	const (
		semFailCriticalErrors = 0x0001
		semNoGPFaultErrorBox  = 0x0002
		semNoOpenFileErrorBox = 0x8000
	)
	setErrorMode.Call(semFailCriticalErrors | semNoGPFaultErrorBox | semNoOpenFileErrorBox)
	// GhostFTP has no private runtime DLL/plugin directory. Remove the current
	// working directory from the DLL search path to reduce DLL planting risk.
	empty, _ := syscall.UTF16PtrFromString("")
	if empty != nil {
		setDllDirectoryW.Call(uintptr(unsafe.Pointer(empty)))
	}
}

func LocalAppData() (string, error) {
	var pathPtr *uint16
	hr, _, callErr := getKnownPath.Call(
		uintptr(unsafe.Pointer(&folderIDLocalAppData)),
		0,
		0,
		uintptr(unsafe.Pointer(&pathPtr)),
	)
	if int32(hr) < 0 || pathPtr == nil {
		if callErr != nil && callErr != syscall.Errno(0) {
			return "", callErr
		}
		return "", errors.New("Windows korisnička mapa nije dostupna")
	}
	defer coTaskFree.Call(uintptr(unsafe.Pointer(pathPtr)))
	const maxKnownFolderChars = 32768
	chars := make([]uint16, 0, 260)
	base := unsafe.Pointer(pathPtr)
	for i := 0; i < maxKnownFolderChars; i++ {
		ch := *(*uint16)(unsafe.Add(base, uintptr(i)*unsafe.Sizeof(uint16(0))))
		if ch == 0 {
			if len(chars) == 0 {
				return "", errors.New("Windows korisnička mapa nije dostupna")
			}
			return syscall.UTF16ToString(chars), nil
		}
		chars = append(chars, ch)
	}
	return "", errors.New("Windows korisnička mapa je preduga")
}

// SystemDirectory resolves the real Windows system directory without trusting
// WINDIR/SystemRoot environment variables.
func SystemDirectory() (string, error) {
	buf := make([]uint16, 32768)
	n, _, callErr := getSystemDirectory.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 || n >= uintptr(len(buf)) {
		if callErr != nil && callErr != syscall.Errno(0) {
			return "", callErr
		}
		return "", errors.New("Windows sistemska mapa nije dostupna")
	}
	return syscall.UTF16ToString(buf[:n]), nil
}

// TrustedAskPassParent confirms that the AskPass helper was launched by the
// Windows OpenSSH transport from System32. This prevents a normal/manual GhostFTP
// launch with crafted AskPass environment variables from entering secret-output mode.
func TrustedAskPassParent() bool {
	ppid := os.Getppid()
	if ppid <= 0 {
		return false
	}
	const processQueryLimitedInformation = 0x1000
	h, _, _ := openProcess.Call(processQueryLimitedInformation, 0, uintptr(ppid))
	if h == 0 {
		return false
	}
	defer closeHandle.Call(h)
	buf := make([]uint16, 32768)
	size := uint32(len(buf))
	r, _, _ := queryFullProcessImageW.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r == 0 || size == 0 || int(size) > len(buf) {
		return false
	}
	parent := filepath.Clean(syscall.UTF16ToString(buf[:size]))
	systemDir, err := SystemDirectory()
	if err != nil {
		return false
	}
	for _, exe := range []string{"ssh.exe", "sftp.exe"} {
		expected := filepath.Clean(filepath.Join(systemDir, "OpenSSH", exe))
		if strings.EqualFold(parent, expected) {
			return true
		}
	}
	return false
}

func multiString(s string) []uint16 {
	runes := []rune(s)
	out := make([]uint16, 0, len(runes)+2)
	for _, r := range runes {
		if r == '|' {
			out = append(out, 0)
			continue
		}
		if r <= 0xffff {
			out = append(out, uint16(r))
		}
	}
	out = append(out, 0, 0)
	return out
}

func ChoosePrivateKey() (string, error) {
	buf := make([]uint16, 32768)
	filter := multiString("SSH privatni ključevi|id_*;*.pem;*.key|Sve datoteke|*.*")
	title, _ := syscall.UTF16PtrFromString("Odaberi SSH privatni ključ")
	ofn := openFileName{
		File:        &buf[0],
		MaxFile:     uint32(len(buf)),
		Filter:      &filter[0],
		FilterIndex: 1,
		Title:       title,
		Flags:       ofnExplorer | ofnFileMustExist | ofnPathMustExist | ofnNoChangeDir | ofnDontAddToRecent,
	}
	ofn.StructSize = uint32(unsafe.Sizeof(ofn))
	r, _, _ := getOpenFile.Call(uintptr(unsafe.Pointer(&ofn)))
	if r == 0 {
		return "", nil // cancel is not an error
	}
	return syscall.UTF16ToString(buf), nil
}

func ChooseDirectory() (string, error) {
	display := make([]uint16, 260)
	title, _ := syscall.UTF16PtrFromString("Odaberi lokalnu mapu za GhostFTP")
	bi := browseInfo{DisplayName: &display[0], Title: title, Flags: bifReturnOnlyFS | bifNewDialogStyle}
	pidl, _, _ := browseFolder.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return "", nil // cancel
	}
	defer coTaskFree.Call(pidl)
	path := make([]uint16, 32768)
	r, _, callErr := getPathID.Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if r == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return "", callErr
		}
		return "", errors.New("odabranu mapu nije moguće pročitati")
	}
	return syscall.UTF16ToString(path), nil
}

func taskDialogCall(title, instruction, content string, buttons uintptr) (int, bool) {
	t, _ := syscall.UTF16PtrFromString(title)
	i, _ := syscall.UTF16PtrFromString(instruction)
	c, _ := syscall.UTF16PtrFromString(content)
	var pressed int32
	hr, _, _ := taskDialog.Call(
		0, 0,
		uintptr(unsafe.Pointer(t)),
		uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(c)),
		buttons, 0, uintptr(unsafe.Pointer(&pressed)),
	)
	if int32(hr) < 0 {
		return 0, false
	}
	return int(pressed), true
}

func ConfirmDialog(title, instruction, content string) bool {
	const yesNo = 0x0002 | 0x0004
	if pressed, ok := taskDialogCall(title, instruction, content, yesNo); ok {
		return pressed == 6
	}
	return MessageBox(title, instruction+"\n\n"+content, 0x24) == 6
}

func InfoDialog(title, instruction, content string) {
	const okButton = 0x0001
	if _, ok := taskDialogCall(title, instruction, content, okButton); ok {
		return
	}
	MessageBox(title, instruction+"\n\n"+content, 0x40)
}

func ErrorDialog(title, instruction, content string) {
	const okButton = 0x0001
	if _, ok := taskDialogCall(title, instruction, content, okButton); ok {
		return
	}
	MessageBox(title, instruction+"\n\n"+content, 0x10)
}

func MessageBox(title, text string, flags uintptr) int {
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString(title)
	r, _, _ := messageBox.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), flags)
	return int(r)
}

var (
	kernel32Mutex = syscall.NewLazyDLL("kernel32.dll")
	createMutex   = kernel32Mutex.NewProc("CreateMutexW")
	closeHandle   = kernel32Mutex.NewProc("CloseHandle")
)

func AcquireSingleInstance(name string) (func(), bool) {
	n, err := syscall.UTF16PtrFromString("Local\\" + name)
	if err != nil {
		return func() {}, false
	}
	h, _, callErr := createMutex.Call(0, 0, uintptr(unsafe.Pointer(n)))
	if h == 0 {
		return func() {}, false
	}
	const errorAlreadyExists = syscall.Errno(183)
	if callErr == errorAlreadyExists {
		closeHandle.Call(h)
		return func() {}, false
	}
	return func() { closeHandle.Call(h) }, true
}
