package directorycompare

import (
	"errors"
	"sort"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

type Status string

const (
	StatusSame        Status = "same"
	StatusLocalOnly   Status = "local_only"
	StatusRemoteOnly  Status = "remote_only"
	StatusNewerLocal  Status = "newer_local"
	StatusNewerRemote Status = "newer_remote"
	StatusConflict    Status = "conflict"
	StatusUnknown     Status = "unknown"

	DefaultTimestampTolerance = 2 * time.Second
	MaxTimestampTolerance     = 5 * time.Minute
)

type Options struct {
	TimestampTolerance time.Duration
}

type Entry struct {
	Name      string     `json:"name"`
	Status    Status     `json:"status"`
	Local     model.Item `json:"local"`
	Remote    model.Item `json:"remote"`
	HasLocal  bool       `json:"hasLocal"`
	HasRemote bool       `json:"hasRemote"`
}

func NormalizeOptions(opts Options) (Options, error) {
	if opts.TimestampTolerance == 0 {
		opts.TimestampTolerance = DefaultTimestampTolerance
	}
	if opts.TimestampTolerance < 0 || opts.TimestampTolerance > MaxTimestampTolerance {
		return Options{}, errors.New("directory comparison timestamp tolerance is outside the safe range")
	}
	return opts, nil
}

func Compare(local, remote []model.Item, opts Options) ([]Entry, error) {
	opts, err := NormalizeOptions(opts)
	if err != nil {
		return nil, err
	}
	localByName := groupByExactName(local)
	remoteByName := groupByExactName(remote)
	names := make([]string, 0, len(localByName)+len(remoteByName))
	seen := make(map[string]struct{}, len(localByName)+len(remoteByName))
	for name := range localByName {
		seen[name] = struct{}{}
		names = append(names, name)
	}
	for name := range remoteByName {
		if _, ok := seen[name]; ok {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	entries := make([]Entry, 0, len(names))
	for _, name := range names {
		locals := localByName[name]
		remotes := remoteByName[name]
		entry := Entry{Name: name}
		if len(locals) > 0 {
			entry.Local = locals[0]
			entry.HasLocal = true
		}
		if len(remotes) > 0 {
			entry.Remote = remotes[0]
			entry.HasRemote = true
		}
		switch {
		case len(locals) > 1 || len(remotes) > 1:
			entry.Status = StatusConflict
		case len(locals) == 0:
			entry.Status = StatusRemoteOnly
		case len(remotes) == 0:
			entry.Status = StatusLocalOnly
		default:
			entry.Status = classifyPair(locals[0], remotes[0], opts.TimestampTolerance)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func groupByExactName(items []model.Item) map[string][]model.Item {
	grouped := make(map[string][]model.Item, len(items))
	for _, item := range items {
		grouped[item.Name] = append(grouped[item.Name], item)
	}
	return grouped
}

func classifyPair(local, remote model.Item, tolerance time.Duration) Status {
	if local.IsSymlink || remote.IsSymlink {
		return StatusUnknown
	}
	if local.IsDirectory != remote.IsDirectory {
		return StatusConflict
	}
	if local.IsDirectory {
		return StatusSame
	}

	localTimeKnown := !local.Modified.IsZero()
	remoteTimeKnown := !remote.Modified.IsZero()
	if !localTimeKnown || !remoteTimeKnown {
		if local.Size != remote.Size {
			return StatusConflict
		}
		return StatusUnknown
	}

	// time.Time.Sub saturates at the duration limits for very distant dates.
	// Always subtract newer from older directionally so no absolute-value
	// negation can overflow the minimum duration and falsely look "within"
	// tolerance.
	if local.Modified.After(remote.Modified) {
		if local.Modified.Sub(remote.Modified) > tolerance {
			return StatusNewerLocal
		}
	} else if remote.Modified.After(local.Modified) {
		if remote.Modified.Sub(local.Modified) > tolerance {
			return StatusNewerRemote
		}
	}

	if local.Size != remote.Size {
		return StatusConflict
	}
	// model.Item currently cannot distinguish a verified zero-byte remote file
	// from a listing whose size fact was absent/malformed and therefore decoded
	// to the zero value. Do not let that ambiguity manufacture StatusSame.
	if local.Size == 0 {
		return StatusUnknown
	}
	return StatusSame
}
