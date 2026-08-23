//go:build !windows

package remote

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bren-wp/by-ftp/internal/security"
)

func prependTestToolDirectory(t *testing.T, dir string) {
	t.Helper()
	oldSystemDirectory := systemDirectory
	systemDirectory = func() (string, error) { return "", errors.New("nema Windows system direktorija") }
	t.Cleanup(func() { systemDirectory = oldSystemDirectory })
	oldPath := os.Getenv("PATH")
	if oldPath == "" {
		t.Setenv("PATH", dir)
		return
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath)
}

func writeExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestCurlFTPProcessSmokeUsesRuntimeSecretAndParsesListing(t *testing.T) {
	dir := t.TempDir()
	prependTestToolDirectory(t, dir)
	writeExecutable(t, filepath.Join(dir, "curl"), `#!/bin/sh
cfg="$(cat)"
case "$cfg" in
  *'user = "tester:sesija-secret"'*) ;;
  *) echo 'nedostaje očekivana vjerodajnica' >&2; exit 41 ;;
esac
printf '%s\n' 'type=file;size=4;modify=20260101010203; test.txt'
`)

	client, err := newCurlFTPWithProtectedSecret("ftp", "example.test", 21, "tester", "sesija-secret", "", 15)
	if err != nil {
		t.Fatal(err)
	}
	if client.passwordBlob == "" || client.passwordBlob == "sesija-secret" {
		t.Fatalf("aktivna FTP tajna nije izdvojena iza runtime tokena: %q", client.passwordBlob)
	}
	token := client.passwordBlob
	items, err := client.List(context.Background(), "/")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "test.txt" || items[0].Size != 4 {
		t.Fatalf("neočekivan FTP smoke listing: %#v", items)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := security.UnprotectRuntimeBytes(token); err == nil {
		t.Fatal("FTP runtime tajna ostala je dostupna nakon Close()")
	}
}

func TestSFTPProcessSmokeUsesStdinWithoutBatchMode(t *testing.T) {
	dir := t.TempDir()
	prependTestToolDirectory(t, dir)
	writeExecutable(t, filepath.Join(dir, "sftp"), `#!/bin/sh
for arg in "$@"; do
  if [ "$arg" = "-b" ]; then
    echo 'OpenSSH batch način ne smije biti uključen' >&2
    exit 42
  fi
done
input="$(cat)"
case "$input" in
  *'ls -la "."'*) ;;
  *) echo 'nedostaje ls naredba na stdin-u' >&2; exit 43 ;;
esac
printf '%s\n' '-rw-r--r-- 1 user group 4 Jan 1 00:00 test.txt'
`)

	knownHosts := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(knownHosts, []byte("example.test ssh-ed25519 AAAATEST\n"), 0600); err != nil {
		t.Fatal(err)
	}
	privateKey := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(privateKey, []byte("test-private-key"), 0600); err != nil {
		t.Fatal(err)
	}
	client, err := newSFTPWithProtectedSecrets(
		"example.test", 22, "tester", "", "", privateKey, "", "",
		knownHosts, "ssh-ed25519", "", 15,
	)
	if err != nil {
		t.Fatal(err)
	}
	items, err := client.List(context.Background(), ".")
	if err != nil {
		_ = client.Close()
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "test.txt" || items[0].Size != 4 {
		_ = client.Close()
		t.Fatalf("neočekivan SFTP smoke listing: %#v", items)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
}
