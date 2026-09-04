package remote

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestRemoveCommittedLocalRollbackReportsCleanupFailure(t *testing.T) {
	const path = "/tmp/GhostFTP-rollback-sensitive"
	want := errors.New("permission denied")
	err := removeCommittedLocalRollback(path, func(got string) error {
		if got != path {
			t.Fatalf("cleanup path mismatch: got %q want %q", got, path)
		}
		return want
	})
	if err == nil {
		t.Fatal("rollback cleanup failure must not be reported as success")
	}
	if !errors.Is(err, want) {
		t.Fatalf("cleanup error lost original cause: %v", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("cleanup error must identify the residual rollback path: %v", err)
	}
}

func TestRemoveCommittedLocalRollbackAcceptsAlreadyAbsentFile(t *testing.T) {
	if err := removeCommittedLocalRollback("missing", func(string) error { return os.ErrNotExist }); err != nil {
		t.Fatalf("already-absent rollback should count as cleaned: %v", err)
	}
}
