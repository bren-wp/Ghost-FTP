package filesearch

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/itemlist"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/security"
)

// Local anchors the entire scan to one opened filesystem root. Child directory
// opens are root-relative, so a rename/symlink swap cannot redirect traversal
// outside the root capability captured at search start.
func Local(ctx context.Context, rootPath string, opts Options, emit EmitFunc) (Stats, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Stats{}, "", err
		}
		rootPath = home
	}
	absRoot, err := filepath.Abs(filepath.Clean(rootPath))
	if err != nil {
		return Stats{}, "", err
	}
	st, err := os.Lstat(absRoot)
	if err != nil {
		return Stats{}, "", err
	}
	if st.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(absRoot) {
		return Stats{}, "", errors.New("recursive search root must not be a symlink or reparse point")
	}
	if !st.IsDir() {
		return Stats{}, "", errors.New("recursive search root is not a directory")
	}

	root, err := os.OpenRoot(absRoot)
	if err != nil {
		return Stats{}, "", err
	}
	defer root.Close()

	withinRoot := func(target string) (string, error) {
		rel, err := filepath.Rel(absRoot, target)
		if err != nil {
			return "", err
		}
		rel = filepath.Clean(rel)
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			return "", errors.New("recursive search path escaped the opened local root")
		}
		return rel, nil
	}

	list := func(scanCtx context.Context, directory string) ([]model.Item, error) {
		if err := scanCtx.Err(); err != nil {
			return nil, err
		}
		rel, err := withinRoot(directory)
		if err != nil {
			return nil, err
		}
		f, err := root.Open(rel)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, errors.New("recursive search path is no longer a directory")
		}

		items := make([]model.Item, 0, 256)
		for {
			if err := scanCtx.Err(); err != nil {
				return nil, err
			}
			entries, readErr := f.ReadDir(256)
			for _, entry := range entries {
				if len(items) >= HardMaxVisited {
					return nil, errors.New("directory exceeds the recursive search safety ceiling")
				}
				entryPath := filepath.Join(directory, entry.Name())
				linkLike := entry.Type()&os.ModeSymlink != 0 || security.IsReparsePoint(entryPath)
				item := model.Item{
					Name:        entry.Name(),
					IsDirectory: entry.IsDir() && !linkLike,
					IsSymlink:   linkLike,
				}
				if entryInfo, infoErr := entry.Info(); infoErr == nil {
					item.Size = entryInfo.Size()
					item.Modified = entryInfo.ModTime()
				}
				items = append(items, item)
			}
			if errors.Is(readErr, io.EOF) {
				break
			}
			if readErr != nil {
				return nil, readErr
			}
		}
		itemlist.Sort(items)
		return items, nil
	}

	join := func(base, name string) (string, error) {
		child, err := security.SafeLocalChild(base, name)
		if err != nil {
			return "", err
		}
		if _, err := withinRoot(child); err != nil {
			return "", err
		}
		return child, nil
	}

	stats, err := Walk(ctx, absRoot, opts, list, join, emit)
	return stats, absRoot, err
}
