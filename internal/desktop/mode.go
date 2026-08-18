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

func clientProductName() string { return clientMode().ProductName() }

func clientSubtitle() string {
	switch clientMode() {
	case clientmode.FTP:
		return "FTP • FTPS  ·  Siguran prijenos datoteka  ·  Brendigo"
	case clientmode.SFTP:
		return "SFTP preko OpenSSH  ·  Host-key provjera  ·  Brendigo"
	default:
		return "FTP • FTPS • SFTP  ·  Siguran prijenos datoteka  ·  Brendigo"
	}
}

func clientShowsSFTPAuth() bool {
	m := clientMode()
	return m == clientmode.Suite || m == clientmode.SFTP
}

func clientHasFixedProtocol() bool { return clientMode() == clientmode.SFTP }
