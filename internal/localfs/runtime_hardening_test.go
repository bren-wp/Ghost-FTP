package localfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListContextAcceptsNilContext(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("ok"), 0600); err != nil {
		t.Fatal(err)
	}
	path, items, err := New().ListContext(nil, dir)
	if err != nil {
		t.Fatalf("nil-context local list failed: %v", err)
	}
	if path == "" || len(items) != 1 || items[0].Name != "index.html" {
		t.Fatalf("unexpected local list result: path=%q items=%#v", path, items)
	}
}
