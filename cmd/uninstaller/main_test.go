//go:build windows

package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSameWindowsPath(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{
			name: "case insensitive",
			a:    `C:\Users\A\AppData\Local\Programs\ByFTP\ByFTP\Uninstall.exe`,
			b:    `c:\users\a\appdata\local\programs\ByFTP\byftp\Uninstall.exe`,
			want: true,
		},
		{
			name: "normalizes extended drive prefix",
			a:    `\\?\C:\Users\A\ByFTP\Uninstall.exe`,
			b:    `C:\Users\A\ByFTP\Uninstall.exe`,
			want: true,
		},
		{
			name: "normalizes extended UNC prefix",
			a:    `\\?\UNC\server\share\ByFTP\Uninstall.exe`,
			b:    `\\server\share\ByFTP\Uninstall.exe`,
			want: true,
		},
		{
			name: "cleans dot segments",
			a:    `C:\Users\A\ByFTP\bin\..\Uninstall.exe`,
			b:    `C:\Users\A\ByFTP\Uninstall.exe`,
			want: true,
		},
		{
			name: "different filename",
			a:    `C:\Users\A\ByFTP\Uninstall.exe`,
			b:    `C:\Users\A\ByFTP\ByFTP.exe`,
			want: false,
		},
		{
			name: "different directory",
			a:    `C:\Users\A\ByFTP\Uninstall.exe`,
			b:    `C:\Users\A\Other\Uninstall.exe`,
			want: false,
		},
		{
			name: "empty first path",
			a:    "",
			b:    `C:\Users\A\ByFTP\Uninstall.exe`,
			want: false,
		},
		{
			name: "empty second path",
			a:    `C:\Users\A\ByFTP\Uninstall.exe`,
			b:    "",
			want: false,
		},
		{
			name: "both empty",
			a:    "",
			b:    "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sameWindowsPath(tt.a, tt.b)
			if got != tt.want {
				t.Fatalf(
					"sameWindowsPath(%q, %q) = %v, want %v",
					tt.a,
					tt.b,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestRemoveUserDataRejectsEmptyPath(t *testing.T) {
	for _, path := range []string{"", " ", "\t", "\r\n"} {
		t.Run("path_"+filepath.Base(path), func(t *testing.T) {
			if err := removeUserData(path); err == nil {
				t.Fatalf("removeUserData(%q) unexpectedly succeeded", path)
			}
		})
	}
}

func TestRemoveUserDataRemovesRegularTree(t *testing.T) {
	root := t.TempDir()
	data := filepath.Join(root, "ByFTP")

	if err := os.MkdirAll(filepath.Join(data, "profiles", "nested"), 0700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(data, "profiles", "nested", "profile.json"),
		[]byte(`{"name":"test"}`),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	if err := removeUserData(data); err != nil {
		t.Fatalf("removeUserData() failed: %v", err)
	}

	_, err := os.Lstat(data)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("user data directory still exists or stat failed unexpectedly: %v", err)
	}
}

func TestRemoveUserDataDoesNotFollowNestedSymlink(t *testing.T) {
	root := t.TempDir()
	data := filepath.Join(root, "ByFTP")
	outside := filepath.Join(root, "outside")

	if err := os.MkdirAll(data, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0700); err != nil {
		t.Fatal(err)
	}

	keep := filepath.Join(outside, "keep.txt")
	const expected = "keep"

	if err := os.WriteFile(keep, []byte(expected), 0600); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(data, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink creation unavailable on this Windows configuration: %v", err)
	}

	if err := removeUserData(data); err != nil {
		t.Fatalf("removeUserData() failed: %v", err)
	}

	content, err := os.ReadFile(keep)
	if err != nil {
		t.Fatalf("outside file was removed or became inaccessible: %v", err)
	}
	if string(content) != expected {
		t.Fatalf("outside file was modified: got %q, want %q", content, expected)
	}

	if _, err := os.Lstat(outside); err != nil {
		t.Fatalf("outside directory was modified or removed: %v", err)
	}

	if _, err := os.Lstat(data); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("user data directory was not removed: %v", err)
	}
}
