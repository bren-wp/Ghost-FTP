package main

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

// The production build temporarily stages one signed x86 and one signed x64
// executable here before compiling each public bootstrap. The tracked .keep
// file keeps ordinary source/test checkouts buildable without generated files.
//
//go:embed all:payload
var payload embed.FS

const maxEmbeddedExecutableSize = 256 << 20

var version = "dev"
var role = "portable"

func normalizedRole(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "setup":
		return "setup", nil
	case "portable":
		return "portable", nil
	default:
		return "", errors.New("unsupported Windows bootstrap role")
	}
}

func payloadPath(arch string) (string, error) {
	switch arch {
	case "x86", "x64":
		return "payload/" + arch + "/GhostFTP.exe", nil
	default:
		return "", errors.New("unsupported Windows payload architecture")
	}
}

func readPayload(arch string) ([]byte, error) {
	path, err := payloadPath(arch)
	if err != nil {
		return nil, err
	}
	data, err := payload.ReadFile(path)
	if err != nil {
		return nil, errors.New("compatible Ghost FTP payload is unavailable")
	}
	if len(data) == 0 || len(data) > maxEmbeddedExecutableSize {
		return nil, errors.New("compatible Ghost FTP payload has an invalid size")
	}
	return data, nil
}

func verifyStaged(path string, expected []byte) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() != int64(len(expected)) {
		return errors.New("staged Ghost FTP payload has an invalid identity or size")
	}

	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, int64(len(expected))+1))
	if err != nil {
		return err
	}
	if n != int64(len(expected)) {
		return errors.New("staged Ghost FTP payload has an invalid size")
	}
	want := sha256.Sum256(expected)
	if !bytes.Equal(h.Sum(nil), want[:]) {
		return errors.New("staged Ghost FTP payload failed integrity verification")
	}
	return nil
}

func stagePayload(data []byte, selectedRole string) (string, func(), error) {
	localAppData, err := platform.LocalAppData()
	if err != nil {
		return "", nil, errors.New("Windows local application-data folder is unavailable")
	}

	f, err := os.CreateTemp(localAppData, ".ghostftp-"+selectedRole+"-*.exe")
	if err != nil {
		return "", nil, errors.New("compatible Ghost FTP payload could not be staged")
	}
	path := f.Name()
	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(path)
	}

	if err := f.Chmod(0700); err != nil {
		cleanup()
		return "", nil, err
	}
	n, err := io.Copy(f, bytes.NewReader(data))
	if err != nil || n != int64(len(data)) {
		cleanup()
		if err != nil {
			return "", nil, err
		}
		return "", nil, io.ErrShortWrite
	}
	if err := f.Sync(); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", nil, err
	}
	if err := verifyStaged(path, data); err != nil {
		_ = os.Remove(path)
		return "", nil, err
	}
	return path, cleanup, nil
}

func runSelected() error {
	selectedRole, err := normalizedRole(role)
	if err != nil {
		return err
	}
	arch, err := platform.NativeWindowsArchitecture()
	if err != nil {
		return err
	}
	data, err := readPayload(arch)
	if err != nil {
		return err
	}
	path, cleanup, err := stagePayload(data, selectedRole)
	if err != nil {
		return err
	}
	defer cleanup()

	cmd := exec.Command(path, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return fmt.Errorf("Ghost FTP %s exited with code %d", selectedRole, exitErr.ExitCode())
		}
		return err
	}
	return nil
}

func main() {
	platform.HardenProcessPrivacy()
	if err := runSelected(); err != nil {
		name := brand.ProductName
		if version != "dev" {
			name += " " + version
		}
		platform.ErrorDialog(
			name,
			"Ghost FTP could not start",
			"This Windows package could not start a compatible Ghost FTP component. Download a fresh copy and try again.",
		)
		os.Exit(1)
	}
}
