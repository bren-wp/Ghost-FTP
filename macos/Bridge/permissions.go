package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/security"
)

var remoteChmodBatch struct {
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	base   string
	mode   string
}

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

func remoteChmodBatchTimeout(count int) time.Duration {
	if count < 1 {
		count = 1
	}
	timeout := 90*time.Second + time.Duration(count-1)*2*time.Second
	if timeout > 10*time.Minute {
		return 10 * time.Minute
	}
	return timeout
}

func cancelRemoteChmodBatch() {
	remoteChmodBatch.mu.Lock()
	cancel := remoteChmodBatch.cancel
	remoteChmodBatch.ctx = nil
	remoteChmodBatch.cancel = nil
	remoteChmodBatch.base = ""
	remoteChmodBatch.mode = ""
	remoteChmodBatch.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func currentRemoteChmodBatch() (context.Context, string, string, bool) {
	remoteChmodBatch.mu.Lock()
	defer remoteChmodBatch.mu.Unlock()
	if remoteChmodBatch.ctx == nil || remoteChmodBatch.cancel == nil {
		return nil, "", "", false
	}
	return remoteChmodBatch.ctx, remoteChmodBatch.base, remoteChmodBatch.mode, true
}

//export GhostFTPBeginRemoteChmodBatch
func GhostFTPBeginRemoteChmodBatch(baseValue, modeValue *C.char, countValue C.int) C.int {
	count := int(countValue)
	if count < 1 || count > 1000 {
		bridgeState.mu.Lock()
		setBridgeError(errors.New("invalid remote permissions batch size"), "Ghost FTP could not start the remote permissions change.")
		bridgeState.mu.Unlock()
		return 0
	}

	bridgeState.mu.Lock()
	base, err := requireRemoteSnapshot(goString(baseValue))
	if err != nil {
		setBridgeError(err, "Ghost FTP could not start the remote permissions change.")
		bridgeState.mu.Unlock()
		return 0
	}
	mode := strings.TrimSpace(goString(modeValue))
	bridgeState.lastError = ""
	bridgeState.mu.Unlock()

	cancelRemoteChmodBatch()
	ctx, cancel := context.WithTimeout(context.Background(), remoteChmodBatchTimeout(count))
	remoteChmodBatch.mu.Lock()
	remoteChmodBatch.ctx = ctx
	remoteChmodBatch.cancel = cancel
	remoteChmodBatch.base = base
	remoteChmodBatch.mode = mode
	remoteChmodBatch.mu.Unlock()
	return 1
}

//export GhostFTPRemoteChmodBatchItem
func GhostFTPRemoteChmodBatchItem(nameValue *C.char) C.int {
	ctx, base, mode, ok := currentRemoteChmodBatch()
	if !ok {
		bridgeState.mu.Lock()
		setBridgeError(errors.New("remote permissions batch is not active"), "Ghost FTP could not change remote permissions.")
		bridgeState.mu.Unlock()
		return 0
	}
	if err := ctx.Err(); err != nil {
		bridgeState.mu.Lock()
		setBridgeError(err, "Ghost FTP remote permissions change was cancelled or timed out.")
		bridgeState.mu.Unlock()
		return 0
	}

	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if _, err := requireRemoteSnapshot(base); err != nil {
		setBridgeError(err, "Ghost FTP could not change remote permissions.")
		return 0
	}
	name := strings.TrimSpace(goString(nameValue))
	if err := requireVisibleRemotePermissionItem(name); err != nil {
		setBridgeError(err, "Ghost FTP could not change remote permissions.")
		return 0
	}
	if err := bridgeState.engine.RemoteChmod(ctx, base, name, mode); err != nil {
		setBridgeError(err, "Ghost FTP could not change remote permissions.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPEndRemoteChmodBatch
func GhostFTPEndRemoteChmodBatch() {
	cancelRemoteChmodBatch()
}

//export GhostFTPCancelRemoteChmodBatch
func GhostFTPCancelRemoteChmodBatch() {
	cancelRemoteChmodBatch()
}
