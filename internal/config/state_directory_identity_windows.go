//go:build windows

package config

import (
	"os"
	"syscall"
)

// primeStateDirectoryIdentity forces os.FileInfo's lazy Windows file identity
// to resolve while the validated path still names the object being bound.
//
// On Windows, os.SameFile loads VolumeSerialNumber/FileIndex lazily from the
// path saved inside FileInfo. If the path is renamed and replaced before the
// first SameFile call, an unprimed historical FileInfo can otherwise resolve
// to the replacement object. Comparing the FileInfo with itself here makes Go
// cache that immutable file ID at bind/revalidation time.
func primeStateDirectoryIdentity(info os.FileInfo) bool {
	return info != nil && os.SameFile(info, info)
}

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
