package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"brendigo.com/byftp/internal/api"
	"brendigo.com/byftp/internal/clientmode"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
	"brendigo.com/byftp/internal/sshclient"
)

var version = "dev"

func main() {
	platform.HardenProcessPrivacy()
	mode := clientmode.SSH
	release, ok := platform.AcquireSingleInstance(mode.InstanceKey())
	if !ok {
		fmt.Fprintln(os.Stderr, mode.ProductName()+" je već pokrenut.")
		return
	}
	defer release()
	base, err := api.DataDir()
	if err != nil { fmt.Fprintln(os.Stderr, err); return }
	dataDir := filepath.Join(base, "clients", mode.Slug())
	localAppData, err := platform.LocalAppData()
	if err != nil || security.EnsureNoRedirectDirectory(localAppData, dataDir) != nil {
		fmt.Fprintln(os.Stderr, "SSH podatkovna mapa nije sigurna.")
		return
	}
	if err := sshclient.Run(context.Background(), dataDir, version); err != nil {
		fmt.Fprintln(os.Stderr, "SSH sesija je završila:", err)
	}
}
