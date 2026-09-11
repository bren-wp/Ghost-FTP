//go:build linux

package desktop

import (
	"fmt"
	"sync"

	"github.com/bren-wp/Ghost-FTP/internal/itemlist"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

type linuxFileSortState struct {
	localIndex  int
	remoteIndex int
}

var linuxFileSortStates sync.Map

func linuxFileSortStateFor(u *linuxDesktop) *linuxFileSortState {
	if value, ok := linuxFileSortStates.Load(u); ok {
		return value.(*linuxFileSortState)
	}
	state := &linuxFileSortState{}
	actual, _ := linuxFileSortStates.LoadOrStore(u, state)
	return actual.(*linuxFileSortState)
}

func linuxFileSortSpecs(remote bool) []itemlist.Spec {
	specs := []itemlist.Spec{
		{Field: itemlist.FieldName},
		{Field: itemlist.FieldName, Descending: true},
		{Field: itemlist.FieldType},
		{Field: itemlist.FieldType, Descending: true},
		{Field: itemlist.FieldSize},
		{Field: itemlist.FieldSize, Descending: true},
		{Field: itemlist.FieldModified},
		{Field: itemlist.FieldModified, Descending: true},
	}
	if remote {
		specs = append(specs,
			itemlist.Spec{Field: itemlist.FieldPermissions},
			itemlist.Spec{Field: itemlist.FieldPermissions, Descending: true},
		)
	}
	return specs
}

func (u *linuxDesktop) linuxFileSortSpec(remote bool) itemlist.Spec {
	state := linuxFileSortStateFor(u)
	index := state.localIndex
	if remote {
		index = state.remoteIndex
	}
	specs := linuxFileSortSpecs(remote)
	if index < 0 || index >= len(specs) {
		return specs[0]
	}
	return specs[index]
}

func (u *linuxDesktop) sortLinuxFileItems(remote bool, items []model.Item) []model.Item {
	out := append([]model.Item(nil), items...)
	itemlist.SortBy(out, u.linuxFileSortSpec(remote))
	return out
}

func (u *linuxDesktop) fileFilterPromptControlRect(remote bool) linuxRect {
	full := u.fileFilterControlRect(remote)
	gap := 8
	sortWidth := min(118, (full.right-full.left-gap)/2)
	return linuxRectWH(full.left, full.top, full.right-full.left-sortWidth-gap, full.bottom-full.top)
}

func (u *linuxDesktop) fileSortControlRect(remote bool) linuxRect {
	full := u.fileFilterControlRect(remote)
	gap := 8
	sortWidth := min(118, (full.right-full.left-gap)/2)
	return linuxRectWH(full.right-sortWidth, full.top, sortWidth, full.bottom-full.top)
}

func (u *linuxDesktop) linuxFileSortLabel(remote bool) string {
	spec := u.linuxFileSortSpec(remote)
	label := u.tr("column.name")
	switch spec.Field {
	case itemlist.FieldType:
		label = u.tr("column.type")
	case itemlist.FieldSize:
		label = u.tr("column.size")
	case itemlist.FieldModified:
		label = u.tr("column.modified")
	case itemlist.FieldPermissions:
		label = u.tr("common.permissions")
	}
	arrow := "↑"
	if spec.Descending {
		arrow = "↓"
	}
	return fmt.Sprintf("%s %s", label, arrow)
}

func (u *linuxDesktop) cycleLinuxFileSort(remote bool) {
	if u == nil || u.busy || u.directoryComparisonActiveLinux() || u.recursiveSearchActive(false) || u.recursiveSearchActive(true) {
		return
	}
	if remote && !u.connected {
		return
	}

	selectedName := selectedLinuxItemName(u.localItems, u.selectedLocal)
	if remote {
		selectedName = selectedLinuxItemName(u.remoteItems, u.selectedRemote)
	}

	state := linuxFileSortStateFor(u)
	specs := linuxFileSortSpecs(remote)
	if remote {
		state.remoteIndex = (state.remoteIndex + 1) % len(specs)
	} else {
		state.localIndex = (state.localIndex + 1) % len(specs)
	}

	filter := u.fileFilterState()
	if filter == nil {
		return
	}
	if remote {
		u.remoteItems = u.sortLinuxFileItems(true, itemlist.Filter(filter.remoteAll, filter.remoteQuery))
		u.selectedRemote = restoreLinuxSelection(u.remoteItems, selectedName)
		u.setStatus(u.tr("section.remote") + " · " + u.linuxFileSortLabel(true))
		return
	}
	u.localItems = u.sortLinuxFileItems(false, itemlist.Filter(filter.localAll, filter.localQuery))
	u.selectedLocal = restoreLinuxSelection(u.localItems, selectedName)
	u.setStatus(u.tr("section.local") + " · " + u.linuxFileSortLabel(false))
}
