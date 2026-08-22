package desktop

import (
	"context"
	"errors"
	"testing"
)

func TestExecuteBatchMutationTracksPartialSuccess(t *testing.T) {
	wantErr := errors.New("druga stavka nije uspjela")
	result := executeBatchMutation(context.Background(), 3, func(_ context.Context, index int) error {
		if index == 1 {
			return wantErr
		}
		return nil
	})
	if result.Succeeded != 2 || result.Failed != 1 {
		t.Fatalf("unexpected counts: succeeded=%d failed=%d", result.Succeeded, result.Failed)
	}
	if !errors.Is(result.Err, wantErr) {
		t.Fatalf("expected joined error to contain %v, got %v", wantErr, result.Err)
	}
}

func TestExecuteBatchMutationStopsAfterContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	result := executeBatchMutation(ctx, 4, func(_ context.Context, index int) error {
		calls++
		if index == 1 {
			cancel()
			return context.Canceled
		}
		return nil
	})
	if calls != 2 {
		t.Fatalf("expected two operations before cancellation, got %d", calls)
	}
	if result.Succeeded != 1 || result.Failed != 3 {
		t.Fatalf("unexpected counts after cancellation: succeeded=%d failed=%d", result.Succeeded, result.Failed)
	}
	if !errors.Is(result.Err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", result.Err)
	}
}

func TestExecuteBatchMutationNilOperationFailsClosed(t *testing.T) {
	result := executeBatchMutation(context.Background(), 2, nil)
	if result.Succeeded != 0 || result.Failed != 2 || result.Err == nil {
		t.Fatalf("unexpected nil-operation result: %+v", result)
	}
}
