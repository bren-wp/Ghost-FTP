package api

import (
	"context"
	"errors"
	"path"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/filesearch"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/security"
)

type SearchOptions = filesearch.Options
type SearchResult = filesearch.Result
type SearchStats = filesearch.Stats
type SearchEmitFunc = filesearch.EmitFunc

func (e *Engine) SearchLocalRecursive(ctx context.Context, root string, opts SearchOptions, emit SearchEmitFunc) (SearchStats, string, error) {
	if e == nil || e.local == nil {
		return SearchStats{}, "", errors.New("local search service is unavailable")
	}
	return filesearch.Local(ctx, root, opts, emit)
}

// SearchRemoteRecursive holds one generation-bound remote operation for the
// whole scan. Disconnect cancels opCtx and waits for its release; a reconnect
// therefore cannot make an in-flight search continue on a different session.
func (e *Engine) SearchRemoteRecursive(ctx context.Context, root string, opts SearchOptions, emit SearchEmitFunc) (SearchStats, error) {
	if e == nil || e.remote == nil {
		return SearchStats{}, errors.New("remote search service is unavailable")
	}
	root = strings.TrimSpace(strings.ReplaceAll(root, "\\", "/"))
	if root == "" {
		root = "/"
	}
	if err := security.ValidateRemotePath(root); err != nil {
		return SearchStats{}, err
	}
	root = path.Clean(root)

	session, opCtx, release, err := e.remote.Operation(ctx)
	if err != nil {
		return SearchStats{}, err
	}
	defer release()

	list := func(scanCtx context.Context, directory string) ([]model.Item, error) {
		return session.List(scanCtx, directory)
	}
	join := func(base, name string) (string, error) {
		if name == "." || name == ".." {
			return "", errors.New("remote listing contains an unsafe traversal name")
		}
		if err := security.ValidateRemoteName(name); err != nil {
			return "", err
		}
		child := path.Clean(path.Join(base, name))
		if err := security.ValidateRemotePath(child); err != nil {
			return "", err
		}
		return child, nil
	}

	return filesearch.Walk(opCtx, root, opts, list, join, emit)
}
