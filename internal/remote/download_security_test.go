package remote

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDownloadedPartAcceptsRegularFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "part")
	if err := os.WriteFile(p, []byte("ok"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateDownloadedPart(p); err != nil {
		t.Fatalf("regular part rejected: %v", err)
	}
}

func TestValidateDownloadedPartRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "part")
	if err := os.WriteFile(target, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink nije dostupan: %v", err)
	}
	if err := validateDownloadedPart(link); err == nil {
		t.Fatal("symlink je prihvaćen kao preuzeta privremena datoteka")
	}
}
