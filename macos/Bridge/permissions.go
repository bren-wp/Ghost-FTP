package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/security"
)

func requireVisibleRemotePermissionItem(name string) error {
	name = strings.TrimSpace(name)
	if err := security.ValidateRemoteName(name); err != nil {
		return err
	}
	item, ok := snapshotItem(bridgeState.remoteItems, name)
	if !ok {
		return errors.New("selected item is no longer in the visible remote folder")
	}
	if item.IsSymlink {
		return errors.New("symbolic-link permissions are not changed")
	}
	return nil
}

//export GhostFTPRemoteChmod
func GhostFTPRemoteChmod(baseValue, nameValue, modeValue *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	base, err := requireRemoteSnapshot(goString(baseValue))
	if err != nil {
		setBridgeError(err, "Ghost FTP could not change remote permissions.")
		return 0
	}
	name := strings.TrimSpace(goString(nameValue))
	if err := requireVisibleRemotePermissionItem(name); err != nil {
		setBridgeError(err, "Ghost FTP could not change remote permissions.")
		return 0
	}
	mode := strings.TrimSpace(goString(modeValue))
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := bridgeState.engine.RemoteChmod(ctx, base, name, mode); err != nil {
		setBridgeError(err, "Ghost FTP could not change remote permissions.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}
