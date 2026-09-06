//go:build windows

package desktop

import (
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/itemlist"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

// LVN_COLUMNCLICK is LVN_FIRST-8. Keep this with the file-pane behavior rather
// than the generic Win32 constants so the sorting contract is easy to audit.
const lvnColumnClick = 0xFFFFFF94

type nmListView struct {
	Hdr       nmhdr
	Item      int32
	SubItem   int32
	NewState  uint32
	OldState  uint32
	Changed   uint32
	PtAction  point
	ItemParam uintptr
}

func nmListViewFromLParam(p uintptr) nmListView {
	var n nmListView
	if p != 0 {
		rtlMoveMemory.Call(uintptr(unsafe.Pointer(&n)), p, unsafe.Sizeof(n))
	}
	return n
}

func fileSortField(column int, remote bool) (itemlist.Field, bool) {
	switch column {
	case 0:
		return itemlist.FieldName, true
	case 1:
		return itemlist.FieldType, true
	case 2:
		return itemlist.FieldSize, true
	case 3:
		return itemlist.FieldModified, true
	case 4:
		if remote {
			return itemlist.FieldPermissions, true
		}
	}
	return itemlist.FieldName, false
}

func (a *app) fileSortState(list uintptr) (column int, descending bool, ok bool) {
	if a == nil {
		return 0, false, false
	}
	switch list {
	case a.localList:
		return a.localSortColumn, a.localSortDescending, true
	case a.remoteList:
		return a.remoteSortColumn, a.remoteSortDescending, true
	default:
		return 0, false, false
	}
}

func (a *app) sortFileItems(list uintptr, items []model.Item) {
	column, descending, ok := a.fileSortState(list)
	if !ok {
		return
	}
	field, valid := fileSortField(column, list == a.remoteList)
	if !valid {
		return
	}
	itemlist.SortBy(items, itemlist.Spec{Field: field, Descending: descending})
}

func (a *app) handleFileColumnClick(list uintptr, column int) {
	if a == nil || (list != a.localList && list != a.remoteList) {
		return
	}
	if _, valid := fileSortField(column, list == a.remoteList); !valid {
		return
	}

	var items []model.Item
	if list == a.localList {
		items = a.localItems
		if column == a.localSortColumn {
			a.localSortDescending = !a.localSortDescending
		} else {
			a.localSortColumn = column
			a.localSortDescending = false
		}
	} else {
		items = a.remoteItems
		if column == a.remoteSortColumn {
			a.remoteSortDescending = !a.remoteSortDescending
		} else {
			a.remoteSortColumn = column
			a.remoteSortDescending = false
		}
	}

	selected := selectedItemNames(list, items)
	a.sortFileItems(list, items)
	a.fillItemList(list, items)
	restoreItemSelection(list, items, selected)
	if header := headerForList(list); header != 0 {
		invalidateRect.Call(header, 0, 1)
	}
}

// fileSortDirectionForHeader returns the active direction only for Local and
// Server file headers. Transfer-queue headers intentionally remain unsorted.
func (a *app) fileSortDirectionForHeader(header uintptr, column int) (descending bool, active bool) {
	if a == nil || header == 0 {
		return false, false
	}
	if header == headerForList(a.localList) {
		if column == a.localSortColumn {
			_, valid := fileSortField(column, false)
			return a.localSortDescending, valid
		}
		return false, false
	}
	if header == headerForList(a.remoteList) {
		if column == a.remoteSortColumn {
			_, valid := fileSortField(column, true)
			return a.remoteSortDescending, valid
		}
	}
	return false, false
}
