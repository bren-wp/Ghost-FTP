package config

import (
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/profilebinding"
	"github.com/bren-wp/Ghost-FTP/internal/security"
)

const (
	bookmarksFile = "bookmarks.json"
	maxBookmarks  = 256
)

type Bookmarks struct {
	store *Store
	mu    sync.Mutex
}

func NewBookmarks(store *Store) *Bookmarks { return &Bookmarks{store: store} }

func validBookmarkID(id string) bool {
	if len(id) != 24 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func normalizeBookmark(b model.Bookmark) (model.Bookmark, error) {
	b.ID = strings.TrimSpace(b.ID)
	b.Name = strings.TrimSpace(b.Name)
	b.Kind = strings.ToLower(strings.TrimSpace(b.Kind))
	if !validBookmarkID(b.ID) {
		return model.Bookmark{}, errors.New("bookmark ima neispravan identitet")
	}
	if b.Name == "" || len(b.Name) > 120 || !utf8.ValidString(b.Name) || strings.ContainsAny(b.Name, "\x00\r\n") {
		return model.Bookmark{}, errors.New("naziv bookmarka je neispravan")
	}
	if b.Path == "" || len(b.Path) > 32767 || !utf8.ValidString(b.Path) || strings.ContainsAny(b.Path, "\x00\r\n") {
		return model.Bookmark{}, errors.New("putanja bookmarka je neispravna")
	}

	switch b.Kind {
	case model.BookmarkKindLocal:
		if !filepath.IsAbs(b.Path) {
			return model.Bookmark{}, errors.New("lokalni bookmark mora koristiti apsolutnu putanju")
		}
		b.Path = filepath.Clean(b.Path)
		// A local bookmark is machine-local navigation metadata. Network/account
		// fields are rejected instead of silently persisting irrelevant identity.
		if b.Protocol != "" || b.Host != "" || b.Port != 0 || b.Username != "" {
			return model.Bookmark{}, errors.New("lokalni bookmark ne smije sadržavati mrežni identitet")
		}
	case model.BookmarkKindRemote:
		if err := security.ValidateRemotePath(b.Path); err != nil {
			return model.Bookmark{}, err
		}
		b.Protocol = strings.ToLower(strings.TrimSpace(b.Protocol))
		b.Host = strings.TrimSpace(b.Host)
		if err := security.ValidateConnection(b.Protocol, b.Host, b.Username, b.Port); err != nil {
			return model.Bookmark{}, err
		}
	default:
		return model.Bookmark{}, errors.New("vrsta bookmarka je neispravna")
	}
	return b, nil
}

func (b *Bookmarks) loadLocked() ([]model.Bookmark, error) {
	var items []model.Bookmark
	source, err := b.store.Read(bookmarksFile, []model.Bookmark{}, &items)
	if err != nil {
		return nil, err
	}
	if source == "fallback" {
		// Missing state means an empty bookmark collection. Existing state that
		// could not be decoded is different: fail closed so a later Save cannot
		// silently replace recoverable user metadata with an empty generation.
		for _, name := range []string{bookmarksFile, bookmarksFile + ".previous"} {
			if _, statErr := os.Lstat(filepath.Join(b.store.Dir(), name)); statErr == nil {
				return nil, errors.New("spremljene bookmarke nije moguće pročitati")
			} else if !errors.Is(statErr, os.ErrNotExist) {
				return nil, statErr
			}
		}
		return []model.Bookmark{}, nil
	}
	if len(items) > maxBookmarks {
		return nil, errors.New("spremljeno je previše bookmarka")
	}
	seen := make(map[string]struct{}, len(items))
	for i := range items {
		normalized, err := normalizeBookmark(items[i])
		if err != nil {
			return nil, errors.New("spremljeni bookmark je neispravan: " + err.Error())
		}
		if _, exists := seen[normalized.ID]; exists {
			return nil, errors.New("spremljeni bookmark identitet je dupliciran")
		}
		seen[normalized.ID] = struct{}{}
		items[i] = normalized
	}
	return items, nil
}

func sortBookmarks(items []model.Bookmark) {
	sort.SliceStable(items, func(i, j int) bool {
		left := strings.ToLower(items[i].Name)
		right := strings.ToLower(items[j].Name)
		if left != right {
			return left < right
		}
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		return items[i].ID < items[j].ID
	})
}

func (b *Bookmarks) List() ([]model.Bookmark, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	items, err := b.loadLocked()
	if err != nil {
		return nil, err
	}
	sortBookmarks(items)
	return append([]model.Bookmark(nil), items...), nil
}

func (b *Bookmarks) Save(in model.BookmarkInput) (model.Bookmark, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	items, err := b.loadLocked()
	if err != nil {
		return model.Bookmark{}, err
	}

	id := strings.TrimSpace(in.ID)
	index := -1
	if id != "" {
		for i := range items {
			if items[i].ID == id {
				index = i
				break
			}
		}
		if index < 0 {
			return model.Bookmark{}, errors.New("bookmark za izmjenu nije pronađen")
		}
	} else {
		if len(items) >= maxBookmarks {
			return model.Bookmark{}, errors.New("dosegnut je maksimalan broj bookmarka")
		}
		id, err = randomID()
		if err != nil {
			return model.Bookmark{}, err
		}
	}

	bookmark, err := normalizeBookmark(model.Bookmark{
		ID: id, Name: in.Name, Kind: in.Kind, Path: in.Path,
		Protocol: in.Protocol, Host: in.Host, Port: in.Port, Username: in.Username,
	})
	if err != nil {
		return model.Bookmark{}, err
	}
	if index >= 0 {
		items[index] = bookmark
	} else {
		items = append(items, bookmark)
	}
	sortBookmarks(items)
	if err := b.store.Write(bookmarksFile, items); err != nil {
		return model.Bookmark{}, err
	}
	return bookmark, nil
}

func (b *Bookmarks) Remove(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	items, err := b.loadLocked()
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	for i := range items {
		if items[i].ID != id {
			continue
		}
		items = append(items[:i], items[i+1:]...)
		return b.store.Write(bookmarksFile, items)
	}
	return errors.New("bookmark nije pronađen")
}

// RemoteBookmarkMatchesAccount prevents one server-side path from being reused
// by another account on the same endpoint. Username is navigation identity here,
// not a credential; no password, key, passphrase or trust secret is persisted.
func RemoteBookmarkMatchesAccount(bookmark model.Bookmark, cfg model.ConnectionConfig) bool {
	if bookmark.Kind != model.BookmarkKindRemote {
		return false
	}
	return profilebinding.AccountMatches(
		bookmark.Protocol, bookmark.Host, bookmark.Port, bookmark.Username,
		cfg.Protocol, cfg.Host, cfg.Port, cfg.Username,
	)
}
