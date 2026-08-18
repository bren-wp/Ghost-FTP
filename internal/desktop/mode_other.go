//go:build !windows

package desktop

import "brendigo.com/byftp/internal/clientmode"

func configureProtocolMode(clientmode.Mode) {}
