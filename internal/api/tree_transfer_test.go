package api

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/remote"
)

type fakeTreeSession struct {
	mu        sync.Mutex
	dirs      map[string]bool
	listCalls map[string]int
}

func newFakeTreeSession(dirs ...string) *fakeTreeSession {
	m := map[string]bool{"/": true}
	for _, d := range dirs {
		m[path.Clean(d)] = true
	}
	return &fakeTreeSession{dirs: m, listCalls: make(map[string]int)}
}

func (f *fakeTreeSession) Protocol() string { return "sftp" }
func (f *fakeTreeSession) Host() string     { return "example.test" }
func (f *fakeTreeSession) Port() int        { return 22 }
func (f *fakeTreeSession) List(_ context.Context, p string) ([]model.Item, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p = path.Clean(p)
	f.listCalls[p]++
	if !f.dirs[p] {
		return nil, errors.New("not found")
	}
	return nil, nil
}
func (f *fakeTreeSession) Mkdir(_ context.Context, base, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	base = path.Clean(base)
	if !f.dirs[base] {
		return errors.New("parent missing")
	}
	f.dirs[path.Join(base, name)] = true
	return nil
}
func (f *fakeTreeSession) Rename(context.Context, string, string, string) error { return nil }
func (f *fakeTreeSession) Delete(context.Context, string, string, bool) error   { return nil }
func (f *fakeTreeSession) Chmod(context.Context, string, string, string) error  { return nil }
func (f *fakeTreeSession) Upload(context.Context, string, string, remote.TransferOptions) error {
	return nil
}
func (f *fakeTreeSession) Download(context.Context, string, string, remote.TransferOptions) error {
	return nil
}
func (f *fakeTreeSession) Close() error { return nil }

func TestEnsureRemoteDirectoryCreatesParents(t *testing.T) {
	f := newFakeTreeSession()
	if err := ensureRemoteDirectory(context.Background(), f, "/one/two/three"); err != nil {
		t.Fatalf("ensureRemoteDirectory: %v", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	got := make([]string, 0, len(f.dirs))
	for d := range f.dirs {
		got = append(got, d)
	}
	sort.Strings(got)
	for _, want := range []string{"/", "/one", "/one/two", "/one/two/three"} {
		if !f.dirs[want] {
			t.Fatalf("missing created directory %q; got %v", want, got)
		}
	}
}

func TestEnsureRemoteDirectoryExisting(t *testing.T) {
	f := newFakeTreeSession("/already")
	if err := ensureRemoteDirectory(context.Background(), f, "/already"); err != nil {
		t.Fatalf("existing directory should succeed: %v", err)
	}
}

func TestEnsureRemoteDirectoryRelativeFromHome(t *testing.T) {
	f := newFakeTreeSession(".")
	if err := ensureRemoteDirectory(context.Background(), f, "public_html/site"); err != nil {
		t.Fatalf("ensure relative remote directory: %v", err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, want := range []string{".", "public_html", "public_html/site"} {
		if !f.dirs[path.Clean(want)] {
			t.Fatalf("missing relative created directory %q; got %#v", want, f.dirs)
		}
	}
}

func TestPrepareLocalDirectoriesRollbackRemovesOnlyCreatedEmptyDirs(t *testing.T) {
	base := t.TempDir()
	preexisting := filepath.Join(base, "existing")
	if err := os.Mkdir(preexisting, 0755); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "download")
	nested := filepath.Join(root, "nested")
	cleanup, err := prepareLocalDirectories(base, []string{root, nested, preexisting})
	if err != nil {
		t.Fatalf("prepareLocalDirectories: %v", err)
	}
	if _, err := os.Stat(nested); err != nil {
		t.Fatalf("nested directory was not prepared: %v", err)
	}
	cleanup()
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("created directory should be removed on rollback, err=%v", err)
	}
	if st, err := os.Stat(preexisting); err != nil || !st.IsDir() {
		t.Fatalf("pre-existing directory must survive rollback, stat=%v err=%v", st, err)
	}
}

func TestPrepareLocalDirectoriesRollbackNeverDeletesNewContent(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "download")
	cleanup, err := prepareLocalDirectories(base, []string{root})
	if err != nil {
		t.Fatalf("prepareLocalDirectories: %v", err)
	}
	marker := filepath.Join(root, "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	cleanup()
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("rollback must not recursively delete content created later: %v", err)
	}
}

func TestEnsureRemoteDirectoryCacheAvoidsRelistingKnownParents(t *testing.T) {
	f := newFakeTreeSession()
	known := make(map[string]struct{})
	for _, target := range []string{"/one", "/one/two", "/one/two/three"} {
		if err := ensureRemoteDirectoryCached(context.Background(), f, target, known); err != nil {
			t.Fatal(err)
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	// Each missing directory needs one failing existence probe and one successful
	// verification after creation. Its already-known parent must not be probed again.
	for _, target := range []string{"/one", "/one/two", "/one/two/three"} {
		if got := f.listCalls[target]; got != 2 {
			t.Fatalf("List(%s) calls=%d want 2", target, got)
		}
	}
}
