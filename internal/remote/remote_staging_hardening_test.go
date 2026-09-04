package remote

import (
	"context"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestRevalidateRemoteCommitRejectsUnsafeStagingObject(t *testing.T) {
	for _, staged := range []model.Item{
		{Name: ".GhostFTP-part-x", IsDirectory: true},
		{Name: ".GhostFTP-part-x", IsSymlink: true},
	} {
		deleted := false
		_, err := revalidateRemoteCommit(
			context.Background(),
			"/dir",
			"file.txt",
			".GhostFTP-part-x",
			false,
			func(context.Context, string) ([]model.Item, error) {
				return []model.Item{staged}, nil
			},
			func(_ context.Context, base, name string, isDir bool) error {
				if base != "/dir" || name != ".GhostFTP-part-x" || isDir {
					t.Fatalf("unexpected cleanup target: base=%q name=%q dir=%v", base, name, isDir)
				}
				deleted = true
				return nil
			},
		)
		if err == nil {
			t.Fatalf("unsafe staging object was accepted: %+v", staged)
		}
		if !deleted {
			t.Fatalf("unsafe staging object was not cleaned: %+v", staged)
		}
	}
}

func TestRevalidateRemoteCommitAllowsHiddenStagingToBeAbsentFromListing(t *testing.T) {
	items, err := revalidateRemoteCommit(
		context.Background(),
		"/dir",
		"file.txt",
		".GhostFTP-part-x",
		false,
		func(context.Context, string) ([]model.Item, error) {
			return []model.Item{{Name: "file.txt", Size: 5}}, nil
		},
		func(context.Context, string, string, bool) error {
			t.Fatal("missing hidden staging entry must not trigger cleanup by itself")
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "file.txt" {
		t.Fatalf("unexpected revalidation snapshot: %+v", items)
	}
}
