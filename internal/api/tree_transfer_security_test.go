package api

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/security"
)

func TestUploadTreeBoundaryRejectsLateRootRedirect(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "upload")
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}

	plan, err := buildUploadPlan(context.Background(), root, "/remote")
	if err != nil {
		t.Fatalf("buildUploadPlan: %v", err)
	}
	if len(plan.requests) != 1 {
		t.Fatalf("requests=%d want 1", len(plan.requests))
	}
	job := plan.requests[0]
	if job.LocalRoot != base {
		t.Fatalf("upload boundary=%q want parent %q", job.LocalRoot, base)
	}

	if err := os.Remove(filepath.Join(root, "file.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, root); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	if err := security.EnsureLocalWithinRoot(job.LocalRoot, job.LocalPath); err == nil {
		t.Fatal("late upload-root symlink redirect unexpectedly accepted")
	}
}

func TestPrepareLocalDirectoriesAnchorsOpenedBoundaryAcrossPathSwap(t *testing.T) {
	parent := t.TempDir()
	boundary := filepath.Join(parent, "boundary")
	movedBoundary := filepath.Join(parent, "boundary-opened")
	if err := os.Mkdir(boundary, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(boundary, "download")

	var swapErr error
	releasePrepared, err := prepareLocalDirectoriesWithHooks(boundary, []string{target}, &localDirectoryPreparationHooks{
		beforeMkdir: func(relative string) {
			if relative != "download" || swapErr != nil {
				return
			}
			if err := os.Rename(boundary, movedBoundary); err != nil {
				swapErr = err
				return
			}
			if err := os.Mkdir(boundary, 0700); err != nil {
				swapErr = err
			}
		},
	})
	if releasePrepared != nil {
		defer releasePrepared()
	}
	if swapErr != nil {
		t.Skipf("platform does not allow deterministic rename of opened boundary: %v", swapErr)
	}
	if err != nil {
		t.Fatalf("prepareLocalDirectoriesWithHooks: %v", err)
	}
	if st, err := os.Stat(filepath.Join(movedBoundary, "download")); err != nil || !st.IsDir() {
		t.Fatalf("directory was not created below opened boundary object, stat=%v err=%v", st, err)
	}
	if _, err := os.Stat(filepath.Join(boundary, "download")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("replacement boundary unexpectedly received prepared directory, err=%v", err)
	}
}

func TestPrepareLocalDirectoriesAnchorsOpenedParentAndNeverDeletesReplacement(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "download")
	nested := filepath.Join(root, "nested")
	movedRoot := filepath.Join(base, "download-opened")
	marker := filepath.Join(root, "replacement.txt")

	var swapErr error
	releasePrepared, err := prepareLocalDirectoriesWithHooks(base, []string{root, nested}, &localDirectoryPreparationHooks{
		beforeMkdir: func(relative string) {
			if relative != filepath.Join("download", "nested") || swapErr != nil {
				return
			}
			if err := os.Rename(root, movedRoot); err != nil {
				swapErr = err
				return
			}
			if err := os.Mkdir(root, 0700); err != nil {
				swapErr = err
				return
			}
			if err := os.WriteFile(marker, []byte("replacement"), 0600); err != nil {
				swapErr = err
			}
		},
	})
	if swapErr != nil {
		if releasePrepared != nil {
			releasePrepared()
		}
		t.Skipf("platform does not allow deterministic rename of opened parent: %v", swapErr)
	}
	if err != nil {
		if releasePrepared != nil {
			releasePrepared()
		}
		t.Fatalf("prepareLocalDirectoriesWithHooks: %v", err)
	}

	if st, err := os.Stat(filepath.Join(movedRoot, "nested")); err != nil || !st.IsDir() {
		releasePrepared()
		t.Fatalf("nested directory was not created below held parent object, stat=%v err=%v", st, err)
	}
	if _, err := os.Stat(filepath.Join(root, "nested")); !errors.Is(err, os.ErrNotExist) {
		releasePrepared()
		t.Fatalf("replacement parent unexpectedly received nested directory, err=%v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		releasePrepared()
		t.Fatalf("replacement marker missing before rooted handles are released: %v", err)
	}

	releasePrepared()
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("releasing preparation deleted replacement user content: %v", err)
	}
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		t.Fatalf("replacement directory must survive release, stat=%v err=%v", st, err)
	}
}
