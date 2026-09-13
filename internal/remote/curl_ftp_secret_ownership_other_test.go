//go:build !windows

package remote

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/security"
)

func TestCurlFTPClosePreservesBorrowedRuntimeSecret(t *testing.T) {
	blob, err := security.ProtectRuntimeString("borrowed-runtime-password")
	if err != nil {
		t.Fatal(err)
	}
	defer security.ForgetRuntimeSecret(blob)

	session := &CurlFTP{passwordBlob: blob, ownsPasswordBlob: false}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	plain, err := security.UnprotectRuntimeBytes(blob)
	if err != nil {
		t.Fatalf("borrowed runtime secret was destroyed by CurlFTP.Close: %v", err)
	}
	security.WipeBytes(plain)
}

func TestCurlFTPCloseForgetsOwnedRuntimeSecret(t *testing.T) {
	blob, err := security.ProtectRuntimeString("owned-runtime-password")
	if err != nil {
		t.Fatal(err)
	}

	session := &CurlFTP{passwordBlob: blob, ownsPasswordBlob: true}
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if plain, err := security.UnprotectRuntimeBytes(blob); err == nil {
		security.WipeBytes(plain)
		t.Fatal("owned runtime secret remained available after CurlFTP.Close")
	}
}
