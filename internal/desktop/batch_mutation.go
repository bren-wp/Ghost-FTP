package desktop

import (
	"context"
	"errors"
)

type batchMutationResult struct {
	Succeeded int
	Failed    int
	Err       error
}

// executeBatchMutation keeps partial-success accounting independent from the
// Windows UI. A cancelled/deadline context stops new mutations immediately and
// counts the remaining work as failed/not attempted so the caller can present
// an honest summary instead of a false all-or-nothing result.
func executeBatchMutation(ctx context.Context, count int, operation func(context.Context, int) error) batchMutationResult {
	if count <= 0 {
		return batchMutationResult{}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if operation == nil {
		return batchMutationResult{Failed: count, Err: errors.New("batch operacija nije dostupna")}
	}

	result := batchMutationResult{}
	var errs []error
	for i := 0; i < count; i++ {
		if err := ctx.Err(); err != nil {
			result.Failed += count - i
			errs = append(errs, err)
			break
		}
		err := operation(ctx, i)
		if err == nil {
			result.Succeeded++
			continue
		}
		result.Failed++
		errs = append(errs, err)
		if ctx.Err() != nil {
			result.Failed += count - i - 1
			break
		}
	}
	result.Err = errors.Join(errs...)
	return result
}
