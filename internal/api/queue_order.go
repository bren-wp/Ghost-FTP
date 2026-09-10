package api

// MoveTransferTop raises one queued transfer to the first queued scheduler
// position. Running and terminal jobs keep their history slots.
func (e *Engine) MoveTransferTop(id string) error {
	return e.transfers.MoveQueuedTop(id)
}

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

// MoveTransferBottom lowers one queued transfer to the final queued scheduler
// position without changing running or terminal history slots.
func (e *Engine) MoveTransferBottom(id string) error {
	return e.transfers.MoveQueuedBottom(id)
}
