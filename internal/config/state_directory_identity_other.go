//go:build !windows

package config

import "os"

func sameStateDirectoryIdentity(before, after os.FileInfo) bool {
	return before != nil && after != nil && os.SameFile(before, after)
}
