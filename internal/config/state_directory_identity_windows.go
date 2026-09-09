//go:build windows

package config

import (
	"os"
	"syscall"
)

func sameStateDirectoryIdentity(before, after os.FileInfo) bool {
	if before == nil || after == nil || !os.SameFile(before, after) {
		return false
	}
	beforeData, beforeOK := before.Sys().(*syscall.Win32FileAttributeData)
	afterData, afterOK := after.Sys().(*syscall.Win32FileAttributeData)
	if !beforeOK || !afterOK || beforeData == nil || afterData == nil {
		return false
	}
	return beforeData.CreationTime == afterData.CreationTime
}
