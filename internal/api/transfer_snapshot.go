package api

import "github.com/bren-wp/Ghost-FTP/internal/model"

// Transfers vraća izolirani snapshot trenutnog reda prijenosa. Pozivatelj ne
// dobiva interne sliceove transfer managera i zato ne može mutirati njegovo
// stanje. Snapshot je autoritativan za terminalne statuse čak i kada je stariji
// event već izbačen iz ograničene povijesti događaja.
func (e *Engine) Transfers() []model.TransferJob {
	return e.transfers.List()
}

// TransferQueueSnapshot returns one isolated queue snapshot together with the
// transfer manager's authoritative paused state. UI surfaces therefore do not
// need to invent or mirror scheduler state independently.
func (e *Engine) TransferQueueSnapshot() ([]model.TransferJob, bool) {
	return e.transfers.Snapshot()
}
