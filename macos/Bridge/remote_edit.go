package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"errors"
	pathpkg "path"
	"strings"
	"time"
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/security"
)

var remoteEditBridgeState struct {
	text     string
	revision string
}

func clearRemoteEditBridgeStateLocked() {
	remoteEditBridgeState.text = ""
	remoteEditBridgeState.revision = ""
}

func remoteEditVisiblePathLocked(base, name string) (string, error) {
	if bridgeState.engine == nil {
		return "", errors.New("engine is not initialized")
	}
	if _, ok := bridgeState.engine.ActiveConnection(); !ok {
		return "", errors.New("not connected")
	}
	base = cleanRemotePath(base)
	if cleanRemotePath(bridgeState.remotePath) != base {
		return "", errors.New("remote folder changed; refresh and try again")
	}
	name = strings.TrimSpace(name)
	if err := security.ValidateRemoteName(name); err != nil {
		return "", err
	}
	item, ok := snapshotItem(bridgeState.remoteItems, name)
	if !ok {
		return "", errors.New("selected item is no longer in the visible folder")
	}
	if item.IsDirectory || item.IsSymlink {
		return "", errors.New("only regular remote files can be edited")
	}
	return pathpkg.Join(base, name), nil
}

//export GhostFTPRemoteEditOpen
func GhostFTPRemoteEditOpen(base, name *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	clearRemoteEditBridgeStateLocked()

	remotePath, err := remoteEditVisiblePathLocked(goString(base), goString(name))
	if err != nil {
		setBridgeError(err, "Ghost FTP could not open the remote file for editing.")
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	doc, err := bridgeState.engine.RemoteEditOpen(ctx, remotePath)
	if err != nil {
		setBridgeError(err, "Ghost FTP could not open the remote file for editing.")
		return 0
	}
	remoteEditBridgeState.text = doc.Text
	remoteEditBridgeState.revision = doc.Revision
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPRemoteEditText
func GhostFTPRemoteEditText() *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	value := C.CString(remoteEditBridgeState.text)
	remoteEditBridgeState.text = ""
	return value
}

//export GhostFTPRemoteEditRevision
func GhostFTPRemoteEditRevision() *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	return C.CString(remoteEditBridgeState.revision)
}

//export GhostFTPRemoteEditMaxBytes
func GhostFTPRemoteEditMaxBytes() C.longlong {
	return C.longlong(api.MaxRemoteEditBytes)
}

// GhostFTPRemoteEditSave returns 1 after a verified save, 2 for a revision
// conflict that requires an explicit reload decision, and 0 for other errors.
//export GhostFTPRemoteEditSave
func GhostFTPRemoteEditSave(base, name, expectedRevision *C.char, text unsafe.Pointer, textLen C.longlong) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()

	remotePath, err := remoteEditVisiblePathLocked(goString(base), goString(name))
	if err != nil {
		setBridgeError(err, "Ghost FTP could not save the remote file.")
		return 0
	}
	length := int64(textLen)
	if length < 0 || length > api.MaxRemoteEditBytes {
		setBridgeError(api.ErrRemoteEditTooLarge, "The edited remote file is too large to save safely.")
		return 0
	}
	if length > 0 && text == nil {
		setBridgeError(errors.New("edited content is unavailable"), "The edited content is unavailable.")
		return 0
	}
	if length > int64(^uint32(0)>>1) {
		setBridgeError(api.ErrRemoteEditTooLarge, "The edited remote file is too large to save safely.")
		return 0
	}
	var content []byte
	if length > 0 {
		content = C.GoBytes(text, C.int(length))
	} else {
		content = []byte{}
	}
	defer security.WipeBytes(content)

	revision := strings.TrimSpace(goString(expectedRevision))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	saved, err := bridgeState.engine.RemoteEditSave(ctx, remotePath, revision, string(content))
	if err != nil {
		if errors.Is(err, api.ErrRemoteEditConflict) {
			bridgeState.lastError = "The remote file changed after it was opened. Reload it before saving to avoid overwriting newer data."
			return 2
		}
		setBridgeError(err, "Ghost FTP could not save the remote file safely.")
		return 0
	}
	remoteEditBridgeState.text = ""
	remoteEditBridgeState.revision = saved.Revision
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPRemoteEditClear
func GhostFTPRemoteEditClear() {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	clearRemoteEditBridgeStateLocked()
}
