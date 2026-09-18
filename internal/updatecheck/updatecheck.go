package updatecheck

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
)

var ErrInvalidCurrentVersion = errors.New("invalid current version")

type Result struct {
	CurrentVersion string
	DisplayVersion string
	UpdateURL      string
	Simulated      bool
}

// Simulate performs a local-only update flow. It deliberately does not query
// any third-party release API. The installed binary version remains the source
// of truth; callers may animate progress and then report a completed simulation.
func Simulate(currentVersion string) (Result, error) {
	current, err := parseVersion(currentVersion)
	if err != nil {
		return Result{}, ErrInvalidCurrentVersion
	}
	value := current.String()
	return Result{
		CurrentVersion: value,
		DisplayVersion: value,
		UpdateURL:      brand.UpdateURL,
		Simulated:      true,
	}, nil
}

type version struct {
	major int
	minor int
	patch int
}

func parseVersion(value string) (version, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return version{}, errors.New("version must contain three numeric parts")
	}
	values := [3]int{}
	for i, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0' && part != "0") {
			return version{}, errors.New("invalid numeric version part")
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || n > 999999 {
			return version{}, errors.New("invalid numeric version part")
		}
		values[i] = n
	}
	return version{major: values[0], minor: values[1], patch: values[2]}, nil
}

func (v version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}
