package directorycompare

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestSynchronizedDirectoryNameRequiresSafePairedDirectory(t *testing.T) {
	entry := Entry{
		Name:      "public_html",
		Status:    StatusSame,
		HasLocal:  true,
		HasRemote: true,
		Local:     model.Item{Name: "public_html", IsDirectory: true},
		Remote:    model.Item{Name: "public_html", IsDirectory: true},
	}
	if got, ok := SynchronizedDirectoryName([]Entry{entry}, "public_html"); !ok || got != "public_html" {
		t.Fatalf("got=%q ok=%v", got, ok)
	}
}

func TestSynchronizedDirectoryNameFailsClosedForAmbiguousStates(t *testing.T) {
	base := Entry{
		Name:      "folder",
		Status:    StatusSame,
		HasLocal:  true,
		HasRemote: true,
		Local:     model.Item{Name: "folder", IsDirectory: true},
		Remote:    model.Item{Name: "folder", IsDirectory: true},
	}
	tests := []struct {
		name string
		edit func(*Entry)
	}{
		{"missing local", func(e *Entry) { e.HasLocal = false }},
		{"missing remote", func(e *Entry) { e.HasRemote = false }},
		{"conflict", func(e *Entry) { e.Status = StatusConflict }},
		{"local file", func(e *Entry) { e.Local.IsDirectory = false }},
		{"remote file", func(e *Entry) { e.Remote.IsDirectory = false }},
		{"local symlink", func(e *Entry) { e.Local.IsSymlink = true }},
		{"remote symlink", func(e *Entry) { e.Remote.IsSymlink = true }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entry := base
			tc.edit(&entry)
			if got, ok := SynchronizedDirectoryName([]Entry{entry}, "folder"); ok || got != "" {
				t.Fatalf("got=%q ok=%v", got, ok)
			}
		})
	}
}

func TestSynchronizedDirectoryNameUsesExactName(t *testing.T) {
	entry := Entry{
		Name:      "Folder",
		Status:    StatusSame,
		HasLocal:  true,
		HasRemote: true,
		Local:     model.Item{Name: "Folder", IsDirectory: true},
		Remote:    model.Item{Name: "Folder", IsDirectory: true},
	}
	if got, ok := SynchronizedDirectoryName([]Entry{entry}, "folder"); ok || got != "" {
		t.Fatalf("case-folded synchronization unexpectedly accepted: %q %v", got, ok)
	}
}
