package desktop

import (
	"sync"

	"brendigo.com/byftp/internal/clientmode"
)

var (
	modeMu sync.RWMutex
	mode   = clientmode.Suite
)

// SetClientMode se poziva jednom tijekom startupa prije stvaranja UI-a.
// Lock ostaje namjerno jer paket ima i testne/terminalne ulaze koji se mogu
// pokretati paralelno u istom test procesu.
func SetClientMode(next clientmode.Mode) {
	modeMu.Lock()
	mode = next
	modeMu.Unlock()
}

func clientMode() clientmode.Mode {
	modeMu.RLock()
	defer modeMu.RUnlock()
	return mode
}
