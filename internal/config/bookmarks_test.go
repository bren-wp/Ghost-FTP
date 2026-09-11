package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestBookmarksSaveListUpdateRemoveLocal(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	bookmarks := NewBookmarks(store)
	path := filepath.Join(dir, "projects", "..", "projects")

	saved, err := bookmarks.Save(model.BookmarkInput{
		Name: " Projects ", Kind: model.BookmarkKindLocal, Path: path,
	})
	if err != nil {
		t.Fatalf("Save local bookmark: %v", err)
	}
	if saved.Name != "Projects" || saved.Path != filepath.Clean(path) || saved.ID == "" {
		t.Fatalf("unexpected normalized bookmark: %#v", saved)
	}

	items, err := bookmarks.List()
	if err != nil || len(items) != 1 || items[0].ID != saved.ID {
		t.Fatalf("List = %#v, %v", items, err)
	}

	updated, err := bookmarks.Save(model.BookmarkInput{
		ID: saved.ID, Name: "Work", Kind: model.BookmarkKindLocal, Path: dir,
	})
	if err != nil {
		t.Fatalf("Update local bookmark: %v", err)
	}
	if updated.ID != saved.ID || updated.Name != "Work" {
		t.Fatalf("update changed identity or name: %#v", updated)
	}
	if err := bookmarks.Remove(saved.ID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	items, err = bookmarks.List()
	if err != nil || len(items) != 0 {
		t.Fatalf("List after remove = %#v, %v", items, err)
	}
}

func TestBookmarksRejectRelativeLocalAndNetworkFieldsOnLocal(t *testing.T) {
	bookmarks := NewBookmarks(New(t.TempDir()))
	if _, err := bookmarks.Save(model.BookmarkInput{Name: "bad", Kind: model.BookmarkKindLocal, Path: "relative/path"}); err == nil {
		t.Fatal("relative local bookmark was accepted")
	}
	if _, err := bookmarks.Save(model.BookmarkInput{
		Name: "bad", Kind: model.BookmarkKindLocal, Path: t.TempDir(), Host: "example.com",
	}); err == nil {
		t.Fatal("local bookmark with network identity was accepted")
	}
}

func TestRemoteBookmarksAreAccountBoundAndPersistNoSecretSchema(t *testing.T) {
	dir := t.TempDir()
	bookmarks := NewBookmarks(New(dir))
	saved, err := bookmarks.Save(model.BookmarkInput{
		Name: "Web root", Kind: model.BookmarkKindRemote, Path: "/public_html",
		Protocol: "SFTP", Host: " Example.COM. ", Port: 22, Username: "alice",
	})
	if err != nil {
		t.Fatalf("Save remote bookmark: %v", err)
	}
	if saved.Protocol != "sftp" || saved.Host != "Example.COM." {
		t.Fatalf("unexpected public identity normalization: %#v", saved)
	}
	if !RemoteBookmarkMatchesAccount(saved, model.ConnectionConfig{Protocol: "sftp", Host: "example.com", Port: 22, Username: "alice"}) {
		t.Fatal("canonical same-account connection did not match remote bookmark")
	}
	if RemoteBookmarkMatchesAccount(saved, model.ConnectionConfig{Protocol: "sftp", Host: "example.com", Port: 22, Username: "bob"}) {
		t.Fatal("remote bookmark crossed account boundary")
	}

	data, err := os.ReadFile(filepath.Join(dir, bookmarksFile))
	if err != nil {
		t.Fatalf("Read persisted bookmarks: %v", err)
	}
	text := strings.ToLower(string(data))
	for _, forbidden := range []string{"password", "passphrase", "privatekey", "fingerprint", "profileid"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("non-secret bookmark file contains forbidden field %q: %s", forbidden, text)
		}
	}
}

func TestBookmarksCorruptExistingStateFailsClosed(t *testing.T) {
	dir := t.TempDir()
	bookmarks := NewBookmarks(New(dir))
	if err := os.WriteFile(filepath.Join(dir, bookmarksFile), []byte("{not-json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := bookmarks.List(); err == nil {
		t.Fatal("corrupt existing bookmark state silently became an empty list")
	}
	if _, err := bookmarks.Save(model.BookmarkInput{Name: "new", Kind: model.BookmarkKindLocal, Path: dir}); err == nil {
		t.Fatal("save replaced corrupt bookmark state instead of failing closed")
	}
}

func TestBookmarksRejectUnknownEditAndMissingRemove(t *testing.T) {
	bookmarks := NewBookmarks(New(t.TempDir()))
	if _, err := bookmarks.Save(model.BookmarkInput{
		ID: strings.Repeat("a", 24), Name: "missing", Kind: model.BookmarkKindLocal, Path: t.TempDir(),
	}); err == nil {
		t.Fatal("unknown bookmark edit was accepted")
	}
	if err := bookmarks.Remove(strings.Repeat("b", 24)); err == nil {
		t.Fatal("missing bookmark removal was accepted")
	}
}
