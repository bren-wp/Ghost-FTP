package remote

import (
	"context"
	"errors"
	"github.com/bren-wp/by-ftp/internal/model"
)

var ErrSkipped = errors.New("prijenos je preskočen jer odredišna datoteka već postoji")

type TransferOptions struct {
	KeepBackup   bool
	SkipExisting bool
}

type Session interface {
	Protocol() string
	Host() string
	Port() int
	List(context.Context, string) ([]model.Item, error)
	Mkdir(context.Context, string, string) error
	Rename(context.Context, string, string, string) error
	Delete(context.Context, string, string, bool) error
	Chmod(context.Context, string, string, string) error
	Upload(context.Context, string, string, TransferOptions) error
	Download(context.Context, string, string, TransferOptions) error
	Close() error
}
