package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
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

// decodeState is transactional with respect to out. encoding/json may modify a
// destination before returning an error; decoding current and previous state
// directly into the same object can therefore create a hybrid generation that
// was never written to disk. Decode into a fresh value and publish it only after
// the complete JSON document has been accepted.
func decodeState(data []byte, out any) error {
	rv := reflect.ValueOf(out)
	if !rv.IsValid() || rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("odredište state podataka mora biti neprazan pokazivač")
	}
	tmp := reflect.New(rv.Elem().Type())
	if err := json.Unmarshal(data, tmp.Interface()); err != nil {
		return err
	}
	rv.Elem().Set(tmp.Elem())
	return nil
}

func copyFallback(fallback, out any) error {
	b, err := json.Marshal(fallback)
	if err != nil {
		return err
	}
	return decodeState(b, out)
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
		if err = decodeState(data, out); err == nil {
			return "current", nil
		}
	}
	if data, err := readLimited(path + ".previous"); err == nil {
		if err = decodeState(data, out); err == nil {
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

// writeSyncedTemp writes one complete generation into an unpredictable private
// temporary file in the final directory and syncs its contents before rename.
// The caller owns the returned path and must either replace it or remove it.
func writeSyncedTemp(dir, pattern string, data []byte) (string, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	name := f.Name()
	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(name)
	}
	if err = f.Chmod(0600); err != nil {
		cleanup()
		return "", err
	}
	if _, err = f.Write(data); err != nil {
		cleanup()
		return "", err
	}
	if err = f.Sync(); err != nil {
		cleanup()
		return "", err
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}

func replaceSyncedGeneration(dir, tmp, dst string) error {
	if err := replaceFile(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	// A successful rename changes directory metadata. On Unix this explicit
	// directory sync is needed to request durable visibility of the new entry;
	// Windows replaceFile already uses MOVEFILE_WRITE_THROUGH.
	return syncStateDirectory(dir)
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

	// Keep only a known-valid previous generation. The previous generation is
	// synced before the new current generation is activated, preserving a
	// recovery point if a crash happens between the two replacements.
	if existing, e := readLimited(path); e == nil && json.Valid(existing) {
		prevTmp, e := writeSyncedTemp(s.dir, "."+name+".previous-*.tmp", existing)
		if e != nil {
			return e
		}
		if e = replaceSyncedGeneration(s.dir, prevTmp, path+".previous"); e != nil {
			return e
		}
	}

	tmp, err := writeSyncedTemp(s.dir, "."+name+"-*.tmp", data)
	if err != nil {
		return err
	}
	return replaceSyncedGeneration(s.dir, tmp, path)
}
