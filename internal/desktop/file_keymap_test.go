package desktop

import "testing"

func TestFilePaneKeyActionFor(t *testing.T) {
	tests := []struct {
		name     string
		key      uint16
		selected int
		want     filePaneKeyAction
	}{
		{name: "backspace without selection goes up", key: fileKeyBackspace, selected: 0, want: filePaneKeyUp},
		{name: "enter opens one item", key: fileKeyEnter, selected: 1, want: filePaneKeyOpen},
		{name: "enter ignores multiple items", key: fileKeyEnter, selected: 2, want: filePaneKeyNone},
		{name: "f2 renames one item", key: fileKeyF2, selected: 1, want: filePaneKeyRename},
		{name: "f2 ignores no selection", key: fileKeyF2, selected: 0, want: filePaneKeyNone},
		{name: "delete accepts one item", key: fileKeyDelete, selected: 1, want: filePaneKeyDelete},
		{name: "delete accepts multiple items", key: fileKeyDelete, selected: 4, want: filePaneKeyDelete},
		{name: "delete ignores no selection", key: fileKeyDelete, selected: 0, want: filePaneKeyNone},
		{name: "other key ignored", key: 'A', selected: 1, want: filePaneKeyNone},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := filePaneKeyActionFor(tc.key, tc.selected); got != tc.want {
				t.Fatalf("filePaneKeyActionFor(%#x, %d)=%d want %d", tc.key, tc.selected, got, tc.want)
			}
		})
	}
}
