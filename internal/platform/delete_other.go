//go:build !windows

package platform

func ScheduleDeleteOnReboot(string) error { return nil }
