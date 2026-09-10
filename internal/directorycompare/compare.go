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
	Name      string
	Status    Status
	Local     model.Item
	Remote    model.Item
	HasLocal  bool
	HasRemote bool
}

func NormalizeOptions(in Options) (Options, error) {
	if in.TimestampTolerance < 0 {
		return Options{}, errors.New("directory comparison timestamp tolerance must not be negative")
	}
	if in.TimestampTolerance == 0 {
		in.TimestampTolerance = DefaultTimestampTolerance
	}
	if in.TimestampTolerance > MaxTimestampTolerance {
		return Options{}, errors.New("directory comparison timestamp tolerance is too large")
	}
	return in, nil
}

// Compare classifies one local and one remote directory snapshot without
// performing I/O or mutating either input. Names are matched exactly: silently
// folding case would be unsafe when one side is case-sensitive and can contain
// both "File" and "file".
func Compare(local, remote []model.Item, in Options) ([]Entry, error) {
	opts, err := NormalizeOptions(in)
	if err != nil {
		return nil, err
	}

	localByName := groupByName(local)
	remoteByName := groupByName(remote)
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

	out := make([]Entry, 0, len(names))
	for _, name := range names {
		locals := localByName[name]
		remotes := remoteByName[name]
		entry := Entry{Name: name, HasLocal: len(locals) > 0, HasRemote: len(remotes) > 0}
		if len(locals) > 0 {
			entry.Local = locals[0]
		}
		if len(remotes) > 0 {
			entry.Remote = remotes[0]
		}

		switch {
		case len(locals) > 1 || len(remotes) > 1:
			entry.Status = StatusConflict
		case len(locals) == 0:
			if entry.Remote.IsSymlink {
				entry.Status = StatusUnknown
			} else {
				entry.Status = StatusRemoteOnly
			}
		case len(remotes) == 0:
			if entry.Local.IsSymlink {
				entry.Status = StatusUnknown
			} else {
				entry.Status = StatusLocalOnly
			}
		default:
			entry.Status = classifyPair(entry.Local, entry.Remote, opts.TimestampTolerance)
		}
		out = append(out, entry)
	}
	return out, nil
}

func groupByName(items []model.Item) map[string][]model.Item {
	groups := make(map[string][]model.Item, len(items))
	for _, item := range items {
		groups[item.Name] = append(groups[item.Name], item)
	}
	return groups
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

	delta := local.Modified.Sub(remote.Modified)
	if delta < 0 {
		delta = -delta
	}
	if delta <= tolerance {
		if local.Size == remote.Size {
			return StatusSame
		}
		return StatusConflict
	}
	if local.Modified.After(remote.Modified) {
		return StatusNewerLocal
	}
	return StatusNewerRemote
}
