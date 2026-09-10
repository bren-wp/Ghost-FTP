package api

import (
	"context"
	"errors"

	"github.com/bren-wp/Ghost-FTP/internal/config"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

type BookmarkNavigation struct {
	Bookmark  model.Bookmark
	LocalBase string
	Items     []model.Item
}

func (e *Engine) SaveLocalBookmark(id, name, path string) (model.Bookmark, error) {
	return e.bookmarks.Save(model.BookmarkInput{
		ID: id, Name: name, Kind: model.BookmarkKindLocal, Path: path,
	})
}

// SaveRemoteBookmark binds navigation metadata to the real active server
// account held by the remote manager. UI text is deliberately not accepted as
// host/account identity, and no profile is created as a side effect.
func (e *Engine) SaveRemoteBookmark(id, name, path string) (model.Bookmark, error) {
	cfg, connected := e.remote.Config()
	if !connected {
		return model.Bookmark{}, errors.New("remote bookmark zahtijeva aktivnu vezu")
	}
	return e.bookmarks.Save(model.BookmarkInput{
		ID: id, Name: name, Kind: model.BookmarkKindRemote, Path: path,
		Protocol: cfg.Protocol, Host: cfg.Host, Port: cfg.Port, Username: cfg.Username,
	})
}

// NavigateBookmark proves the target before any desktop pane commits it. Local
// bookmarks must successfully list as a real directory. Remote bookmarks must
// match the active account, list through that session, and still see the same
// connection identity after listing; reconnect/account changes therefore cannot
// turn a stale bookmark into navigation authority for another server context.
func (e *Engine) NavigateBookmark(ctx context.Context, id string) (BookmarkNavigation, error) {
	bookmark, err := e.bookmarks.Get(id)
	if err != nil {
		return BookmarkNavigation{}, err
	}

	switch bookmark.Kind {
	case model.BookmarkKindLocal:
		base, items, err := e.local.ListContext(ctx, bookmark.Path)
		if err != nil {
			return BookmarkNavigation{}, err
		}
		return BookmarkNavigation{Bookmark: bookmark, LocalBase: base, Items: items}, nil

	case model.BookmarkKindRemote:
		cfg, connected := e.remote.Config()
		if !connected {
			return BookmarkNavigation{}, errors.New("remote bookmark zahtijeva aktivnu vezu")
		}
		if !config.RemoteBookmarkMatchesAccount(bookmark, cfg) {
			return BookmarkNavigation{}, errors.New("remote bookmark pripada drugom server računu")
		}
		identityBefore, err := e.remote.ConnectionIdentity()
		if err != nil {
			return BookmarkNavigation{}, err
		}
		items, err := e.RemoteList(ctx, bookmark.Path)
		if err != nil {
			return BookmarkNavigation{}, err
		}
		identityAfter, err := e.remote.ConnectionIdentity()
		if err != nil {
			return BookmarkNavigation{}, err
		}
		if identityBefore != identityAfter {
			return BookmarkNavigation{}, errors.New("veza se promijenila tijekom otvaranja bookmarka")
		}
		cfgAfter, connected := e.remote.Config()
		if !connected || !config.RemoteBookmarkMatchesAccount(bookmark, cfgAfter) {
			return BookmarkNavigation{}, errors.New("veza se promijenila tijekom otvaranja bookmarka")
		}
		return BookmarkNavigation{Bookmark: bookmark, Items: items}, nil
	default:
		return BookmarkNavigation{}, errors.New("vrsta bookmarka je neispravna")
	}
}
