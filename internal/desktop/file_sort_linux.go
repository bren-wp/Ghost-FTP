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

func (u *linuxDesktop) linuxFileHeaderField(remote bool, x int) (itemlist.Field, bool) {
	r := u.fileFilterListRect(remote)
	if x < r.left || x >= r.right {
		return itemlist.FieldName, false
	}
	width := r.right - r.left
	sizeX := r.left + width*54/100
	modifiedX := r.left + width*70/100
	permissionsX := r.left + width*88/100
	if remote {
		sizeX = r.left + width*48/100
		modifiedX = r.left + width*64/100
		permissionsX = r.left + width*84/100
	}
	switch {
	case x < sizeX:
		return itemlist.FieldName, true
	case x < modifiedX:
		return itemlist.FieldSize, true
	case !remote || x < permissionsX:
		return itemlist.FieldModified, true
	default:
		return itemlist.FieldPermissions, true
	}
}

func (u *linuxDesktop) setLinuxFileSortField(remote bool, field itemlist.Field) {
	if u == nil || u.busy || u.directoryComparisonActiveLinux() || u.recursiveSearchActive(false) || u.recursiveSearchActive(true) {
		return
	}
	if remote && !u.connected {
		return
	}
	state := linuxFileSortStateFor(u)
	specs := linuxFileSortSpecs(remote)
	current := u.linuxFileSortSpec(remote)
	descending := false
	if current.Field == field {
		descending = !current.Descending
	}
	target := 0
	for index, spec := range specs {
		if spec.Field == field && spec.Descending == descending {
			target = index
			break
		}
	}
	if remote {
		state.remoteIndex = target
	} else {
		state.localIndex = target
	}
	filter := u.fileFilterState()
	if filter == nil {
		return
	}
	if remote {
		selectedName := selectedLinuxItemName(u.remoteItems, u.selectedRemote)
		u.remoteItems = u.sortLinuxFileItems(true, itemlist.Filter(filter.remoteAll, filter.remoteQuery))
		u.selectedRemote = restoreLinuxSelection(u.remoteItems, selectedName)
		u.setStatus(u.tr("section.remote") + " · " + u.linuxFileSortLabel(true))
		return
	}
	selectedName := selectedLinuxItemName(u.localItems, u.selectedLocal)
	u.localItems = u.sortLinuxFileItems(false, itemlist.Filter(filter.localAll, filter.localQuery))
	u.selectedLocal = restoreLinuxSelection(u.localItems, selectedName)
	u.setStatus(u.tr("section.local") + " · " + u.linuxFileSortLabel(false))
}

func (u *linuxDesktop) handleLinuxFileSortHeaderMouse(x, y int) bool {
	for _, remote := range []bool{false, true} {
		r := u.fileFilterListRect(remote)
		header := linuxRect{left: r.left, top: r.top, right: r.right, bottom: min(r.bottom, r.top+28)}
		if !header.contains(x, y) {
			continue
		}
		field, ok := u.linuxFileHeaderField(remote, x)
		if !ok {
			return true
		}
		u.setLinuxFileSortField(remote, field)
		return true
	}
	return false
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
