//go:build windows

package profilebinding

import "strings"

func privateKeyPathEqual(a, b string) bool {
	return strings.EqualFold(a, b)
}
