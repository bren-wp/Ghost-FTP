package api

import (
	"errors"

	"github.com/bren-wp/Ghost-FTP/internal/directorycompare"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

type DirectoryComparisonStatus = directorycompare.Status
type DirectoryComparisonEntry = directorycompare.Entry
type DirectoryComparisonOptions = directorycompare.Options

const (
	DirectorySame        = directorycompare.StatusSame
	DirectoryLocalOnly   = directorycompare.StatusLocalOnly
	DirectoryRemoteOnly  = directorycompare.StatusRemoteOnly
	DirectoryNewerLocal  = directorycompare.StatusNewerLocal
	DirectoryNewerRemote = directorycompare.StatusNewerRemote
	DirectoryConflict    = directorycompare.StatusConflict
	DirectoryUnknown     = directorycompare.StatusUnknown
)

// CompareDirectoryItems compares already-listed local and remote directory
// snapshots. It performs no filesystem/network I/O and cannot mutate either
// side. Freshness of the supplied snapshots remains the caller's responsibility.
func (e *Engine) CompareDirectoryItems(local, remote []model.Item, opts DirectoryComparisonOptions) ([]DirectoryComparisonEntry, error) {
	if e == nil {
		return nil, errors.New("directory comparison service is unavailable")
	}
	return directorycompare.Compare(local, remote, opts)
}

// SynchronizedDirectoryName returns an exact child name only when comparison
// proves the entry is an ordinary directory present on both sides.
func (e *Engine) SynchronizedDirectoryName(entries []DirectoryComparisonEntry, name string) (string, bool) {
	if e == nil {
		return "", false
	}
	return directorycompare.SynchronizedDirectoryName(entries, name)
}
