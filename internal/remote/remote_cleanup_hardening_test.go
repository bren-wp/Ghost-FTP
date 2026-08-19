package remote

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"brendigo.com/byftp/internal/model"
)

func TestCleanupFailureReturnsOriginalWhenCleanupSucceeds(t *testing.T) {
	original := errors.New("connection reset")
	err := cleanupFailure(original, "/www", ".byftp-part-test", func(context.Context, string, string, bool) error {
		return nil
	})
	if !errors.Is(err, original) {
		t.Fatalf("original error was not preserved: %v", err)
	}
	if isRemoteResidualArtifactError(err) {
		t.Fatalf("successful cleanup must not create residual error: %v", err)
	}
}

func TestCleanupFailureAcceptsConfirmedMissingArtifact(t *testing.T) {
	original := errors.New("upload failed")
	err := cleanupFailure(original, "/www", ".byftp-part-test", func(context.Context, string, string, bool) error {
		return errors.New("No such file")
	})
	if !errors.Is(err, original) || isRemoteResidualArtifactError(err) {
		t.Fatalf("confirmed missing artifact should preserve original error: %v", err)
	}
}

func TestCleanupFailureMarksUncertainRemoteState(t *testing.T) {
	original := errors.New("connection reset")
	cleanup := errors.New("permission denied")
	err := cleanupFailure(original, "/www", ".byftp-part-test", func(context.Context, string, string, bool) error {
		return cleanup
	})
	if !isRemoteResidualArtifactError(err) {
		t.Fatalf("cleanup failure must produce residual error: %v", err)
	}
	if !errors.Is(err, original) || !errors.Is(err, cleanup) {
		t.Fatalf("residual error must preserve both causes: %v", err)
	}
	if !strings.Contains(err.Error(), "/www/.byftp-part-test") {
		t.Fatalf("residual object is not identified: %v", err)
	}
}

func TestCommittedCleanupFailureMarksCommittedState(t *testing.T) {
	cleanup := errors.New("permission denied")
	err := committedCleanupFailure(nil, "/www", ".byftp-rollback-test", func(context.Context, string, string, bool) error {
		return cleanup
	})
	var residual *remoteResidualArtifactError
	if !errors.As(err, &residual) || !residual.committed {
		t.Fatalf("post-commit cleanup must be marked committed: %v", err)
	}
	if !strings.Contains(err.Error(), "aktivirana") {
		t.Fatalf("post-commit error must not imply that activation failed: %v", err)
	}
}

func TestResidualCleanupErrorBlocksAutomaticRetry(t *testing.T) {
	transport := &toolError{tool: "curl", code: 56, message: "recv failure"}
	if !IsRetryable(transport) {
		t.Fatal("transport interruption should be retryable before cleanup uncertainty")
	}
	err := cleanupFailure(transport, "/www", ".byftp-part-test", func(context.Context, string, string, bool) error {
		return errors.New("connection reset while deleting")
	})
	if IsRetryable(err) {
		t.Fatalf("uncertain remote state must block automatic retry: %v", err)
	}
}

func TestCommitRemoteTempReportsPostCommitRollbackCleanupFailure(t *testing.T) {
	var renamed [][2]string
	var deleted []string
	ops := remoteCommitOps{
		rename: func(_ context.Context, _ string, from, to string) error {
			renamed = append(renamed, [2]string{from, to})
			return nil
		},
		delete: func(_ context.Context, _ string, name string, _ bool) error {
			deleted = append(deleted, name)
			return errors.New("permission denied")
		},
	}
	err := commitRemoteTemp(context.Background(), []model.Item{{Name: "index.html"}}, "/www", "index.html", ".part", ".byftp-rollback-test", false, ops)
	if err == nil || !isRemoteResidualArtifactError(err) || IsRetryable(err) {
		t.Fatalf("post-commit cleanup failure must be explicit and non-retryable: %v", err)
	}
	wantRenamed := [][2]string{{"index.html", ".byftp-rollback-test"}, {".part", "index.html"}}
	if !reflect.DeepEqual(renamed, wantRenamed) {
		t.Fatalf("activation sequence=%v want=%v", renamed, wantRenamed)
	}
	if !reflect.DeepEqual(deleted, []string{".byftp-rollback-test"}) {
		t.Fatalf("cleanup target=%v", deleted)
	}
}

func TestRevalidateRemoteCommitSurfacesCleanupFailure(t *testing.T) {
	listErr := errors.New("connection reset")
	_, err := revalidateRemoteCommit(context.Background(), "/www", "index.html", ".part", false,
		func(context.Context, string) ([]model.Item, error) { return nil, listErr },
		func(context.Context, string, string, bool) error { return errors.New("permission denied") },
	)
	if err == nil || !isRemoteResidualArtifactError(err) || IsRetryable(err) {
		t.Fatalf("revalidation cleanup failure must block retry: %v", err)
	}
	if !errors.Is(err, listErr) {
		t.Fatalf("revalidation cause was lost: %v", err)
	}
}

func TestReplaceLocalFileAtomicRemovesPartAfterFinalActivationFailure(t *testing.T) {
	dir := t.TempDir()
	part := filepath.Join(dir, "part.tmp")
	if err := os.WriteFile(part, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	// The target parent does not exist. This deterministically exercises the
	// no-existing-target final RenameNoReplace failure branch.
	local := filepath.Join(dir, "missing", "target.txt")
	if err := replaceLocalFileAtomic(local, part, false); err == nil {
		t.Fatal("expected final activation failure")
	}
	if _, err := os.Lstat(part); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed download part was left behind: %v", err)
	}
}
