//go:build linux

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

const prSetDumpable = 4

func LocalAppData() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("user home directory is unavailable")
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(xdg) {
		return filepath.Clean(xdg), nil
	}
	return filepath.Join(home, ".local", "share"), nil
}

func SystemDirectory() (string, error) { return "", errors.New("Windows system directory is unavailable on Linux") }

// HardenProcessPrivacy is deliberately called before any application state or
// credential material is created. A restrictive umask protects newly-created
// files even if a later call site forgets to pass a private mode. PR_SET_DUMPABLE
// prevents ordinary same-UID process-memory inspection and automatic core dumps
// of the credential-bearing client process. Failure to apply the optional
// prctl hardening is non-fatal because some sandboxed runtimes block prctl.
func HardenProcessPrivacy() {
	syscall.Umask(0o077)
	_, _, _ = syscall.Syscall6(syscall.SYS_PRCTL, prSetDumpable, 0, 0, 0, 0, 0)
}

func TrustedAskPassParent() bool { return false }

func ChoosePrivateKey() (string, error) {
	return "", errors.New("private-key selection is provided by the Linux desktop UI")
}
func ChooseDirectory() (string, error) {
	return "", errors.New("directory selection is provided by the Linux desktop UI")
}
func MessageBox(title, text string, flags uintptr) int {
	_, _ = os.Stderr.WriteString(title + ": " + text + "\n")
	return 0
}
func ConfirmDialog(string, string, string) bool { return false }
func InfoDialog(title, instruction, content string) {
	_, _ = os.Stdout.WriteString(title + ": " + instruction + "\n" + content + "\n")
}
func ErrorDialog(title, instruction, content string) {
	_, _ = os.Stderr.WriteString(title + ": " + instruction + "\n" + content + "\n")
}

func privateRuntimeDirectory() (string, error) {
	uid := os.Geteuid()
	if candidate := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); candidate != "" && filepath.IsAbs(candidate) {
		candidate = filepath.Clean(candidate)
		if err := validatePrivateRuntimeDirectory(candidate, uid); err == nil {
			return candidate, nil
		}
	}

	base := filepath.Join(os.TempDir(), "ghostftp-"+strconv.Itoa(uid))
	if err := os.Mkdir(base, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return "", err
	}
	if err := validatePrivateRuntimeDirectory(base, uid); err != nil {
		return "", err
	}
	return base, nil
}

func validatePrivateRuntimeDirectory(path string, uid int) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("runtime directory is not a private real directory")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != uid {
		return errors.New("runtime directory is not owned by the current user")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return errors.New("runtime directory permissions are too broad")
	}
	return nil
}

func lockFileName(identity string) string {
	var b strings.Builder
	b.WriteString("ghostftp-")
	for _, r := range strings.ToLower(identity) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	b.WriteString(".lock")
	return b.String()
}

// AcquireSingleInstance uses an owner-only advisory lock rather than a process
// name scan. O_NOFOLLOW plus post-open identity/ownership validation blocks a
// symlink substitution at the lock-file boundary. The descriptor is kept open
// for the complete application lifetime so a crashed process automatically
// releases the kernel lock.
func AcquireSingleInstance(identity string) (func(), bool) {
	runtimeDir, err := privateRuntimeDirectory()
	if err != nil {
		return func() {}, false
	}
	lockPath := filepath.Join(runtimeDir, lockFileName(identity))
	fd, err := syscall.Open(lockPath, syscall.O_RDWR|syscall.O_CREAT|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return func() {}, false
	}
	file := os.NewFile(uintptr(fd), lockPath)
	if file == nil {
		_ = syscall.Close(fd)
		return func() {}, false
	}
	fail := func() (func(), bool) {
		_ = file.Close()
		return func() {}, false
	}

	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return fail()
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Geteuid() {
		return fail()
	}
	if info.Mode().Perm() != 0o600 {
		if err := file.Chmod(0o600); err != nil {
			return fail()
		}
		info, err = file.Stat()
		if err != nil || info.Mode().Perm() != 0o600 {
			return fail()
		}
	}
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fail()
	}

	var once sync.Once
	release := func() {
		once.Do(func() {
			_ = syscall.Flock(fd, syscall.LOCK_UN)
			_ = file.Close()
			current, currentErr := os.Lstat(lockPath)
			if currentErr == nil && os.SameFile(info, current) {
				_ = os.Remove(lockPath)
			}
		})
	}
	return release, true
}
