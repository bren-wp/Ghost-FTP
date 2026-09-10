package directorycompare

// SynchronizedDirectoryName returns the exact child name only when comparison
// proves that both snapshots contain the same ordinary directory entry. The
// caller still owns platform/protocol-specific safe path resolution and fresh
// listing. This helper deliberately provides no transfer or mutation action.
func SynchronizedDirectoryName(entries []Entry, name string) (string, bool) {
	for _, entry := range entries {
		if entry.Name != name {
			continue
		}
		if entry.Status != StatusSame || !entry.HasLocal || !entry.HasRemote {
			return "", false
		}
		if !entry.Local.IsDirectory || !entry.Remote.IsDirectory {
			return "", false
		}
		if entry.Local.IsSymlink || entry.Remote.IsSymlink {
			return "", false
		}
		return entry.Name, true
	}
	return "", false
}
