package api

// MoveTransferUp raises one queued transfer by one queued position. The
// transfer manager rejects running, completed and unknown jobs.
func (e *Engine) MoveTransferUp(id string) error {
	return e.transfers.MoveQueuedUp(id)
}

// MoveTransferDown lowers one queued transfer by one queued position. Running
// and terminal jobs keep their scheduler/history position.
func (e *Engine) MoveTransferDown(id string) error {
	return e.transfers.MoveQueuedDown(id)
}
