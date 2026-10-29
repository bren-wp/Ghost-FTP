//go:build !windows

package main

import (
 "errors"
 server "ghostftp/server"
)

func runAsWindowsService(_ *server.Server,_ string)error {
 return errors.New("-service requires Windows Service Control Manager")
}
