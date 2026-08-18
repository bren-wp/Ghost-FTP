//go:build !windows

package config

import (
	"errors"
	"os"
)

func replaceFile(src, dst string) error { return os.Rename(src, dst) }

// syncStateDirectory makes the directory-entry update from rename visible to
// the filesystem's durability boundary. The temporary file itself is synced
// before rename; syncing the containing directory afterwards closes the Linux
// metadata durability gap and requests the corresponding best-effort fsync on
// other supported Unix platforms.
func syncStateDirectory(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if !st.IsDir() {
		return errors.New("state putanja nije direktorij")
	}
	return f.Sync()
}
