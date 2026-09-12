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
	"sync"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/api"
)

var directoryCompareState struct {
	mu         sync.Mutex
	seq        uint64
	entries    []api.DirectoryComparisonEntry
	localBase  string
	remoteBase string
}

var cancelDirectoryCompare context.CancelFunc

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

func beginDirectoryCompare() (context.Context, uint64) {
	directoryCompareState.mu.Lock()
	defer directoryCompareState.mu.Unlock()
	if cancelDirectoryCompare != nil {
		cancelDirectoryCompare()
	}
	directoryCompareState.seq++
	seq := directoryCompareState.seq
	directoryCompareState.entries = nil
	directoryCompareState.localBase = ""
	directoryCompareState.remoteBase = ""
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	cancelDirectoryCompare = cancel
	return ctx, seq
}

func finishDirectoryCompare(seq uint64, localBase, remoteBase string, entries []api.DirectoryComparisonEntry) {
	directoryCompareState.mu.Lock()
	defer directoryCompareState.mu.Unlock()
	if directoryCompareState.seq != seq {
		return
	}
	directoryCompareState.entries = append([]api.DirectoryComparisonEntry(nil), entries...)
	directoryCompareState.localBase = localBase
	directoryCompareState.remoteBase = remoteBase
	cancelDirectoryCompare = nil
}

func setDirectoryCompareError(err error, fallback string) {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	setBridgeError(err, fallback)
}

//export GhostFTPCompareDirectories
func GhostFTPCompareDirectories(localBaseValue, remoteBaseValue *C.char) C.int {
	engine, localBase, remoteBase, err := requireDirectoryCompareSnapshot(goString(localBaseValue), goString(remoteBaseValue))
	if err != nil {
		setDirectoryCompareError(err, "Ghost FTP could not start directory comparison.")
		return 0
	}
	ctx, seq := beginDirectoryCompare()
	defer func() {
		directoryCompareState.mu.Lock()
		if directoryCompareState.seq == seq && cancelDirectoryCompare != nil {
			cancelDirectoryCompare()
			cancelDirectoryCompare = nil
		}
		directoryCompareState.mu.Unlock()
	}()

	localResolved, localItems, err := engine.LocalList(ctx, localBase)
	if err == nil {
		var remoteItems []apiItemAlias
		_ = remoteItems
	}
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
	finishDirectoryCompare(seq, localResolved, remoteBase, entries)
	setDirectoryCompareError(nil, "")
	return 1
}

// apiItemAlias exists only to keep the compiler from permitting accidental
// direct filesystem/network traversal additions in the guarded block above.
type apiItemAlias = struct{}

//export GhostFTPCancelDirectoryCompare
func GhostFTPCancelDirectoryCompare() {
	directoryCompareState.mu.Lock()
	defer directoryCompareState.mu.Unlock()
	if cancelDirectoryCompare != nil {
		cancelDirectoryCompare()
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
	if !ok { return C.CString("") }
	return C.CString(entry.Name)
}

//export GhostFTPDirectoryCompareStatus
func GhostFTPDirectoryCompareStatus(index C.int) *C.char {
	entry, ok := directoryCompareEntryAt(index)
	if !ok { return C.CString("") }
	return C.CString(string(entry.Status))
}

//export GhostFTPDirectoryCompareHasLocal
func GhostFTPDirectoryCompareHasLocal(index C.int) C.int {
	entry, ok := directoryCompareEntryAt(index)
	if ok && entry.HasLocal { return 1 }
	return 0
}

//export GhostFTPDirectoryCompareHasRemote
func GhostFTPDirectoryCompareHasRemote(index C.int) C.int {
	entry, ok := directoryCompareEntryAt(index)
	if ok && entry.HasRemote { return 1 }
	return 0
}

//export GhostFTPDirectoryCompareLocalSize
func GhostFTPDirectoryCompareLocalSize(index C.int) C.longlong {
	entry, ok := directoryCompareEntryAt(index)
	if !ok || !entry.HasLocal { return 0 }
	return C.longlong(entry.Local.Size)
}

//export GhostFTPDirectoryCompareRemoteSize
func GhostFTPDirectoryCompareRemoteSize(index C.int) C.longlong {
	entry, ok := directoryCompareEntryAt(index)
	if !ok || !entry.HasRemote { return 0 }
	return C.longlong(entry.Remote.Size)
}

//export GhostFTPDirectoryCompareLocalModifiedUnix
func GhostFTPDirectoryCompareLocalModifiedUnix(index C.int) C.longlong {
	entry, ok := directoryCompareEntryAt(index)
	if !ok || !entry.HasLocal { return 0 }
	return itemModifiedUnix(entry.Local)
}

//export GhostFTPDirectoryCompareRemoteModifiedUnix
func GhostFTPDirectoryCompareRemoteModifiedUnix(index C.int) C.longlong {
	entry, ok := directoryCompareEntryAt(index)
	if !ok || !entry.HasRemote { return 0 }
	return itemModifiedUnix(entry.Remote)
}

//export GhostFTPDirectoryCompareCanOpenBoth
func GhostFTPDirectoryCompareCanOpenBoth(index C.int) C.int {
	entry, ok := directoryCompareEntryAt(index)
	if !ok { return 0 }
	bridgeState.mu.Lock()
	engine := bridgeState.engine
	bridgeState.mu.Unlock()
	if engine == nil { return 0 }
	directoryCompareState.mu.Lock()
	entries := append([]api.DirectoryComparisonEntry(nil), directoryCompareState.entries...)
	directoryCompareState.mu.Unlock()
	if _, ok := engine.SynchronizedDirectoryName(entries, entry.Name); ok { return 1 }
	return 0
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
	if cancelDirectoryCompare != nil {
		cancelDirectoryCompare()
	}
	cancelDirectoryCompare = nil
	directoryCompareState.seq++
	directoryCompareState.entries = nil
	directoryCompareState.localBase = ""
	directoryCompareState.remoteBase = ""
}
