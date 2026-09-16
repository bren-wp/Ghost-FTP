//go:build windows

package main

import "testing"

func TestShouldRemoveProtocolAfterUninstall(t *testing.T) {
	tests := []struct {
		name                                             string
		exitCode                                         int
		interactive, beforeKnown, beforeExists           bool
		afterKnown, afterExists                          bool
		want                                             bool
	}{
		{"successful uninstall", 0, true, true, true, true, false, true},
		{"cancel keeps uninstall entry", 0, true, true, true, true, true, false},
		{"preexisting missing entry", 0, true, true, false, true, false, false},
		{"before state unknown", 0, true, false, false, true, false, false},
		{"after state unknown", 0, true, true, true, false, false, false},
		{"uninstall failed", 1, true, true, true, true, false, false},
		{"not interactive", 0, false, true, true, true, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRemoveProtocolAfterUninstall(tt.exitCode, tt.interactive, tt.beforeKnown, tt.beforeExists, tt.afterKnown, tt.afterExists)
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
