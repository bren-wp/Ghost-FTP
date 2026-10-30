//go:build !windows

package main

// Headless Linux builds never launch a browser or pretend to have a native
// graphical control center. Administrators use systemd or a terminal and must
// explicitly configure credentials and a TLS identity.
func runNativeControlCenter() bool { return false }
