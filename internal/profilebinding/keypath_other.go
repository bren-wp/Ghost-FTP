//go:build !windows

package profilebinding

func privateKeyPathEqual(a, b string) bool {
	return a == b
}
