package desktop

const navigationHistoryLimit = 128

type pathHistory struct {
	entries []string
	index   int
}

func (h *pathHistory) reset() {
	h.entries = nil
	h.index = -1
}

func (h *pathHistory) current() (string, bool) {
	if h == nil || h.index < 0 || h.index >= len(h.entries) {
		return "", false
	}
	return h.entries[h.index], true
}

func (h *pathHistory) commit(path string) {
	if h == nil || path == "" {
		return
	}
	if current, ok := h.current(); ok && current == path {
		return
	}
	if h.index >= 0 && h.index+1 < len(h.entries) {
		h.entries = append([]string(nil), h.entries[:h.index+1]...)
	}
	h.entries = append(h.entries, path)
	if len(h.entries) > navigationHistoryLimit {
		drop := len(h.entries) - navigationHistoryLimit
		h.entries = append([]string(nil), h.entries[drop:]...)
	}
	h.index = len(h.entries) - 1
}

func (h *pathHistory) backTarget() (string, int, bool) {
	if h == nil || h.index <= 0 || h.index > len(h.entries)-1 {
		return "", -1, false
	}
	index := h.index - 1
	return h.entries[index], index, true
}

func (h *pathHistory) forwardTarget() (string, int, bool) {
	if h == nil || h.index < 0 || h.index+1 >= len(h.entries) {
		return "", -1, false
	}
	index := h.index + 1
	return h.entries[index], index, true
}

func (h *pathHistory) selectResolved(index int, resolved string) bool {
	if h == nil || index < 0 || index >= len(h.entries) || resolved == "" {
		return false
	}
	h.entries[index] = resolved
	h.index = index
	return true
}

func (h *pathHistory) canBack() bool {
	_, _, ok := h.backTarget()
	return ok
}

func (h *pathHistory) canForward() bool {
	_, _, ok := h.forwardTarget()
	return ok
}
