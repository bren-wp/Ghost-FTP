package localfs

import (
	"brendigo.com/byftp/internal/itemlist"
	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Service struct{}

func New() *Service { return &Service{} }
func cleanPath(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return os.UserHomeDir()
	}
	return filepath.Abs(filepath.Clean(p))
}
func (s *Service) List(p string) (string, []model.Item, error) {
	return s.ListContext(context.Background(), p)
}

func (s *Service) ListContext(ctx context.Context, p string) (string, []model.Item, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	p, err := cleanPath(p)
	if err != nil {
		return "", nil, err
	}
	st, err := os.Stat(p)
	if err != nil {
		return "", nil, err
	}
	if !st.IsDir() {
		return "", nil, errors.New("lokalna putanja nije direktorij")
	}
	f, err := os.Open(p)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	const maxLocalItems = 50000
	items := make([]model.Item, 0, 512)
	for {
		if err := ctx.Err(); err != nil {
			return "", nil, err
		}
		ents, readErr := f.ReadDir(512)
		for _, e := range ents {
			if len(items) >= maxLocalItems {
				return "", nil, errors.New("mapa sadrži previše stavki za siguran prikaz")
			}
			entryPath := filepath.Join(p, e.Name())
			linkLike := e.Type()&os.ModeSymlink != 0 || security.IsReparsePoint(entryPath)
			item := model.Item{Name: e.Name(), IsDirectory: e.IsDir() && !linkLike, IsSymlink: linkLike}
			if info, er := e.Info(); er == nil {
				item.Size = info.Size()
				item.Modified = info.ModTime()
			}
			items = append(items, item)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return "", nil, readErr
		}
	}
	itemlist.Sort(items)
	return p, items, nil
}
func (s *Service) Mkdir(base, name string) error {
	base, err := cleanPath(base)
	if err != nil {
		return err
	}
	p, err := security.SafeLocalChild(base, name)
	if err != nil {
		return err
	}
	return os.Mkdir(p, 0755)
}
func (s *Service) Rename(base, oldName, newName string) error {
	base, err := cleanPath(base)
	if err != nil {
		return err
	}
	a, err := security.SafeLocalChild(base, oldName)
	if err != nil {
		return err
	}
	b, err := security.SafeLocalChild(base, newName)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(b); err == nil {
		return errors.New("ciljna stavka već postoji")
	} else if !os.IsNotExist(err) {
		return err
	}
	return platform.RenameNoReplace(a, b)
}
func (s *Service) Delete(base, name string) error {
	base, err := cleanPath(base)
	if err != nil {
		return err
	}
	p, err := security.SafeLocalChild(base, name)
	if err != nil {
		return err
	}
	if filepath.Clean(p) == filepath.Clean(base) {
		return errors.New("nije dopušteno brisanje korijenskog direktorija")
	}
	if _, err := os.Lstat(p); err != nil {
		if os.IsNotExist(err) {
			return errors.New("lokalna stavka više ne postoji")
		}
		return err
	}
	return security.RemoveTreeNoFollow(p)
}
