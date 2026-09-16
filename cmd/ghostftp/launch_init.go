package main

import (
	"os"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/desktop"
)

// configureProtocolLaunch runs before the normal desktop startup path so it
// cannot interfere with AskPass handling. Invalid or credential-bearing launch
// URIs are ignored and the application continues with its normal empty state.
func init() {
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(arg)), "ghostftp:") {
			continue
		}
		_ = desktop.ConfigureInitialLaunchTarget(strings.TrimSpace(arg))
		return
	}
}
