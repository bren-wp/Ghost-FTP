//go:build !windows

package remote

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSharedHostingFTPQuoteUsesHomeRelativePathAndNoBody(t *testing.T) {
	dir := t.TempDir()
	prependTestToolDirectory(t, dir)
	writeExecutable(t, filepath.Join(dir, "curl"), `#!/bin/sh
cfg="$(cat)"
case "$cfg" in
  *'user = "account@example.com:shared-secret"'*) ;;
  *) echo 'nedostaje shared-hosting korisničko ime ili lozinka' >&2; exit 51 ;;
esac
case "$cfg" in
  *'no-body'*) ;;
  *) echo 'quote operacija nije control-only' >&2; exit 52 ;;
esac
case "$cfg" in
  *'quote = "MKD public_html/test-map"'*) ;;
  *) echo 'MKD nije login-home relativan' >&2; exit 53 ;;
esac
case "$cfg" in
  *'quote = "MKD /public_html/test-map"'*) echo 'MKD je pogrešno server-apsolutan' >&2; exit 54 ;;
esac
exit 0
`)

	client, err := newCurlFTPWithProtectedSecret("ftp", "ftp.example.test", 21, "account@example.com", "shared-secret", "", 15)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.Mkdir(context.Background(), "/public_html", "test-map"); err != nil {
		t.Fatalf("shared-hosting MKD nije prošao: %v", err)
	}
}

func TestSharedHostingFTPDisablesRepeatedMLSDWhenListFallbackWorks(t *testing.T) {
	dir := t.TempDir()
	prependTestToolDirectory(t, dir)
	calls := filepath.Join(dir, "calls.txt")
	t.Setenv("GhostFTP_TEST_CALLS", calls)
	writeExecutable(t, filepath.Join(dir, "curl"), `#!/bin/sh
cfg="$(cat)"
case "$cfg" in
  *'request = "MLSD"'*)
    printf 'M' >> "$GhostFTP_TEST_CALLS"
    printf '%s\n' 'legacy server response without MLSD facts'
    exit 0
    ;;
esac
printf 'L' >> "$GhostFTP_TEST_CALLS"
printf '%s\n' '-rw-r--r-- 1 user group 4 Aug 19 12:00 index.php'
`)

	client, err := newCurlFTPWithProtectedSecret("ftp", "ftp.example.test", 21, "account@example.com", "shared-secret", "", 15)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	for i := 0; i < 2; i++ {
		items, err := client.List(context.Background(), "/public_html")
		if err != nil {
			t.Fatalf("LIST fallback pokušaj %d: %v", i+1, err)
		}
		if len(items) != 1 || items[0].Name != "index.php" {
			t.Fatalf("neočekivan LIST fallback rezultat: %#v", items)
		}
	}
	got, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "MLL" {
		t.Fatalf("MLSD/LIST slijed=%q want %q", string(got), "MLL")
	}
}
