package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
	"github.com/bren-wp/Ghost-FTP/internal/security"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
)

var bridgeState struct {
	mu                 sync.Mutex
	engine             *api.Engine
	lastError          string
	pendingFingerprint string
}

func setBridgeError(err error, fallback string) {
	if err == nil {
		bridgeState.lastError = ""
		return
	}
	bridgeState.lastError = usererror.Message(err, fallback)
}

func goString(value *C.char) string {
	if value == nil {
		return ""
	}
	return C.GoString(value)
}

func askPassHelperPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	path := filepath.Join(filepath.Dir(exe), "GhostFTPAskPass")
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 {
		return "", errors.New("bundled macOS authentication helper is unavailable")
	}
	return path, nil
}

//export GhostFTPCreateEngine
func GhostFTPCreateEngine() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine != nil {
		return 1
	}
	platform.HardenProcessPrivacy()
	dataDir, err := api.DataDir()
	if err != nil {
		setBridgeError(err, "Ghost FTP could not access its application data folder.")
		return 0
	}
	localAppData, err := platform.LocalAppData()
	if err != nil {
		setBridgeError(err, "Ghost FTP could not access the macOS Application Support folder.")
		return 0
	}
	if err := security.EnsureNoRedirectDirectory(localAppData, dataDir); err != nil {
		setBridgeError(err, "The Ghost FTP data folder is not safe to use.")
		return 0
	}
	helper, err := askPassHelperPath()
	if err != nil {
		setBridgeError(err, "Ghost FTP could not initialize secure SFTP authentication.")
		return 0
	}
	engine, err := api.New(dataDir, helper)
	if err != nil {
		setBridgeError(err, "Ghost FTP could not start its connection engine.")
		return 0
	}
	bridgeState.engine = engine
	bridgeState.lastError = ""
	bridgeState.pendingFingerprint = ""
	return 1
}

// GhostFTPConnect returns 1 for connected, 2 when SFTP host-key trust is
// required, and 0 for a user-safe failure exposed through GhostFTPLastError.
//export GhostFTPConnect
func GhostFTPConnect(protocol, host *C.char, port C.int, username, password, privateKeyPath, passphrase, trustFingerprint *C.char, rememberFingerprint C.int) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	cfg := model.ConnectionConfig{
		Protocol:       strings.ToLower(strings.TrimSpace(goString(protocol))),
		Host:           strings.TrimSpace(goString(host)),
		Port:           int(port),
		Username:       goString(username),
		Password:       goString(password),
		PrivateKeyPath: strings.TrimSpace(goString(privateKeyPath)),
		Passphrase:     goString(passphrase),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	result, err := bridgeState.engine.Connect(ctx, "", cfg, strings.TrimSpace(goString(trustFingerprint)), rememberFingerprint != 0)
	if err != nil {
		bridgeState.pendingFingerprint = ""
		setBridgeError(err, "Connection failed. Check the connection details and try again.")
		return 0
	}
	bridgeState.lastError = ""
	if result.RequiresTrust {
		bridgeState.pendingFingerprint = result.Fingerprint
		return 2
	}
	bridgeState.pendingFingerprint = ""
	if result.Connected {
		return 1
	}
	setBridgeError(errors.New("connection was not established"), "Connection failed. Please try again.")
	return 0
}

//export GhostFTPDisconnect
func GhostFTPDisconnect() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := bridgeState.engine.Disconnect(ctx); err != nil {
		setBridgeError(err, "Ghost FTP could not disconnect cleanly.")
		return 0
	}
	bridgeState.pendingFingerprint = ""
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPCancelPendingTrust
func GhostFTPCancelPendingTrust() {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine != nil {
		bridgeState.engine.CancelPendingTrust()
	}
	bridgeState.pendingFingerprint = ""
}

//export GhostFTPIsConnected
func GhostFTPIsConnected() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		return 0
	}
	_, ok := bridgeState.engine.ActiveConnection()
	if ok {
		return 1
	}
	return 0
}

//export GhostFTPLastError
func GhostFTPLastError() *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	return C.CString(bridgeState.lastError)
}

//export GhostFTPPendingFingerprint
func GhostFTPPendingFingerprint() *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	return C.CString(bridgeState.pendingFingerprint)
}

//export GhostFTPShutdown
func GhostFTPShutdown() {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine != nil {
		bridgeState.engine.CancelPendingTrust()
		bridgeState.engine.Close()
		bridgeState.engine = nil
	}
	bridgeState.pendingFingerprint = ""
	bridgeState.lastError = ""
}

//export GhostFTPFreeCString
func GhostFTPFreeCString(value *C.char) {
	if value != nil {
		C.free(unsafe.Pointer(value))
	}
}

func main() {}
