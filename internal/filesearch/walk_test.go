package filesearch

import (
	"context"
	"errors"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestNormalizeOptionsAppliesSafeDefaults(t *testing.T) {
	got, err := NormalizeOptions(Options{Query: " report "})
	if err != nil {
		t.Fatal(err)
	}
	if got.Query != "report" || got.MaxDepth != DefaultMaxDepth || got.MaxVisited != DefaultMaxVisited || got.MaxResults != DefaultMaxResults || got.BatchSize != DefaultBatchSize || got.Timeout != DefaultTimeout {
		t.Fatalf("unexpected defaults: %#v", got)
	}
}

func TestNormalizeOptionsRejectsEmptyAndUnsafeBounds(t *testing.T) {
	cases := []Options{
		{},
		{Query: "x", MaxDepth: HardMaxDepth + 1},
		{Query: "x", MaxVisited: HardMaxVisited + 1},
		{Query: "x", MaxResults: HardMaxResults + 1},
		{Query: "x", BatchSize: HardMaxBatchSize + 1},
		{Query: "x", Timeout: HardMaxTimeout + time.Second},
	}
	for _, tc := range cases {
		if _, err := NormalizeOptions(tc); err == nil {
			t.Fatalf("expected rejection for %#v", tc)
		}
	}
}

func TestWalkEmitsIncrementallyAndNeverTraversesSymlink(t *testing.T) {
	tree := map[string][]model.Item{
		"/": {
			{Name: "alpha.txt"},
			{Name: "folder", IsDirectory: true},
			{Name: "link", IsSymlink: true},
		},
		"/folder": {
			{Name: "ALPHA-2.txt"},
			{Name: "other.txt"},
		},
	}
	listed := []string{}
	list := func(_ context.Context, directory string) ([]model.Item, error) {
		listed = append(listed, directory)
		items, ok := tree[directory]
		if !ok {
			return nil, errors.New("unexpected traversal")
		}
		return items, nil
	}
	join := func(base, name string) (string, error) { return path.Join(base, name), nil }
	var batches [][]Result
	stats, err := Walk(context.Background(), "/", Options{Query: "alpha", BatchSize: 1}, list, join, func(batch []Result) error {
		batches = append(batches, batch)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(listed, []string{"/", "/folder"}) {
		t.Fatalf("listed=%v", listed)
	}
	if stats.Results != 2 || stats.Directories != 2 || stats.Visited != 5 || stats.StopReason != StopNone {
		t.Fatalf("stats=%#v", stats)
	}
	if len(batches) != 2 || len(batches[0]) != 1 || len(batches[1]) != 1 {
		t.Fatalf("batches=%#v", batches)
	}
	if batches[0][0].Path != "/alpha.txt" || batches[1][0].Path != "/folder/ALPHA-2.txt" {
		t.Fatalf("unexpected results: %#v", batches)
	}
}

func TestWalkStopsAtVisitedLimit(t *testing.T) {
	list := func(_ context.Context, _ string) ([]model.Item, error) {
		return []model.Item{{Name: "a"}, {Name: "b"}, {Name: "c"}}, nil
	}
	stats, err := Walk(context.Background(), "/", Options{Query: "missing", MaxVisited: 2}, list, func(base, name string) (string, error) {
		return path.Join(base, name), nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Visited != 2 || stats.StopReason != StopItemLimit {
		t.Fatalf("stats=%#v", stats)
	}
}

func TestWalkStopsAtResultLimitAfterEmittingResult(t *testing.T) {
	list := func(_ context.Context, _ string) ([]model.Item, error) {
		return []model.Item{{Name: "match-a"}, {Name: "match-b"}}, nil
	}
	var got []Result
	stats, err := Walk(context.Background(), "/", Options{Query: "match", MaxResults: 1, BatchSize: 10}, list, func(base, name string) (string, error) {
		return path.Join(base, name), nil
	}, func(batch []Result) error {
		got = append(got, batch...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || stats.Results != 1 || stats.StopReason != StopResultLimit {
		t.Fatalf("results=%#v stats=%#v", got, stats)
	}
}

func TestWalkReportsDepthLimitWithoutFollowingDeeperDirectory(t *testing.T) {
	tree := map[string][]model.Item{
		"/":       {{Name: "level1", IsDirectory: true}},
		"/level1": {{Name: "level2", IsDirectory: true}},
	}
	listed := []string{}
	stats, err := Walk(context.Background(), "/", Options{Query: "nomatch", MaxDepth: 1}, func(_ context.Context, directory string) ([]model.Item, error) {
		listed = append(listed, directory)
		return tree[directory], nil
	}, func(base, name string) (string, error) {
		return path.Join(base, name), nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(listed, []string{"/", "/level1"}) || stats.StopReason != StopDepthLimit {
		t.Fatalf("listed=%v stats=%#v", listed, stats)
	}
}

func TestWalkCancellationPropagatesWithoutMoreListings(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	_, err := Walk(ctx, "/", Options{Query: "x"}, func(_ context.Context, _ string) ([]model.Item, error) {
		calls++
		return nil, nil
	}, func(base, name string) (string, error) {
		return path.Join(base, name), nil
	}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
	if calls != 0 {
		t.Fatalf("list called %d times after cancellation", calls)
	}
}

func TestWalkRejectsJoinFailureBeforeTraversalOfChild(t *testing.T) {
	joinErr := errors.New("unsafe child")
	stats, err := Walk(context.Background(), "/", Options{Query: "x"}, func(_ context.Context, _ string) ([]model.Item, error) {
		return []model.Item{{Name: "x", IsDirectory: true}}, nil
	}, func(_, _ string) (string, error) {
		return "", joinErr
	}, nil)
	if !errors.Is(err, joinErr) || stats.Visited != 0 {
		t.Fatalf("err=%v stats=%#v", err, stats)
	}
}
