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

	"github.com/bren-wp/Ghost-FTP/internal/api"
)

var recursiveSearchState struct {
	mu      sync.Mutex
	seq     uint64
	results []api.SearchResult
	stats   api.SearchStats
}

var cancelRecursiveSearch context.CancelFunc

func requireLocalSearchSnapshot(base string) (*api.Engine, string, error) {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		return nil, "", errors.New("engine is not initialized")
	}
	base = strings.TrimSpace(base)
	if base == "" || bridgeState.localPath == "" {
		return nil, "", errors.New("local folder snapshot is unavailable")
	}
	if filepath.Clean(base) != filepath.Clean(bridgeState.localPath) {
		return nil, "", errors.New("local folder changed; refresh and try again")
	}
	return bridgeState.engine, bridgeState.localPath, nil
}

func requireRemoteSearchSnapshot(base string) (*api.Engine, string, error) {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		return nil, "", errors.New("engine is not initialized")
	}
	if _, ok := bridgeState.engine.ActiveConnection(); !ok {
		return nil, "", errors.New("not connected")
	}
	base = cleanRemotePath(base)
	if bridgeState.remotePath == "" || cleanRemotePath(bridgeState.remotePath) != base {
		return nil, "", errors.New("remote folder changed; refresh and try again")
	}
	return bridgeState.engine, cleanRemotePath(bridgeState.remotePath), nil
}

func beginRecursiveSearch() (context.Context, uint64) {
	recursiveSearchState.mu.Lock()
	defer recursiveSearchState.mu.Unlock()
	if cancelRecursiveSearch != nil {
		cancelRecursiveSearch()
	}
	recursiveSearchState.seq++
	seq := recursiveSearchState.seq
	recursiveSearchState.results = nil
	recursiveSearchState.stats = api.SearchStats{}
	ctx, cancel := context.WithCancel(context.Background())
	cancelRecursiveSearch = cancel
	return ctx, seq
}

func finishRecursiveSearch(seq uint64, stats api.SearchStats) {
	recursiveSearchState.mu.Lock()
	defer recursiveSearchState.mu.Unlock()
	if recursiveSearchState.seq != seq {
		return
	}
	recursiveSearchState.stats = stats
	cancelRecursiveSearch = nil
}

func appendRecursiveSearchResults(seq uint64, batch []api.SearchResult) error {
	recursiveSearchState.mu.Lock()
	defer recursiveSearchState.mu.Unlock()
	if recursiveSearchState.seq != seq {
		return context.Canceled
	}
	recursiveSearchState.results = append(recursiveSearchState.results, batch...)
	return nil
}

func setRecursiveSearchError(err error, fallback string) {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	setBridgeError(err, fallback)
}

//export GhostFTPSearchLocalRecursive
func GhostFTPSearchLocalRecursive(base, queryValue *C.char) C.int {
	engine, root, err := requireLocalSearchSnapshot(goString(base))
	if err != nil {
		setRecursiveSearchError(err, "Ghost FTP could not start the local recursive search.")
		return 0
	}
	query := strings.TrimSpace(goString(queryValue))
	ctx, seq := beginRecursiveSearch()
	stats, _, err := engine.SearchLocalRecursive(ctx, root, api.SearchOptions{Query: query}, func(batch []api.SearchResult) error {
		return appendRecursiveSearchResults(seq, batch)
	})
	finishRecursiveSearch(seq, stats)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return 2
		}
		setRecursiveSearchError(err, "Ghost FTP could not complete the local recursive search.")
		return 0
	}
	setRecursiveSearchError(nil, "")
	return 1
}

//export GhostFTPSearchRemoteRecursive
func GhostFTPSearchRemoteRecursive(base, queryValue *C.char) C.int {
	engine, root, err := requireRemoteSearchSnapshot(goString(base))
	if err != nil {
		setRecursiveSearchError(err, "Ghost FTP could not start the remote recursive search.")
		return 0
	}
	query := strings.TrimSpace(goString(queryValue))
	ctx, seq := beginRecursiveSearch()
	stats, err := engine.SearchRemoteRecursive(ctx, root, api.SearchOptions{Query: query}, func(batch []api.SearchResult) error {
		return appendRecursiveSearchResults(seq, batch)
	})
	finishRecursiveSearch(seq, stats)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			return 2
		}
		setRecursiveSearchError(err, "Ghost FTP could not complete the remote recursive search.")
		return 0
	}
	setRecursiveSearchError(nil, "")
	return 1
}

//export GhostFTPCancelRecursiveSearch
func GhostFTPCancelRecursiveSearch() {
	recursiveSearchState.mu.Lock()
	defer recursiveSearchState.mu.Unlock()
	if cancelRecursiveSearch != nil {
		cancelRecursiveSearch()
	}
}

func recursiveSearchResultAt(index C.int) (api.SearchResult, bool) {
	recursiveSearchState.mu.Lock()
	defer recursiveSearchState.mu.Unlock()
	i := int(index)
	if i < 0 || i >= len(recursiveSearchState.results) {
		return api.SearchResult{}, false
	}
	return recursiveSearchState.results[i], true
}

//export GhostFTPSearchResultCount
func GhostFTPSearchResultCount() C.int {
	recursiveSearchState.mu.Lock()
	defer recursiveSearchState.mu.Unlock()
	return C.int(len(recursiveSearchState.results))
}

//export GhostFTPSearchResultName
func GhostFTPSearchResultName(index C.int) *C.char {
	result, ok := recursiveSearchResultAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(result.Item.Name)
}

//export GhostFTPSearchResultParent
func GhostFTPSearchResultParent(index C.int) *C.char {
	result, ok := recursiveSearchResultAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(result.Parent)
}

//export GhostFTPSearchResultPath
func GhostFTPSearchResultPath(index C.int) *C.char {
	result, ok := recursiveSearchResultAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(result.Path)
}

//export GhostFTPSearchResultSize
func GhostFTPSearchResultSize(index C.int) C.longlong {
	result, ok := recursiveSearchResultAt(index)
	if !ok {
		return 0
	}
	return C.longlong(result.Item.Size)
}

//export GhostFTPSearchResultIsDirectory
func GhostFTPSearchResultIsDirectory(index C.int) C.int {
	result, ok := recursiveSearchResultAt(index)
	if ok && result.Item.IsDirectory {
		return 1
	}
	return 0
}

//export GhostFTPSearchResultIsSymlink
func GhostFTPSearchResultIsSymlink(index C.int) C.int {
	result, ok := recursiveSearchResultAt(index)
	if ok && result.Item.IsSymlink {
		return 1
	}
	return 0
}

//export GhostFTPSearchResultModifiedUnix
func GhostFTPSearchResultModifiedUnix(index C.int) C.longlong {
	result, ok := recursiveSearchResultAt(index)
	if !ok {
		return 0
	}
	return itemModifiedUnix(result.Item)
}

//export GhostFTPSearchVisited
func GhostFTPSearchVisited() C.int {
	recursiveSearchState.mu.Lock()
	defer recursiveSearchState.mu.Unlock()
	return C.int(recursiveSearchState.stats.Visited)
}

//export GhostFTPSearchStopReason
func GhostFTPSearchStopReason() *C.char {
	recursiveSearchState.mu.Lock()
	defer recursiveSearchState.mu.Unlock()
	return C.CString(recursiveSearchState.stats.StopReason)
}

//export GhostFTPClearRecursiveSearch
func GhostFTPClearRecursiveSearch() {
	recursiveSearchState.mu.Lock()
	defer recursiveSearchState.mu.Unlock()
	if cancelRecursiveSearch != nil {
		cancelRecursiveSearch()
	}
	cancelRecursiveSearch = nil
	recursiveSearchState.seq++
	recursiveSearchState.results = nil
	recursiveSearchState.stats = api.SearchStats{}
}
