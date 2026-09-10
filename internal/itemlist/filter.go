package itemlist

import (
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

// Filter returns a new slice containing items whose names match every
// whitespace-delimited query token, case-insensitively. It never mutates the
// source slice, which allows desktop panes to keep one authoritative directory
// snapshot while sorting and rendering an independent visible view.
//
// The filter is intentionally scoped to the currently loaded directory. It
// performs no filesystem or network I/O and therefore cannot turn typing into
// hidden recursive scans or extra server requests.
func Filter(items []model.Item, query string) []model.Item {
	tokens := filterTokens(query)
	if len(items) == 0 {
		return nil
	}

	out := make([]model.Item, 0, len(items))
	if len(tokens) == 0 {
		return append(out, items...)
	}

	for _, item := range items {
		name := strings.ToLower(item.Name)
		matched := true
		for _, token := range tokens {
			if !strings.Contains(name, token) {
				matched = false
				break
			}
		}
		if matched {
			out = append(out, item)
		}
	}
	return out
}

func filterTokens(query string) []string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(fields) == 0 {
		return nil
	}
	return fields
}
