package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"strings"
	"sync"
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

const maxQueueSelection = 1000

var transferQueueState struct {
	mu     sync.RWMutex
	jobs   []model.TransferJob
	paused bool
}

func transferQueueEngine() (*api.Engine, error) {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		return nil, errors.New("engine is not initialized")
	}
	return bridgeState.engine, nil
}

func setTransferQueueError(err error, fallback string) {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	setBridgeError(err, fallback)
}

func refreshTransferQueueSnapshot(engine *api.Engine) {
	jobs, paused := engine.TransferQueueSnapshot()
	transferQueueState.mu.Lock()
	transferQueueState.jobs = append(transferQueueState.jobs[:0], jobs...)
	transferQueueState.paused = paused
	transferQueueState.mu.Unlock()
}

//export GhostFTPRefreshTransferQueue
func GhostFTPRefreshTransferQueue() C.int {
	engine, err := transferQueueEngine()
	if err != nil {
		setTransferQueueError(err, "Ghost FTP could not read the transfer queue.")
		return 0
	}
	refreshTransferQueueSnapshot(engine)
	setTransferQueueError(nil, "")
	return 1
}

//export GhostFTPTransferQueuePaused
func GhostFTPTransferQueuePaused() C.int {
	transferQueueState.mu.RLock()
	defer transferQueueState.mu.RUnlock()
	if transferQueueState.paused {
		return 1
	}
	return 0
}

//export GhostFTPTransferCount
func GhostFTPTransferCount() C.int {
	transferQueueState.mu.RLock()
	defer transferQueueState.mu.RUnlock()
	return C.int(len(transferQueueState.jobs))
}

func transferJobAt(index C.int) (model.TransferJob, bool) {
	transferQueueState.mu.RLock()
	defer transferQueueState.mu.RUnlock()
	i := int(index)
	if i < 0 || i >= len(transferQueueState.jobs) {
		return model.TransferJob{}, false
	}
	return transferQueueState.jobs[i], true
}

//export GhostFTPTransferID
func GhostFTPTransferID(index C.int) *C.char {
	job, ok := transferJobAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(job.ID)
}

//export GhostFTPTransferDirection
func GhostFTPTransferDirection(index C.int) *C.char {
	job, ok := transferJobAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(job.Direction)
}

//export GhostFTPTransferLocalPath
func GhostFTPTransferLocalPath(index C.int) *C.char {
	job, ok := transferJobAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(job.LocalPath)
}

//export GhostFTPTransferRemotePath
func GhostFTPTransferRemotePath(index C.int) *C.char {
	job, ok := transferJobAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(job.RemotePath)
}

//export GhostFTPTransferStatus
func GhostFTPTransferStatus(index C.int) *C.char {
	job, ok := transferJobAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(job.Status)
}

//export GhostFTPTransferProgress
func GhostFTPTransferProgress(index C.int) C.double {
	job, ok := transferJobAt(index)
	if !ok {
		return 0
	}
	return C.double(job.Progress)
}

//export GhostFTPTransferBytesTransferred
func GhostFTPTransferBytesTransferred(index C.int) C.longlong {
	job, ok := transferJobAt(index)
	if !ok {
		return 0
	}
	return C.longlong(job.BytesTransferred)
}

//export GhostFTPTransferBytesTotal
func GhostFTPTransferBytesTotal(index C.int) C.longlong {
	job, ok := transferJobAt(index)
	if !ok {
		return 0
	}
	return C.longlong(job.BytesTotal)
}

//export GhostFTPTransferBytesPerSecond
func GhostFTPTransferBytesPerSecond(index C.int) C.double {
	job, ok := transferJobAt(index)
	if !ok {
		return 0
	}
	return C.double(job.BytesPerSecond)
}

//export GhostFTPTransferETASeconds
func GhostFTPTransferETASeconds(index C.int) C.longlong {
	job, ok := transferJobAt(index)
	if !ok {
		return 0
	}
	return C.longlong(job.ETASeconds)
}

//export GhostFTPTransferAttempts
func GhostFTPTransferAttempts(index C.int) C.int {
	job, ok := transferJobAt(index)
	if !ok {
		return 0
	}
	return C.int(job.Attempts)
}

//export GhostFTPTransferError
func GhostFTPTransferError(index C.int) *C.char {
	job, ok := transferJobAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(job.Error)
}

func selectedTransferIDs(rows *C.int, count C.int, allowed map[string]bool) ([]string, error) {
	n := int(count)
	if n <= 0 || n > maxQueueSelection || rows == nil {
		return nil, errors.New("invalid transfer selection")
	}
	indices := unsafe.Slice(rows, n)
	transferQueueState.mu.RLock()
	defer transferQueueState.mu.RUnlock()
	ids := make([]string, 0, n)
	seen := make(map[string]struct{}, n)
	for _, raw := range indices {
		i := int(raw)
		if i < 0 || i >= len(transferQueueState.jobs) {
			return nil, errors.New("transfer queue changed; refresh and try again")
		}
		job := transferQueueState.jobs[i]
		if !allowed[job.Status] {
			return nil, errors.New("selected transfer is not valid for this action")
		}
		id := strings.TrimSpace(job.ID)
		if id == "" {
			return nil, errors.New("selected transfer has no stable identifier")
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, errors.New("no transfer is selected")
	}
	return ids, nil
}

func finishTransferQueueAction(engine *api.Engine, err error, fallback string) C.int {
	if err != nil {
		refreshTransferQueueSnapshot(engine)
		setTransferQueueError(err, fallback)
		return 0
	}
	refreshTransferQueueSnapshot(engine)
	setTransferQueueError(nil, "")
	return 1
}

//export GhostFTPPauseTransferQueue
func GhostFTPPauseTransferQueue() C.int {
	engine, err := transferQueueEngine()
	if err != nil {
		setTransferQueueError(err, "Ghost FTP could not pause the transfer queue.")
		return 0
	}
	engine.PauseTransfers()
	return finishTransferQueueAction(engine, nil, "")
}

//export GhostFTPResumeTransferQueue
func GhostFTPResumeTransferQueue() C.int {
	engine, err := transferQueueEngine()
	if err != nil {
		setTransferQueueError(err, "Ghost FTP could not resume the transfer queue.")
		return 0
	}
	engine.ResumeTransfers()
	return finishTransferQueueAction(engine, nil, "")
}

//export GhostFTPCancelTransferRows
func GhostFTPCancelTransferRows(rows *C.int, count C.int) C.int {
	ids, err := selectedTransferIDs(rows, count, map[string]bool{"queued": true, "running": true})
	if err != nil {
		setTransferQueueError(err, "Ghost FTP could not cancel the selected transfers.")
		return 0
	}
	engine, err := transferQueueEngine()
	if err != nil {
		setTransferQueueError(err, "Ghost FTP could not cancel the selected transfers.")
		return 0
	}
	return finishTransferQueueAction(engine, engine.CancelTransfers(ids), "Ghost FTP could not cancel the selected transfers.")
}

//export GhostFTPRetryTransferRows
func GhostFTPRetryTransferRows(rows *C.int, count C.int) C.int {
	ids, err := selectedTransferIDs(rows, count, map[string]bool{"failed": true, "cancelled": true})
	if err != nil {
		setTransferQueueError(err, "Ghost FTP could not retry the selected transfers.")
		return 0
	}
	engine, err := transferQueueEngine()
	if err != nil {
		setTransferQueueError(err, "Ghost FTP could not retry the selected transfers.")
		return 0
	}
	return finishTransferQueueAction(engine, engine.RetryTransfers(ids), "Ghost FTP could not retry the selected transfers.")
}

//export GhostFTPClearFinishedTransfers
func GhostFTPClearFinishedTransfers() C.int {
	engine, err := transferQueueEngine()
	if err != nil {
		setTransferQueueError(err, "Ghost FTP could not clear completed transfers.")
		return 0
	}
	engine.ClearFinishedTransfers()
	return finishTransferQueueAction(engine, nil, "")
}

type transferMove func(*api.Engine, string) error

func moveTransferRow(index C.int, move transferMove, fallback string) C.int {
	job, ok := transferJobAt(index)
	if !ok || job.Status != "queued" || strings.TrimSpace(job.ID) == "" {
		setTransferQueueError(errors.New("selected transfer is no longer queued"), fallback)
		return 0
	}
	engine, err := transferQueueEngine()
	if err != nil {
		setTransferQueueError(err, fallback)
		return 0
	}
	return finishTransferQueueAction(engine, move(engine, job.ID), fallback)
}

//export GhostFTPMoveTransferTop
func GhostFTPMoveTransferTop(index C.int) C.int {
	return moveTransferRow(index, func(engine *api.Engine, id string) error { return engine.MoveTransferTop(id) }, "Ghost FTP could not move the transfer to the top.")
}

//export GhostFTPMoveTransferUp
func GhostFTPMoveTransferUp(index C.int) C.int {
	return moveTransferRow(index, func(engine *api.Engine, id string) error { return engine.MoveTransferUp(id) }, "Ghost FTP could not move the transfer up.")
}

//export GhostFTPMoveTransferDown
func GhostFTPMoveTransferDown(index C.int) C.int {
	return moveTransferRow(index, func(engine *api.Engine, id string) error { return engine.MoveTransferDown(id) }, "Ghost FTP could not move the transfer down.")
}

//export GhostFTPMoveTransferBottom
func GhostFTPMoveTransferBottom(index C.int) C.int {
	return moveTransferRow(index, func(engine *api.Engine, id string) error { return engine.MoveTransferBottom(id) }, "Ghost FTP could not move the transfer to the bottom.")
}

//export GhostFTPClearTransferQueueSnapshot
func GhostFTPClearTransferQueueSnapshot() {
	transferQueueState.mu.Lock()
	transferQueueState.jobs = nil
	transferQueueState.paused = false
	transferQueueState.mu.Unlock()
}
