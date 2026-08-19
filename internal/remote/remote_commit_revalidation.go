package remote

import (
	"context"
	"errors"
	"fmt"

	"brendigo.com/byftp/internal/model"
)

type remoteListFunc func(context.Context, string) ([]model.Item, error)
type remoteDeleteFunc func(context.Context, string, string, bool) error

// revalidateRemoteCommit refreshes the destination state after the temporary
// upload has finished. A pre-upload directory listing can be minutes old for a
// large transfer, so commit decisions must not rely on it.
func revalidateRemoteCommit(
	ctx context.Context,
	dir, base, tempName string,
	skipExisting bool,
	list remoteListFunc,
	delete remoteDeleteFunc,
) ([]model.Item, error) {
	if list == nil || delete == nil {
		return nil, errors.New("remote revalidacija nije dostupna")
	}
	cleanupTemp := func() {
		cleanupCtx, cancel := cleanupContext()
		defer cancel()
		_ = delete(cleanupCtx, dir, tempName, false)
	}

	items, err := list(ctx, dir)
	if err != nil {
		cleanupTemp()
		return nil, fmt.Errorf("nije moguće ponovno provjeriti remote odredište prije aktivacije: %w", err)
	}
	// Neki FTP LIST fallbackovi ne prikazuju skrivene .byftp-part-* datoteke,
	// zato odsutnost staging stavke u listingu sama po sebi nije dokaz greške.
	// Ako je server ipak vrati, ona mora ostati obična datoteka; direktorij ili
	// symlink pod našim nasumičnim staging nazivom nikada se ne smije aktivirati.
	if staged, ok := remoteEntry(items, tempName); ok && (staged.IsDirectory || staged.IsSymlink) {
		cleanupTemp()
		return nil, errors.New("remote privremena stavka se promijenila i nije obična datoteka")
	}
	if existing, ok := remoteEntry(items, base); ok {
		if existing.IsDirectory || existing.IsSymlink {
			cleanupTemp()
			return nil, errors.New("remote odredište se promijenilo tijekom prijenosa i više nije obična datoteka")
		}
		if skipExisting {
			cleanupTemp()
			return nil, ErrSkipped
		}
	}
	return items, nil
}
