package desktop

const (
	fileKeyBackspace uint16 = 0x08
	fileKeyEnter     uint16 = 0x0D
	fileKeyDelete    uint16 = 0x2E
	fileKeyF2        uint16 = 0x71
)

type filePaneKeyAction uint8

const (
	filePaneKeyNone filePaneKeyAction = iota
	filePaneKeyOpen
	filePaneKeyRename
	filePaneKeyDelete
	filePaneKeyUp
)

// filePaneKeyActionFor keeps destructive and mutation shortcuts dependent on
// an explicit ListView selection. Enter/F2 require exactly one item because
// opening or renaming an arbitrary item from a multi-selection is ambiguous.
func filePaneKeyActionFor(vKey uint16, selected int) filePaneKeyAction {
	switch vKey {
	case fileKeyBackspace:
		return filePaneKeyUp
	case fileKeyEnter:
		if selected == 1 {
			return filePaneKeyOpen
		}
	case fileKeyF2:
		if selected == 1 {
			return filePaneKeyRename
		}
	case fileKeyDelete:
		if selected > 0 {
			return filePaneKeyDelete
		}
	}
	return filePaneKeyNone
}
