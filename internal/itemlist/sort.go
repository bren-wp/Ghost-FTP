package itemlist

import (
	"sort"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

// Sort orders directories before files and then compares names
// case-insensitively while preserving the original order of equal folded names.
// The folded key is computed once per item so large 50k-entry directory views do
// not allocate lowercase copies for every sort comparison.
func Sort(items []model.Item) {
	if len(items) < 2 {
		return
	}
	type keyedItem struct {
		item model.Item
		key  string
	}
	keyed := make([]keyedItem, len(items))
	for i := range items {
		keyed[i] = keyedItem{item: items[i], key: strings.ToLower(items[i].Name)}
	}
	sort.SliceStable(keyed, func(i, j int) bool {
		if keyed[i].item.IsDirectory != keyed[j].item.IsDirectory {
			return keyed[i].item.IsDirectory
		}
		return keyed[i].key < keyed[j].key
	})
	for i := range keyed {
		items[i] = keyed[i].item
	}
}
