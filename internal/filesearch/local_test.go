package filesearch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestLocalSearchFindsNestedFilesAndReturnsCanonicalRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "Quarterly-Report.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	var got []Result
	stats, canonical, err := Local(context.Background(), root, Options{Query: "report"}, func(batch []Result) error {
		got = append(got, batch...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	wantRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	if canonical != wantRoot {
		t.Fatalf("canonical=%q want=%q", canonical, wantRoot)
	}
	if len(got) != 1 || got[0].Path != filepath.Join(wantRoot, "docs", "Quarterly-Report.txt") || stats.Results != 1 {
		t.Fatalf("got=%#v stats=%#v", got, stats)
	}
}

func TestLocalSearchDoesNotTraverseListedSymlink(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "secret-match.txt"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked")
	if err := os.Symlink(external, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink creation unavailable: %v", err)
		}
		t.Fatal(err)
	}
	var got []Result
	_, _, err := Local(context.Background(), root, Options{Query: "secret-match"}, func(batch []Result) error {
		got = append(got, batch...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("search followed symlink outside root: %#v", got)
	}
}

func TestLocalSearchRejectsSymlinkRoot(t *testing.T) {
	target := t.TempDir()
	linkParent := t.TempDir()
	link := filepath.Join(linkParent, "root-link")
	if err := os.Symlink(target, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink creation unavailable: %v", err)
		}
		t.Fatal(err)
	}
	_, _, err := Local(context.Background(), link, Options{Query: "x"}, nil)
	if err == nil {
		t.Fatal("expected symlink root rejection")
	}
}

func TestLocalSearchPropagatesEmitterFailure(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "match.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	emitErr := errors.New("consumer stopped")
	_, _, err := Local(context.Background(), root, Options{Query: "match"}, func(_ []Result) error {
		return emitErr
	})
	if !errors.Is(err, emitErr) {
		t.Fatalf("err=%v", err)
	}
}

func TestLocalSearchResultIsReadOnlySnapshotShape(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "match.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	var result Result
	_, _, err := Local(context.Background(), root, Options{Query: "match"}, func(batch []Result) error {
		result = batch[0]
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Item != (model.Item{Name: "match.txt", Size: 1, Modified: result.Item.Modified}) {
		t.Fatalf("unexpected snapshot: %#v", result)
	}
	if result.Parent != root && result.Parent != filepath.Clean(root) {
		t.Fatalf("parent=%q root=%q", result.Parent, root)
	}
}
