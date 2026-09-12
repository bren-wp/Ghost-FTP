package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/security"
)

func requireLocalSnapshot(base string) (string, error) {
	base = filepath.Clean(strings.TrimSpace(base))
	if base == "." || base == "" {
		return "", errors.New("local folder is unavailable")
	}
	if filepath.Clean(bridgeState.localPath) != base {
		return "", errors.New("local folder changed; refresh and try again")
	}
	return base, nil
}

func requireRemoteSnapshot(base string) (string, error) {
	base = cleanRemotePath(base)
	if cleanRemotePath(bridgeState.remotePath) != base {
		return "", errors.New("remote folder changed; refresh and try again")
	}
	if bridgeState.engine == nil {
		return "", errors.New("engine is not initialized")
	}
	if _, ok := bridgeState.engine.ActiveConnection(); !ok {
		return "", errors.New("not connected")
	}
	return base, nil
}

func requireVisibleLocalItem(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("selected local item is unavailable")
	}
	if _, ok := snapshotItem(bridgeState.localItems, name); !ok {
		return errors.New("selected item is no longer in the visible local folder")
	}
	return nil
}

func requireVisibleRemoteItem(name string) (bool, error) {
	name = strings.TrimSpace(name)
	if err := security.ValidateRemoteName(name); err != nil {
		return false, err
	}
	item, ok := snapshotItem(bridgeState.remoteItems, name)
	if !ok {
		return false, errors.New("selected item is no longer in the visible remote folder")
	}
	return item.IsDirectory, nil
}

//export GhostFTPLocalMkdir
func GhostFTPLocalMkdir(baseValue, nameValue *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	base, err := requireLocalSnapshot(goString(baseValue))
	if err != nil {
		setBridgeError(err, "Ghost FTP could not create the local folder.")
		return 0
	}
	name := strings.TrimSpace(goString(nameValue))
	if err := bridgeState.engine.LocalMkdir(base, name); err != nil {
		setBridgeError(err, "Ghost FTP could not create the local folder.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPLocalRename
func GhostFTPLocalRename(baseValue, oldNameValue, newNameValue *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	base, err := requireLocalSnapshot(goString(baseValue))
	if err != nil {
		setBridgeError(err, "Ghost FTP could not rename the local item.")
		return 0
	}
	oldName := strings.TrimSpace(goString(oldNameValue))
	if err := requireVisibleLocalItem(oldName); err != nil {
		setBridgeError(err, "Ghost FTP could not rename the local item.")
		return 0
	}
	newName := strings.TrimSpace(goString(newNameValue))
	if err := bridgeState.engine.LocalRename(base, oldName, newName); err != nil {
		setBridgeError(err, "Ghost FTP could not rename the local item.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPLocalDelete
func GhostFTPLocalDelete(baseValue, nameValue *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	base, err := requireLocalSnapshot(goString(baseValue))
	if err != nil {
		setBridgeError(err, "Ghost FTP could not delete the local item.")
		return 0
	}
	name := strings.TrimSpace(goString(nameValue))
	if err := requireVisibleLocalItem(name); err != nil {
		setBridgeError(err, "Ghost FTP could not delete the local item.")
		return 0
	}
	if err := bridgeState.engine.LocalDelete(base, name); err != nil {
		setBridgeError(err, "Ghost FTP could not delete the local item.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPRemoteMkdir
func GhostFTPRemoteMkdir(baseValue, nameValue *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	base, err := requireRemoteSnapshot(goString(baseValue))
	if err != nil {
		setBridgeError(err, "Ghost FTP could not create the remote folder.")
		return 0
	}
	name := strings.TrimSpace(goString(nameValue))
	if err := security.ValidateRemoteName(name); err != nil {
		setBridgeError(err, "Ghost FTP could not create the remote folder.")
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := bridgeState.engine.RemoteMkdir(ctx, base, name); err != nil {
		setBridgeError(err, "Ghost FTP could not create the remote folder.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPRemoteRename
func GhostFTPRemoteRename(baseValue, oldNameValue, newNameValue *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	base, err := requireRemoteSnapshot(goString(baseValue))
	if err != nil {
		setBridgeError(err, "Ghost FTP could not rename the remote item.")
		return 0
	}
	oldName := strings.TrimSpace(goString(oldNameValue))
	if _, err := requireVisibleRemoteItem(oldName); err != nil {
		setBridgeError(err, "Ghost FTP could not rename the remote item.")
		return 0
	}
	newName := strings.TrimSpace(goString(newNameValue))
	if err := security.ValidateRemoteName(newName); err != nil {
		setBridgeError(err, "Ghost FTP could not rename the remote item.")
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := bridgeState.engine.RemoteRename(ctx, base, oldName, newName); err != nil {
		setBridgeError(err, "Ghost FTP could not rename the remote item.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPRemoteDelete
func GhostFTPRemoteDelete(baseValue, nameValue *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	base, err := requireRemoteSnapshot(goString(baseValue))
	if err != nil {
		setBridgeError(err, "Ghost FTP could not delete the remote item.")
		return 0
	}
	name := strings.TrimSpace(goString(nameValue))
	isDirectory, err := requireVisibleRemoteItem(name)
	if err != nil {
		setBridgeError(err, "Ghost FTP could not delete the remote item.")
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := bridgeState.engine.RemoteDelete(ctx, base, name, isDirectory); err != nil {
		setBridgeError(err, "Ghost FTP could not delete the remote item.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}
