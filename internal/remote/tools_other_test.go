//go:build !windows

package remote

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFindCurlPrefersNativeUnixBinary(t *testing.T) {
	dir := t.TempDir()
	native := filepath.Join(dir, "curl")
	windowsName := filepath.Join(dir, "curl.exe")
	for _, file := range []string{native, windowsName} {
		if err := os.WriteFile(file, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	oldSystemDirectory := systemDirectory
	calledSystemDirectory := false
	systemDirectory = func() (string, error) {
		calledSystemDirectory = true
		return "", errors.New("system directory must not be used on Unix")
	}
	defer func() { systemDirectory = oldSystemDirectory }()

	got, err := findCurl()
	if err != nil {
		t.Fatal(err)
	}
	if got != native {
		t.Fatalf("findCurl()=%q want native %q", got, native)
	}
	if calledSystemDirectory {
		t.Fatal("Unix findCurl unexpectedly queried the Windows system directory")
	}
}
