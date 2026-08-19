package remote

import (
	"context"
	"errors"
	"strings"
	"testing"
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
