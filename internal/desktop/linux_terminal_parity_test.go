//go:build linux

package desktop

import (
	"bufio"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestTerminalRemotePathUsesCurrentRemoteDirectory(t *testing.T) {
	tests := []struct {
		current string
		value   string
		want    string
	}{
		{"/public_html", "assets/app.css", "/public_html/assets/app.css"},
		{"/public_html", "/logs/error.log", "/logs/error.log"},
		{".", "site/index.html", "site/index.html"},
		{".", ".", "."},
		{"site", "assets", "site/assets"},
	}
	for _, tc := range tests {
		if got := terminalRemotePath(tc.current, tc.value); got != tc.want {
			t.Fatalf("terminalRemotePath(%q, %q) = %q, want %q", tc.current, tc.value, got, tc.want)
		}
	}
}

func TestTerminalLocalPathUsesLocalPanelDirectory(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "home", "ghost", "work")
	if got := terminalLocalPath(root, "site/index.html"); got != filepath.Join(root, "site", "index.html") {
		t.Fatalf("relative local path = %q", got)
	}
	absolute := filepath.Join(string(filepath.Separator), "tmp", "target.txt")
	if got := terminalLocalPath(root, absolute); got != filepath.Clean(absolute) {
		t.Fatalf("absolute local path = %q, want %q", got, filepath.Clean(absolute))
	}
}

func TestConfirmTerminalDeleteHonorsDisabledSetting(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))
	if !confirmTerminalDelete(reader, "en", model.Settings{ConfirmDelete: false}, "/tmp/file") {
		t.Fatal("disabled confirmation must permit the typed delete operation")
	}
}

func TestConfirmTerminalDeleteIsFailClosed(t *testing.T) {
	for _, input := range []string{"\n", "no\n", "unexpected\n"} {
		reader := bufio.NewReader(strings.NewReader(input))
		if confirmTerminalDelete(reader, "en", model.Settings{ConfirmDelete: true}, "/tmp/file") {
			t.Fatalf("delete confirmation unexpectedly accepted %q", input)
		}
	}
	reader := bufio.NewReader(strings.NewReader("yes\n"))
	if !confirmTerminalDelete(reader, "en", model.Settings{ConfirmDelete: true}, "/tmp/file") {
		t.Fatal("explicit affirmative confirmation should be accepted")
	}
}
