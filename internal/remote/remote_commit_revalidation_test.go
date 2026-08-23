package remote

import (
	"context"
	"errors"
	"testing"

	"github.com/bren-wp/by-ftp/internal/model"
)

func TestRevalidateRemoteCommitSkipNewExistingFile(t *testing.T) {
	deleted := false
	items, err := revalidateRemoteCommit(context.Background(), "/dir", "file.txt", ".byftp-part-x", true,
		func(context.Context, string) ([]model.Item, error) {
			return []model.Item{{Name: "file.txt"}}, nil
		},
		func(_ context.Context, base, name string, isDir bool) error {
			if base != "/dir" || name != ".byftp-part-x" || isDir {
				t.Fatalf("unexpected cleanup target: %q %q dir=%v", base, name, isDir)
			}
			deleted = true
			return nil
		},
	)
	if !errors.Is(err, ErrSkipped) {
		t.Fatalf("expected ErrSkipped, got items=%v err=%v", items, err)
	}
	if !deleted {
		t.Fatal("temporary upload was not cleaned after SkipExisting revalidation")
	}
}

func TestRevalidateRemoteCommitRejectsNewDirectoryOrSymlink(t *testing.T) {
	for _, item := range []model.Item{
		{Name: "file.txt", IsDirectory: true},
		{Name: "file.txt", IsSymlink: true},
	} {
		deleted := false
		_, err := revalidateRemoteCommit(context.Background(), "/dir", "file.txt", ".byftp-part-x", false,
			func(context.Context, string) ([]model.Item, error) { return []model.Item{item}, nil },
			func(context.Context, string, string, bool) error { deleted = true; return nil },
		)
		if err == nil {
			t.Fatalf("dangerous target was accepted: %+v", item)
		}
		if !deleted {
			t.Fatalf("temporary upload was not cleaned: %+v", item)
		}
	}
}

func TestRevalidateRemoteCommitCleansTempOnListFailure(t *testing.T) {
	deleted := false
	listErr := errors.New("listing failed")
	_, err := revalidateRemoteCommit(context.Background(), "/dir", "file.txt", ".byftp-part-x", false,
		func(context.Context, string) ([]model.Item, error) { return nil, listErr },
		func(context.Context, string, string, bool) error { deleted = true; return nil },
	)
	if !errors.Is(err, listErr) {
		t.Fatalf("expected wrapped list error, got %v", err)
	}
	if !deleted {
		t.Fatal("temporary upload was not cleaned after list failure")
	}
}

func TestRevalidateRemoteCommitReturnsFreshOverwriteSnapshot(t *testing.T) {
	fresh := []model.Item{{Name: "file.txt", Size: 42}}
	deleted := false
	items, err := revalidateRemoteCommit(context.Background(), "/dir", "file.txt", ".byftp-part-x", false,
		func(context.Context, string) ([]model.Item, error) { return fresh, nil },
		func(context.Context, string, string, bool) error { deleted = true; return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if deleted {
		t.Fatal("temporary upload must remain for valid overwrite commit")
	}
	if len(items) != 1 || items[0].Size != 42 {
		t.Fatalf("fresh snapshot was not returned: %+v", items)
	}
}

func TestRevalidateRemoteCommitAllowsStillMissingTarget(t *testing.T) {
	items, err := revalidateRemoteCommit(context.Background(), "/dir", "file.txt", ".byftp-part-x", true,
		func(context.Context, string) ([]model.Item, error) { return nil, nil },
		func(context.Context, string, string, bool) error { t.Fatal("unexpected cleanup"); return nil },
	)
	if err != nil || len(items) != 0 {
		t.Fatalf("missing target should be commit-ready: items=%v err=%v", items, err)
	}
}
