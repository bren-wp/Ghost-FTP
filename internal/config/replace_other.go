//go:build !windows

package config

import "os"

func replaceFile(src, dst string) error { return os.Rename(src, dst) }
