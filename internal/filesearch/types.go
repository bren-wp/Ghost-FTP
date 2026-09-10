package filesearch

import (
	"errors"
	"strings"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

const (
	DefaultMaxDepth   = 12
	HardMaxDepth      = 32
	DefaultMaxVisited = 20000
	HardMaxVisited    = 50000
	DefaultMaxResults = 1000
	HardMaxResults    = 5000
	DefaultBatchSize  = 50
	HardMaxBatchSize  = 200
	DefaultTimeout    = 20 * time.Second
	HardMaxTimeout    = 60 * time.Second
)

const (
	StopNone        = ""
	StopDepthLimit  = "depth_limit"
	StopItemLimit   = "item_limit"
	StopResultLimit = "result_limit"
	StopTimeLimit   = "time_limit"
)

var ErrEmptyQuery = errors.New("recursive search query must not be empty")

type Options struct {
	Query      string
	MaxDepth   int
	MaxVisited int
	MaxResults int
	BatchSize  int
	Timeout    time.Duration
}

type Result struct {
	Path   string
	Parent string
	Depth  int
	Item   model.Item
}

type Stats struct {
	Visited     int
	Directories int
	Results     int
	StopReason  string
}

type EmitFunc func([]Result) error

type ListFunc func(path string) ([]model.Item, error)
type JoinFunc func(base, name string) (string, error)

func NormalizeOptions(in Options) (Options, error) {
	in.Query = strings.TrimSpace(in.Query)
	if in.Query == "" {
		return Options{}, ErrEmptyQuery
	}
	if in.MaxDepth == 0 {
		in.MaxDepth = DefaultMaxDepth
	}
	if in.MaxVisited == 0 {
		in.MaxVisited = DefaultMaxVisited
	}
	if in.MaxResults == 0 {
		in.MaxResults = DefaultMaxResults
	}
	if in.BatchSize == 0 {
		in.BatchSize = DefaultBatchSize
	}
	if in.Timeout == 0 {
		in.Timeout = DefaultTimeout
	}
	if in.MaxDepth < 1 || in.MaxDepth > HardMaxDepth {
		return Options{}, errors.New("recursive search depth limit is outside the safe range")
	}
	if in.MaxVisited < 1 || in.MaxVisited > HardMaxVisited {
		return Options{}, errors.New("recursive search item limit is outside the safe range")
	}
	if in.MaxResults < 1 || in.MaxResults > HardMaxResults {
		return Options{}, errors.New("recursive search result limit is outside the safe range")
	}
	if in.BatchSize < 1 || in.BatchSize > HardMaxBatchSize {
		return Options{}, errors.New("recursive search batch size is outside the safe range")
	}
	if in.Timeout < time.Second || in.Timeout > HardMaxTimeout {
		return Options{}, errors.New("recursive search timeout is outside the safe range")
	}
	return in, nil
}
