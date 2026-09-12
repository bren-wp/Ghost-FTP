package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/itemlist"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

var macFileFilterState struct {
	localQuery    string
	remoteQuery   string
	localVisible  []model.Item
	remoteVisible []model.Item
}

func updateLocalFilterLocked(query string) {
	macFileFilterState.localQuery = strings.TrimSpace(query)
	macFileFilterState.localVisible = itemlist.Filter(bridgeState.localItems, macFileFilterState.localQuery)
}

func updateRemoteFilterLocked(query string) {
	macFileFilterState.remoteQuery = strings.TrimSpace(query)
	macFileFilterState.remoteVisible = itemlist.Filter(bridgeState.remoteItems, macFileFilterState.remoteQuery)
}

//export GhostFTPLocalFilter
func GhostFTPLocalFilter(query *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	updateLocalFilterLocked(goString(query))
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPRemoteFilter
func GhostFTPRemoteFilter(query *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	if _, ok := bridgeState.engine.ActiveConnection(); !ok {
		macFileFilterState.remoteVisible = nil
		setBridgeError(errors.New("not connected"), "Connect to a server before filtering remote files.")
		return 0
	}
	updateRemoteFilterLocked(goString(query))
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPLocalFilteredItemCount
func GhostFTPLocalFilteredItemCount() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	return C.int(len(macFileFilterState.localVisible))
}

//export GhostFTPLocalFilteredItemName
func GhostFTPLocalFilteredItemName(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(macFileFilterState.localVisible, index)
	if !ok {
		return C.CString("")
	}
	return C.CString(item.Name)
}

//export GhostFTPLocalFilteredItemSize
func GhostFTPLocalFilteredItemSize(index C.int) C.longlong {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(macFileFilterState.localVisible, index)
	if !ok {
		return 0
	}
	return C.longlong(item.Size)
}

//export GhostFTPLocalFilteredItemIsDirectory
func GhostFTPLocalFilteredItemIsDirectory(index C.int) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(macFileFilterState.localVisible, index)
	if ok && item.IsDirectory {
		return 1
	}
	return 0
}

//export GhostFTPLocalFilteredItemIsSymlink
func GhostFTPLocalFilteredItemIsSymlink(index C.int) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(macFileFilterState.localVisible, index)
	if ok && item.IsSymlink {
		return 1
	}
	return 0
}

//export GhostFTPLocalFilteredItemModifiedUnix
func GhostFTPLocalFilteredItemModifiedUnix(index C.int) C.longlong {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(macFileFilterState.localVisible, index)
	if !ok {
		return 0
	}
	return itemModifiedUnix(item)
}

//export GhostFTPRemoteFilteredItemCount
func GhostFTPRemoteFilteredItemCount() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	return C.int(len(macFileFilterState.remoteVisible))
}

//export GhostFTPRemoteFilteredItemName
func GhostFTPRemoteFilteredItemName(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(macFileFilterState.remoteVisible, index)
	if !ok {
		return C.CString("")
	}
	return C.CString(item.Name)
}

//export GhostFTPRemoteFilteredItemSize
func GhostFTPRemoteFilteredItemSize(index C.int) C.longlong {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(macFileFilterState.remoteVisible, index)
	if !ok {
		return 0
	}
	return C.longlong(item.Size)
}

//export GhostFTPRemoteFilteredItemIsDirectory
func GhostFTPRemoteFilteredItemIsDirectory(index C.int) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(macFileFilterState.remoteVisible, index)
	if ok && item.IsDirectory {
		return 1
	}
	return 0
}

//export GhostFTPRemoteFilteredItemIsSymlink
func GhostFTPRemoteFilteredItemIsSymlink(index C.int) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(macFileFilterState.remoteVisible, index)
	if ok && item.IsSymlink {
		return 1
	}
	return 0
}

//export GhostFTPRemoteFilteredItemModifiedUnix
func GhostFTPRemoteFilteredItemModifiedUnix(index C.int) C.longlong {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(macFileFilterState.remoteVisible, index)
	if !ok {
		return 0
	}
	return itemModifiedUnix(item)
}

//export GhostFTPRemoteFilteredItemPermissions
func GhostFTPRemoteFilteredItemPermissions(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := itemAt(macFileFilterState.remoteVisible, index)
	if !ok {
		return C.CString("")
	}
	return C.CString(item.Permissions)
}
