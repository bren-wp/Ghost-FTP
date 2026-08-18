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
// Mode određuje dostupne protokole i korisnički identitet odvojenog klijenta.
func SetClientMode(next clientmode.Mode) {
	modeMu.Lock()
	mode = next
	modeMu.Unlock()
	configureProtocolMode(next)
}

func clientMode() clientmode.Mode {
	modeMu.RLock()
	defer modeMu.RUnlock()
	return mode
}
