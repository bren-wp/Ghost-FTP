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

func readLimited(path string) ([]byte, error) {
	lst, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !lst.Mode().IsRegular() || lst.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("state putanja nije obična datoteka")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, errors.New("state putanja nije obična datoteka")
	}
	if st.Size() > maxStateSize {
		return nil, errors.New("state datoteka je prevelika")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxStateSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxStateSize {
		return nil, errors.New("state datoteka je prevelika")
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
