//go:build windows

package desktop

import "unsafe"

const wmMeasureItem = 0x002C

var fillRectW = user32.NewProc("FillRect")

type measureItemStruct struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemWidth  uint32
	ItemHeight uint32
	ItemData   uintptr
}

func measureItemFromLParam(p uintptr) measureItemStruct {
	var item measureItemStruct
	if p != 0 {
		rtlMoveMemory.Call(uintptr(unsafe.Pointer(&item)), p, unsafe.Sizeof(item))
	}
	return item
}

func measureItemToLParam(p uintptr, item measureItemStruct) {
	if p != 0 {
		rtlMoveMemory.Call(p, uintptr(unsafe.Pointer(&item)), unsafe.Sizeof(item))
	}
}
