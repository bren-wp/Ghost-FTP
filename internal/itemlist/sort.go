package itemlist

import (
	"sort"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

type Field uint8

const (
	FieldName Field = iota
	FieldType
	FieldSize
	FieldModified
	FieldPermissions
)

type Spec struct {
	Field      Field
	Descending bool
}

type keyedItem struct {
	item           model.Item
	nameKey        string
	typeKey        string
	permissionsKey string
}

// Sort preserves the historical default ordering: directories first, then
// names case-insensitively in ascending order. Equal folded names keep their
// original order.
func Sort(items []model.Item) {
	SortBy(items, Spec{Field: FieldName})
}

// SortBy provides deterministic file-pane sorting without allowing descending
// order to push directories below files. Expensive folded string keys are
// computed once per item so large 50k-entry directory views do not allocate in
// every comparison.
func SortBy(items []model.Item, spec Spec) {
	if len(items) < 2 {
		return
	}
	if spec.Field > FieldPermissions {
		spec.Field = FieldName
	}

	keyed := make([]keyedItem, len(items))
	for i := range items {
		keyed[i] = keyedItem{
			item:           items[i],
			nameKey:        strings.ToLower(items[i].Name),
			typeKey:        itemTypeKey(items[i]),
			permissionsKey: strings.ToLower(strings.TrimSpace(items[i].Permissions)),
		}
	}

	sort.SliceStable(keyed, func(i, j int) bool {
		a, b := keyed[i], keyed[j]
		if a.item.IsDirectory != b.item.IsDirectory {
			return a.item.IsDirectory
		}

		cmp, reversible := comparePrimary(a, b, spec.Field)
		if cmp != 0 {
			if reversible && spec.Descending {
				return cmp > 0
			}
			return cmp < 0
		}

		// Name sorting intentionally remains stable for equal folded names. For
		// every other field, a case-insensitive name tie-break keeps the view
		// deterministic without making equal-name entries jump around.
		if spec.Field == FieldName || a.nameKey == b.nameKey {
			return false
		}
		return a.nameKey < b.nameKey
	})

	for i := range keyed {
		items[i] = keyed[i].item
	}
}

// comparePrimary returns a comparison and whether descending order may reverse
// it. Metadata gaps use a non-reversible comparison so unknown timestamps or
// permissions remain at the bottom in both directions.
func comparePrimary(a, b keyedItem, field Field) (cmp int, reversible bool) {
	switch field {
	case FieldType:
		return compareString(a.typeKey, b.typeKey), true
	case FieldSize:
		return compareInt64(a.item.Size, b.item.Size), true
	case FieldModified:
		aZero, bZero := a.item.Modified.IsZero(), b.item.Modified.IsZero()
		if aZero != bZero {
			if aZero {
				return 1, false
			}
			return -1, false
		}
		if aZero {
			return 0, false
		}
		if a.item.Modified.Before(b.item.Modified) {
			return -1, true
		}
		if a.item.Modified.After(b.item.Modified) {
			return 1, true
		}
		return 0, true
	case FieldPermissions:
		aEmpty, bEmpty := a.permissionsKey == "", b.permissionsKey == ""
		if aEmpty != bEmpty {
			if aEmpty {
				return 1, false
			}
			return -1, false
		}
		return compareString(a.permissionsKey, b.permissionsKey), true
	default:
		return compareString(a.nameKey, b.nameKey), true
	}
}

func compareString(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareInt64(a, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func itemTypeKey(item model.Item) string {
	if item.IsDirectory {
		return "folder"
	}
	if item.IsSymlink {
		return "link"
	}
	name := strings.ToLower(item.Name)
	if dot := strings.LastIndexByte(name, '.'); dot >= 0 && dot+1 < len(name) {
		return name[dot+1:]
	}
	return "file"
}
