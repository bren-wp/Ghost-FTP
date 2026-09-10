//go:build windows

package platform

import "testing"

func TestSettingsNumberInputLengthFollowsBounds(t *testing.T) {
	cases := []struct {
		field SettingsDialogNumber
		want  uintptr
	}{
		{field: SettingsDialogNumber{Min: 0, Max: 8}, want: 1},
		{field: SettingsDialogNumber{Min: 0, Max: 60}, want: 2},
		{field: SettingsDialogNumber{Min: 0, Max: 1048576}, want: 7},
		{field: SettingsDialogNumber{Min: -30, Max: 30}, want: 3},
	}
	for _, tc := range cases {
		if got := settingsNumberInputLength(tc.field); got != tc.want {
			t.Fatalf("settingsNumberInputLength(%+v) = %d, want %d", tc.field, got, tc.want)
		}
	}
}
