//go:build windows

package main

import "testing"

func TestShouldRemoveOwnedProtocolHandlerRequiresStartedUninstall(t *testing.T) {
	cases := []struct {
		name              string
		exitCode          int
		cleanupAuthorized bool
		want              bool
	}{
		{name: "confirmed uninstall", exitCode: 0, cleanupAuthorized: true, want: true},
		{name: "cancelled uninstall", exitCode: 0, cleanupAuthorized: false, want: false},
		{name: "failed uninstall", exitCode: 1, cleanupAuthorized: true, want: false},
		{name: "failed before authorization", exitCode: 1, cleanupAuthorized: false, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRemoveOwnedProtocolHandler(tc.exitCode, tc.cleanupAuthorized); got != tc.want {
				t.Fatalf("shouldRemoveOwnedProtocolHandler(%d, %v) = %v, want %v", tc.exitCode, tc.cleanupAuthorized, got, tc.want)
			}
		})
	}
}
