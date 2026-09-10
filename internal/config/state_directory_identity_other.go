//go:build !windows

package config

import "os"

func primeStateDirectoryIdentity(info os.FileInfo) bool {
	return info != nil
}

func sameStateDirectoryIdentity(before, after os.FileInfo) bool {
	return before != nil && after != nil && os.SameFile(before, after)
}
