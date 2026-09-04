package api

import (
	"context"
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
