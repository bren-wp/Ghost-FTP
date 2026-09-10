package filesearch

import (
	"context"
	"errors"

	"github.com/bren-wp/Ghost-FTP/internal/itemlist"
)

type directoryNode struct {
	path  string
	depth int
}

// Walk performs a breadth-first, read-only recursive name search. All expensive
// I/O is owned by the supplied ListFunc, every discovered child path must pass
// through JoinFunc, and results are emitted in bounded batches. The walker never
// exposes an operation that mutates a discovered object.
func Walk(parent context.Context, root string, in Options, list ListFunc, join JoinFunc, emit EmitFunc) (Stats, error) {
	if parent == nil {
		parent = context.Background()
	}
	opts, err := NormalizeOptions(in)
	if err != nil {
		return Stats{}, err
	}
	if list == nil || join == nil {
		return Stats{}, errors.New("recursive search requires listing and path-join functions")
	}

	scanCtx, cancel := context.WithTimeout(parent, opts.Timeout)
	defer cancel()

	stats := Stats{}
	queue := []directoryNode{{path: root, depth: 0}}
	batch := make([]Result, 0, opts.BatchSize)
	depthLimited := false

	flush := func() error {
		if len(batch) == 0 || emit == nil {
			batch = batch[:0]
			return nil
		}
		payload := append([]Result(nil), batch...)
		batch = batch[:0]
		return emit(payload)
	}
	stopForContext := func() (bool, error) {
		if scanCtx.Err() == nil {
			return false, nil
		}
		if parent.Err() != nil {
			return true, parent.Err()
		}
		if errors.Is(scanCtx.Err(), context.DeadlineExceeded) {
			stats.StopReason = StopTimeLimit
			return true, nil
		}
		return true, scanCtx.Err()
	}

	for len(queue) > 0 {
		if stop, stopErr := stopForContext(); stop {
			if flushErr := flush(); flushErr != nil {
				return stats, flushErr
			}
			return stats, stopErr
		}

		node := queue[0]
		queue = queue[1:]
		items, listErr := list(scanCtx, node.path)
		if listErr != nil {
			if stop, stopErr := stopForContext(); stop {
				if flushErr := flush(); flushErr != nil {
					return stats, flushErr
				}
				return stats, stopErr
			}
			return stats, listErr
		}
		stats.Directories++

		for _, item := range items {
			if stop, stopErr := stopForContext(); stop {
				if flushErr := flush(); flushErr != nil {
					return stats, flushErr
				}
				return stats, stopErr
			}
			if stats.Visited >= opts.MaxVisited {
				stats.StopReason = StopItemLimit
				if err := flush(); err != nil {
					return stats, err
				}
				return stats, nil
			}

			childPath, err := join(node.path, item.Name)
			if err != nil {
				return stats, err
			}
			stats.Visited++

			if itemlist.MatchesName(item.Name, opts.Query) {
				batch = append(batch, Result{
					Path:   childPath,
					Parent: node.path,
					Depth:  node.depth + 1,
					Item:   item,
				})
				stats.Results++
				if len(batch) >= opts.BatchSize {
					if err := flush(); err != nil {
						return stats, err
					}
				}
				if stats.Results >= opts.MaxResults {
					stats.StopReason = StopResultLimit
					if err := flush(); err != nil {
						return stats, err
					}
					return stats, nil
				}
			}

			if item.IsDirectory && !item.IsSymlink {
				if node.depth < opts.MaxDepth {
					queue = append(queue, directoryNode{path: childPath, depth: node.depth + 1})
				} else {
					depthLimited = true
				}
			}
		}
	}

	if err := flush(); err != nil {
		return stats, err
	}
	if depthLimited {
		stats.StopReason = StopDepthLimit
	}
	return stats, nil
}
