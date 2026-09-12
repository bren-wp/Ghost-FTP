package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"errors"
	"os"
	pathpkg "path"
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
	localPath          string
	localItems         []model.Item
	remotePath         string
	remoteItems        []model.Item
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

func itemAt(items []model.Item, index C.int) (model.Item, bool) {
	i := int(index)
	if i < 0 || i >= len(items) {
		return model.Item{}, false
	}
	return items[i], true
}

func itemModifiedUnix(item model.Item) C.longlong {
	if item.Modified.IsZero() {
		return 0
	}
	return C.longlong(item.Modified.Unix())
}

func cleanRemotePath(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return "/"
	}
	return pathpkg.Clean(value)
}

func snapshotItem(items []model.Item, name string) (model.Item, bool) {
	for _, item := range items {
		if item.Name == name {
			return item, true
		}
	}
	return model.Item{}, false
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
	bridgeState.localPath = ""
	bridgeState.localItems = nil
	bridgeState.remotePath = ""
	bridgeState.remoteItems = nil
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
	bridgeState.remotePath = ""
	bridgeState.remoteItems = nil
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
	bridgeState.remotePath = ""
	bridgeState.remoteItems = nil
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

//export GhostFTPLocalList
func GhostFTPLocalList(requestedPath *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resolved, items, err := bridgeState.engine.LocalList(ctx, strings.TrimSpace(goString(requestedPath)))
	if err != nil {
		setBridgeError(err, "Ghost FTP could not read the local folder.")
		return 0
	}
	bridgeState.localPath = resolved
	bridgeState.localItems = append(bridgeState.localItems[:0], items...)
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPLocalPath
func GhostFTPLocalPath() *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	return C.CString(bridgeState.localPath)
}

//export GhostFTPLocalItemCount
func GhostFTPLocalItemCount() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	return C.int(len(bridgeState.localItems))
}

//export GhostFTPLocalItemName
func GhostFTPLocalItemName(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(bridgeState.localItems, index)
	if !ok {
		return C.CString("")
	}
	return C.CString(item.Name)
}

//export GhostFTPLocalItemSize
func GhostFTPLocalItemSize(index C.int) C.longlong {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(bridgeState.localItems, index)
	if !ok {
		return 0
	}
	return C.longlong(item.Size)
}

//export GhostFTPLocalItemIsDirectory
func GhostFTPLocalItemIsDirectory(index C.int) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(bridgeState.localItems, index)
	if ok && item.IsDirectory {
		return 1
	}
	return 0
}

//export GhostFTPLocalItemIsSymlink
func GhostFTPLocalItemIsSymlink(index C.int) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(bridgeState.localItems, index)
	if ok && item.IsSymlink {
		return 1
	}
	return 0
}

//export GhostFTPLocalItemModifiedUnix
func GhostFTPLocalItemModifiedUnix(index C.int) C.longlong {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(bridgeState.localItems, index)
	if !ok {
		return 0
	}
	return itemModifiedUnix(item)
}

//export GhostFTPRemoteList
func GhostFTPRemoteList(requestedPath *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	if _, ok := bridgeState.engine.ActiveConnection(); !ok {
		setBridgeError(errors.New("not connected"), "Connect to a server before browsing remote files.")
		return 0
	}
	target := cleanRemotePath(goString(requestedPath))
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	items, err := bridgeState.engine.RemoteList(ctx, target)
	if err != nil {
		setBridgeError(err, "Ghost FTP could not read the remote folder.")
		return 0
	}
	bridgeState.remotePath = target
	bridgeState.remoteItems = append(bridgeState.remoteItems[:0], items...)
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPRemotePath
func GhostFTPRemotePath() *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	return C.CString(bridgeState.remotePath)
}

//export GhostFTPRemoteItemCount
func GhostFTPRemoteItemCount() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	return C.int(len(bridgeState.remoteItems))
}

//export GhostFTPRemoteItemName
func GhostFTPRemoteItemName(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(bridgeState.remoteItems, index)
	if !ok {
		return C.CString("")
	}
	return C.CString(item.Name)
}

//export GhostFTPRemoteItemSize
func GhostFTPRemoteItemSize(index C.int) C.longlong {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(bridgeState.remoteItems, index)
	if !ok {
		return 0
	}
	return C.longlong(item.Size)
}

//export GhostFTPRemoteItemIsDirectory
func GhostFTPRemoteItemIsDirectory(index C.int) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(bridgeState.remoteItems, index)
	if ok && item.IsDirectory {
		return 1
	}
	return 0
}

//export GhostFTPRemoteItemIsSymlink
func GhostFTPRemoteItemIsSymlink(index C.int) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(bridgeState.remoteItems, index)
	if ok && item.IsSymlink {
		return 1
	}
	return 0
}

//export GhostFTPRemoteItemModifiedUnix
func GhostFTPRemoteItemModifiedUnix(index C.int) C.longlong {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(bridgeState.remoteItems, index)
	if !ok {
		return 0
	}
	return itemModifiedUnix(item)
}

//export GhostFTPRemoteItemPermissions
func GhostFTPRemoteItemPermissions(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(bridgeState.remoteItems, index)
	if !ok {
		return C.CString("")
	}
	return C.CString(item.Permissions)
}

func queueVisibleTransfer(direction, localBase, remoteBase, name string) error {
	if bridgeState.engine == nil {
		return errors.New("engine is not initialized")
	}
	name = strings.TrimSpace(name)
	if err := security.ValidateRemoteName(name); err != nil {
		return err
	}
	localBase = filepath.Clean(strings.TrimSpace(localBase))
	remoteBase = cleanRemotePath(remoteBase)
	if localBase == "." || localBase == "" {
		return errors.New("local folder is unavailable")
	}

	var item model.Item
	var ok bool
	if direction == "upload" {
		if filepath.Clean(bridgeState.localPath) != localBase {
			return errors.New("local folder changed; refresh and try again")
		}
		item, ok = snapshotItem(bridgeState.localItems, name)
	} else {
		if cleanRemotePath(bridgeState.remotePath) != remoteBase {
			return errors.New("remote folder changed; refresh and try again")
		}
		item, ok = snapshotItem(bridgeState.remoteItems, name)
	}
	if !ok {
		return errors.New("selected item is no longer in the visible folder")
	}
	if item.IsSymlink {
		return errors.New("symbolic links are not transferred")
	}

	localPath, err := security.SafeLocalChild(localBase, item.Name)
	if err != nil {
		return err
	}
	remotePath := pathpkg.Join(remoteBase, item.Name)
	if item.IsDirectory {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		_, err := bridgeState.engine.AddTreeTransfer(ctx, direction, localPath, remotePath)
		return err
	}
	_, err = bridgeState.engine.AddTransfer(direction, localPath, remotePath, localBase)
	return err
}

//export GhostFTPUpload
func GhostFTPUpload(localBase, remoteBase, name *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if err := queueVisibleTransfer("upload", goString(localBase), goString(remoteBase), goString(name)); err != nil {
		setBridgeError(err, "Ghost FTP could not queue the upload.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPDownload
func GhostFTPDownload(localBase, remoteBase, name *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if err := queueVisibleTransfer("download", goString(localBase), goString(remoteBase), goString(name)); err != nil {
		setBridgeError(err, "Ghost FTP could not queue the download.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
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
	bridgeState.localPath = ""
	bridgeState.localItems = nil
	bridgeState.remotePath = ""
	bridgeState.remoteItems = nil
}

//export GhostFTPFreeCString
func GhostFTPFreeCString(value *C.char) {
	if value != nil {
		C.free(unsafe.Pointer(value))
	}
}

func main() {}
