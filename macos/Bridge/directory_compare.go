package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"errors"
	pathpkg "path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/security"
)

var directoryCompareState struct {
	mu              sync.Mutex
	seq             uint64
	cancelRequested bool
	cancel          context.CancelFunc
	entries         []api.DirectoryComparisonEntry
	localBase       string
	remoteBase      string
}

func prepareDirectoryCompareOperation(clearResults bool) uint64 {
	directoryCompareState.mu.Lock()
	defer directoryCompareState.mu.Unlock()
	if directoryCompareState.cancel != nil {
		directoryCompareState.cancel()
	}
	directoryCompareState.seq++
	directoryCompareState.cancelRequested = false
	directoryCompareState.cancel = nil
	if clearResults {
		directoryCompareState.entries = nil
		directoryCompareState.localBase = ""
		directoryCompareState.remoteBase = ""
	}
	return directoryCompareState.seq
}

func beginDirectoryCompareOperation(seq uint64) (context.Context, bool) {
	directoryCompareState.mu.Lock()
	defer directoryCompareState.mu.Unlock()
	if seq == 0 || directoryCompareState.seq != seq || directoryCompareState.cancelRequested {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	directoryCompareState.cancel = cancel
	return ctx, true
}

func finishDirectoryCompareOperation(seq uint64) {
	directoryCompareState.mu.Lock()
	defer directoryCompareState.mu.Unlock()
	if directoryCompareState.seq != seq {
		return
	}
	if directoryCompareState.cancel != nil {
		directoryCompareState.cancel()
	}
	directoryCompareState.cancel = nil
}

func publishDirectoryCompare(seq uint64, localBase, remoteBase string, entries []api.DirectoryComparisonEntry) bool {
	directoryCompareState.mu.Lock()
	defer directoryCompareState.mu.Unlock()
	if directoryCompareState.seq != seq || directoryCompareState.cancelRequested {
		return false
	}
	directoryCompareState.entries = append([]api.DirectoryComparisonEntry(nil), entries...)
	directoryCompareState.localBase = localBase
	directoryCompareState.remoteBase = remoteBase
	return true
}

func requireDirectoryCompareSnapshot(localBase, remoteBase string) (*api.Engine, string, string, error) {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		return nil, "", "", errors.New("engine is not initialized")
	}
	if _, ok := bridgeState.engine.ActiveConnection(); !ok {
		return nil, "", "", errors.New("not connected")
	}
	localBase = strings.TrimSpace(localBase)
	if localBase == "" || bridgeState.localPath == "" || filepath.Clean(localBase) != filepath.Clean(bridgeState.localPath) {
		return nil, "", "", errors.New("local folder changed; refresh and try again")
	}
	remoteBase = cleanRemotePath(remoteBase)
	if bridgeState.remotePath == "" || cleanRemotePath(bridgeState.remotePath) != remoteBase {
		return nil, "", "", errors.New("remote folder changed; refresh and try again")
	}
	return bridgeState.engine, bridgeState.localPath, remoteBase, nil
}

func setDirectoryCompareError(err error, fallback string) {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	setBridgeError(err, fallback)
}

// GhostFTPPrepareDirectoryCompare arms cancellation before the compare is
// queued on the serialized engine worker. A Cancel/Close/Disconnect that
// happens while the worker is still busy is therefore latched instead of lost.
//export GhostFTPPrepareDirectoryCompare
func GhostFTPPrepareDirectoryCompare() C.ulonglong {
	return C.ulonglong(prepareDirectoryCompareOperation(true))
}

//export GhostFTPCompareDirectories
func GhostFTPCompareDirectories(operationToken C.ulonglong, localBaseValue, remoteBaseValue *C.char) C.int {
	seq := uint64(operationToken)
	ctx, ok := beginDirectoryCompareOperation(seq)
	if !ok {
		return 2
	}
	defer finishDirectoryCompareOperation(seq)

	engine, localBase, remoteBase, err := requireDirectoryCompareSnapshot(goString(localBaseValue), goString(remoteBaseValue))
	if err != nil {
		setDirectoryCompareError(err, "Ghost FTP could not start directory comparison.")
		return 0
	}
	localResolved, localItems, err := engine.LocalList(ctx, localBase)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return 2
		}
		setDirectoryCompareError(err, "Ghost FTP could not read the local folder for comparison.")
		return 0
	}
	remoteItems, err := engine.RemoteList(ctx, remoteBase)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return 2
		}
		setDirectoryCompareError(err, "Ghost FTP could not read the remote folder for comparison.")
		return 0
	}
	entries, err := engine.CompareDirectoryItems(localItems, remoteItems, api.DirectoryComparisonOptions{})
	if err != nil {
		setDirectoryCompareError(err, "Ghost FTP could not compare these folders.")
		return 0
	}
	if ctx.Err() != nil || !publishDirectoryCompare(seq, localResolved, remoteBase, entries) {
		return 2
	}
	setDirectoryCompareError(nil, "")
	return 1
}

//export GhostFTPCancelDirectoryCompare
func GhostFTPCancelDirectoryCompare() {
	directoryCompareState.mu.Lock()
	defer directoryCompareState.mu.Unlock()
	directoryCompareState.cancelRequested = true
	if directoryCompareState.cancel != nil {
		directoryCompareState.cancel()
	}
}

func directoryCompareEntryAt(index C.int) (api.DirectoryComparisonEntry, bool) {
	directoryCompareState.mu.Lock()
	defer directoryCompareState.mu.Unlock()
	i := int(index)
	if i < 0 || i >= len(directoryCompareState.entries) {
		return api.DirectoryComparisonEntry{}, false
	}
	return directoryCompareState.entries[i], true
}

//export GhostFTPDirectoryCompareCount
func GhostFTPDirectoryCompareCount() C.int {
	directoryCompareState.mu.Lock()
	defer directoryCompareState.mu.Unlock()
	return C.int(len(directoryCompareState.entries))
}

//export GhostFTPDirectoryCompareName
func GhostFTPDirectoryCompareName(index C.int) *C.char {
	entry, ok := directoryCompareEntryAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(entry.Name)
}

//export GhostFTPDirectoryCompareStatus
func GhostFTPDirectoryCompareStatus(index C.int) *C.char {
	entry, ok := directoryCompareEntryAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(string(entry.Status))
}

//export GhostFTPDirectoryCompareHasLocal
func GhostFTPDirectoryCompareHasLocal(index C.int) C.int {
	entry, ok := directoryCompareEntryAt(index)
	if ok && entry.HasLocal {
		return 1
	}
	return 0
}

//export GhostFTPDirectoryCompareHasRemote
func GhostFTPDirectoryCompareHasRemote(index C.int) C.int {
	entry, ok := directoryCompareEntryAt(index)
	if ok && entry.HasRemote {
		return 1
	}
	return 0
}

//export GhostFTPDirectoryCompareLocalSize
func GhostFTPDirectoryCompareLocalSize(index C.int) C.longlong {
	entry, ok := directoryCompareEntryAt(index)
	if !ok || !entry.HasLocal {
		return 0
	}
	return C.longlong(entry.Local.Size)
}

//export GhostFTPDirectoryCompareRemoteSize
func GhostFTPDirectoryCompareRemoteSize(index C.int) C.longlong {
	entry, ok := directoryCompareEntryAt(index)
	if !ok || !entry.HasRemote {
		return 0
	}
	return C.longlong(entry.Remote.Size)
}

//export GhostFTPDirectoryCompareLocalModifiedUnix
func GhostFTPDirectoryCompareLocalModifiedUnix(index C.int) C.longlong {
	entry, ok := directoryCompareEntryAt(index)
	if !ok || !entry.HasLocal {
		return 0
	}
	return itemModifiedUnix(entry.Local)
}

//export GhostFTPDirectoryCompareRemoteModifiedUnix
func GhostFTPDirectoryCompareRemoteModifiedUnix(index C.int) C.longlong {
	entry, ok := directoryCompareEntryAt(index)
	if !ok || !entry.HasRemote {
		return 0
	}
	return itemModifiedUnix(entry.Remote)
}

func synchronizedDirectoryForOpen(index C.int) (*api.Engine, string, string, string, error) {
	directoryCompareState.mu.Lock()
	i := int(index)
	if i < 0 || i >= len(directoryCompareState.entries) {
		directoryCompareState.mu.Unlock()
		return nil, "", "", "", errors.New("directory comparison selection is no longer valid")
	}
	entry := directoryCompareState.entries[i]
	entries := append([]api.DirectoryComparisonEntry(nil), directoryCompareState.entries...)
	localBase := directoryCompareState.localBase
	remoteBase := directoryCompareState.remoteBase
	directoryCompareState.mu.Unlock()

	engine, currentLocal, currentRemote, err := requireDirectoryCompareSnapshot(localBase, remoteBase)
	if err != nil {
		return nil, "", "", "", err
	}
	name, ok := engine.SynchronizedDirectoryName(entries, entry.Name)
	if !ok {
		return nil, "", "", "", errors.New("selected item is not a synchronized ordinary directory")
	}
	return engine, currentLocal, currentRemote, name, nil
}

//export GhostFTPDirectoryCompareCanOpenBoth
func GhostFTPDirectoryCompareCanOpenBoth(index C.int) C.int {
	if _, _, _, _, err := synchronizedDirectoryForOpen(index); err == nil {
		return 1
	}
	return 0
}

// GhostFTPPrepareDirectoryCompareOpen retains the completed comparison result
// while arming cancellation for the queued atomic Open Both operation.
//export GhostFTPPrepareDirectoryCompareOpen
func GhostFTPPrepareDirectoryCompareOpen() C.ulonglong {
	return C.ulonglong(prepareDirectoryCompareOperation(false))
}

func commitComparedDirectories(seq uint64, engine *api.Engine, localBase, remoteBase, localResolved, remoteTarget string, localItems, remoteItems []model.Item) bool {
	directoryCompareState.mu.Lock()
	defer directoryCompareState.mu.Unlock()
	if directoryCompareState.seq != seq || directoryCompareState.cancelRequested {
		return false
	}

	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine != engine || bridgeState.engine == nil {
		return false
	}
	if _, ok := bridgeState.engine.ActiveConnection(); !ok {
		return false
	}
	if filepath.Clean(bridgeState.localPath) != filepath.Clean(localBase) || cleanRemotePath(bridgeState.remotePath) != cleanRemotePath(remoteBase) {
		return false
	}
	bridgeState.localPath = localResolved
	bridgeState.localItems = append(bridgeState.localItems[:0], localItems...)
	bridgeState.remotePath = remoteTarget
	bridgeState.remoteItems = append(bridgeState.remoteItems[:0], remoteItems...)
	updateLocalFilterLocked("")
	updateRemoteFilterLocked("")
	bridgeState.lastError = ""
	return true
}

// GhostFTPOpenComparedDirectoryBoth stages both listings and commits neither
// visible snapshot unless both reads and the final stale-snapshot check pass.
//export GhostFTPOpenComparedDirectoryBoth
func GhostFTPOpenComparedDirectoryBoth(operationToken C.ulonglong, index C.int) C.int {
	seq := uint64(operationToken)
	ctx, ok := beginDirectoryCompareOperation(seq)
	if !ok {
		return 2
	}
	defer finishDirectoryCompareOperation(seq)

	engine, localBase, remoteBase, name, err := synchronizedDirectoryForOpen(index)
	if err != nil {
		setDirectoryCompareError(err, "Ghost FTP could not open the compared directory.")
		return 0
	}
	localTarget, err := security.SafeLocalChild(localBase, name)
	if err != nil {
		setDirectoryCompareError(err, "Ghost FTP could not safely open the local directory.")
		return 0
	}
	if err := security.ValidateRemoteName(name); err != nil {
		setDirectoryCompareError(err, "Ghost FTP could not safely open the remote directory.")
		return 0
	}
	remoteTarget := pathpkg.Clean(pathpkg.Join(remoteBase, name))
	if err := security.ValidateRemotePath(remoteTarget); err != nil {
		setDirectoryCompareError(err, "Ghost FTP could not safely open the remote directory.")
		return 0
	}

	localResolved, localItems, err := engine.LocalList(ctx, localTarget)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return 2
		}
		setDirectoryCompareError(err, "Ghost FTP could not read the compared local directory.")
		return 0
	}
	remoteItems, err := engine.RemoteList(ctx, remoteTarget)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return 2
		}
		setDirectoryCompareError(err, "Ghost FTP could not read the compared remote directory.")
		return 0
	}
	if ctx.Err() != nil || !commitComparedDirectories(seq, engine, localBase, remoteBase, localResolved, remoteTarget, localItems, remoteItems) {
		return 2
	}
	setDirectoryCompareError(nil, "")
	return 1
}

//export GhostFTPDirectoryCompareLocalBase
func GhostFTPDirectoryCompareLocalBase() *C.char {
	directoryCompareState.mu.Lock()
	defer directoryCompareState.mu.Unlock()
	return C.CString(directoryCompareState.localBase)
}

//export GhostFTPDirectoryCompareRemoteBase
func GhostFTPDirectoryCompareRemoteBase() *C.char {
	directoryCompareState.mu.Lock()
	defer directoryCompareState.mu.Unlock()
	return C.CString(directoryCompareState.remoteBase)
}

//export GhostFTPClearDirectoryCompare
func GhostFTPClearDirectoryCompare() {
	directoryCompareState.mu.Lock()
	defer directoryCompareState.mu.Unlock()
	if directoryCompareState.cancel != nil {
		directoryCompareState.cancel()
	}
	directoryCompareState.cancel = nil
	directoryCompareState.cancelRequested = true
	directoryCompareState.seq++
	directoryCompareState.entries = nil
	directoryCompareState.localBase = ""
	directoryCompareState.remoteBase = ""
}
