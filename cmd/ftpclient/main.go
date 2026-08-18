package main

import (
	"brendigo.com/byftp/internal/appstart"
	"brendigo.com/byftp/internal/clientmode"
)

var version = "dev"

func main() {
	appstart.RunFileClient(clientmode.FTP, version)
}
