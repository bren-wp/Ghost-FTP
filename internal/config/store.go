package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const maxStateSize = 4 << 20

type Store struct {
	dir string
	mu  sync.Mutex
}

func New(dir string) *Store  { return &Store{dir: dir} }
func (s *Store) Dir() string { return s.dir }

func validateStateName(name string) error {
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\\`) || name == "." || name == ".." {
		return errors.New("neispravan naziv datoteke postavki")
	}
	if len(name) > 128 {
		return errors.New("naziv datoteke postavki je predug")
	}
	return nil
}

func copyFallback(fallback, out any) error {
	b, err := json.Marshal(fallback)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func (s *Store) Read(name string, fallback any, out any) (string, error) {
	if err := validateStateName(name); err != nil {
		return "fallback", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return "fallback", err
	}

	path := filepath.Join(s.dir, name)
	if data, err := readLimited(path); err == nil {
		if err = json.Unmarshal(data, out); err == nil {
			return "current", nil
		}
	}
	if data, err := readLimited(path + ".previous"); err == nil {
		if err = json.Unmarshal(data, out); err == nil {
			return "previous", nil
		}
	}

	// A damaged/missing state file must not prevent the desktop app from booting.
	// Defaults are safe and the next successful write repairs the current state.
	if err := copyFallback(fallback, out); err != nil {
		return "fallback", err
	}
	return "fallback", nil
}

type stateOpenFunc func(string) (*os.File, error)

func readLimited(path string) ([]byte, error) {
	return readLimitedWithOpen(path, os.Open)
}

func sameStateSnapshot(before, after os.FileInfo) bool {
	if before == nil || after == nil || !before.Mode().IsRegular() || !after.Mode().IsRegular() {
		return false
	}
	if !os.SameFile(before, after) {
		return false
	}
	// SameFile is the primary identity check. Size and modification time are also
	// compared because some filesystems may recycle an identifier immediately
	// after delete/replace. Legitimate concurrent edits are rejected and retried
	// through the store's previous/default generation rather than read mid-write.
	return before.Size() == after.Size() && before.ModTime().Equal(after.ModTime())
}

// readLimitedWithOpen verifies that the object opened is the same stable regular
// file observed by Lstat. This closes path-swap windows where a local process
// replaces a validated state path immediately before or during the read.
func readLimitedWithOpen(path string, openFile stateOpenFunc) ([]byte, error) {
	lst, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !lst.Mode().IsRegular() || lst.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("state putanja nije obična datoteka")
	}
	if lst.Size() > maxStateSize {
		return nil, errors.New("state datoteka je prevelika")
	}
	f, err := openFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	opened, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !sameStateSnapshot(lst, opened) {
		return nil, errors.New("state datoteka se promijenila tijekom sigurnog otvaranja")
	}
	if opened.Size() > maxStateSize {
		return nil, errors.New("state datoteka je prevelika")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxStateSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxStateSize {
		return nil, errors.New("state datoteka je prevelika")
	}
	afterRead, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !sameStateSnapshot(opened, afterRead) || int64(len(data)) != afterRead.Size() {
		return nil, errors.New("state datoteka se promijenila tijekom čitanja")
	}
	return data, nil
}

func (s *Store) Write(name string, value any) error {
	if err := validateStateName(name); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > maxStateSize {
		return errors.New("state podatak je prevelik")
	}
	data = append(data, '\n')
	path := filepath.Join(s.dir, name)

	// Keep only a known-valid previous generation. Temporary files are created
	// with unpredictable names in the same directory and exclusive creation,
	// so stale/symlinked predictable .tmp paths cannot be followed.
	if existing, e := readLimited(path); e == nil && json.Valid(existing) {
		prevTmp, e := os.CreateTemp(s.dir, "."+name+".previous-*.tmp")
		if e != nil {
			return e
		}
		prevName := prevTmp.Name()
		if e = prevTmp.Chmod(0600); e == nil {
			_, e = prevTmp.Write(existing)
		}
		if e == nil {
			e = prevTmp.Sync()
		}
		closeErr := prevTmp.Close()
		if e == nil {
			e = closeErr
		}
		if e != nil {
			_ = os.Remove(prevName)
			return e
		}
		if e = replaceFile(prevName, path+".previous"); e != nil {
			_ = os.Remove(prevName)
			return e
		}
	}

	f, err := os.CreateTemp(s.dir, "."+name+"-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err = replaceFile(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
