package itemlist

import (
	"strings"
	"unicode"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

// Filter returns a new slice containing items whose names match every
// whitespace-delimited query token with Unicode-aware case folding. It never
// mutates the source slice, which allows desktop panes to keep one authoritative
// directory snapshot while sorting and rendering an independent visible view.
//
// The filter is intentionally scoped to the currently loaded directory. It
// performs no filesystem or network I/O and therefore cannot turn typing into
// hidden recursive scans or extra server requests.
func Filter(items []model.Item, query string) []model.Item {
	if len(items) == 0 {
		return nil
	}

	out := make([]model.Item, 0, len(items))
	if len(filterTokens(query)) == 0 {
		return append(out, items...)
	}

	for _, item := range items {
		if MatchesName(item.Name, query) {
			out = append(out, item)
		}
	}
	return out
}

// MatchesName applies the same Unicode-aware, whitespace-token AND semantics
// used by the current-folder filter. Recursive search reuses this helper so the
// two user-facing search surfaces cannot drift to different case behavior.
func MatchesName(name, query string) bool {
	tokens := filterTokens(query)
	if len(tokens) == 0 {
		return true
	}
	for _, token := range tokens {
		if !containsFold(name, token) {
			return false
		}
	}
	return true
}

func filterTokens(query string) []string {
	fields := strings.Fields(strings.TrimSpace(query))
	if len(fields) == 0 {
		return nil
	}
	return fields
}

// containsFold reports whether token occurs in value under Unicode simple case
// folding. unicode.SimpleFold handles case-fold orbits that lowercasing alone
// cannot represent correctly, such as Greek sigma (Σ, σ, ς).
func containsFold(value, token string) bool {
	if token == "" {
		return true
	}
	valueRunes := []rune(value)
	tokenRunes := []rune(token)
	if len(tokenRunes) > len(valueRunes) {
		return false
	}
	for start := 0; start+len(tokenRunes) <= len(valueRunes); start++ {
		matched := true
		for offset, want := range tokenRunes {
			if !equalFoldRune(valueRunes[start+offset], want) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

func equalFoldRune(left, right rune) bool {
	if left == right {
		return true
	}
	for folded := unicode.SimpleFold(left); folded != left; folded = unicode.SimpleFold(folded) {
		if folded == right {
			return true
		}
	}
	return false
}
