package desktop

import (
	"fmt"
	"testing"
)

func TestPathHistoryBackForwardAndBranching(t *testing.T) {
	var h pathHistory
	h.reset()
	h.commit("a")
	h.commit("b")
	h.commit("c")

	target, index, ok := h.backTarget()
	if !ok || target != "b" {
		t.Fatalf("back target = %q, %v", target, ok)
	}
	if !h.selectResolved(index, "b") {
		t.Fatal("select back target failed")
	}
	target, index, ok = h.forwardTarget()
	if !ok || target != "c" {
		t.Fatalf("forward target = %q, %v", target, ok)
	}
	if !h.selectResolved(index, "c") {
		t.Fatal("select forward target failed")
	}

	target, index, ok = h.backTarget()
	if !ok || target != "b" {
		t.Fatal("second back target missing")
	}
	h.selectResolved(index, "b")
	h.commit("d")
	if h.canForward() {
		t.Fatal("new navigation must discard stale forward history")
	}
	if current, ok := h.current(); !ok || current != "d" {
		t.Fatalf("current = %q, %v", current, ok)
	}
}

func TestPathHistoryDeduplicatesAndBounds(t *testing.T) {
	var h pathHistory
	h.reset()
	h.commit("same")
	h.commit("same")
	if len(h.entries) != 1 {
		t.Fatalf("duplicate path stored: %v", h.entries)
	}
	for i := 0; i < navigationHistoryLimit+16; i++ {
		h.commit(fmt.Sprintf("p-%03d", i))
	}
	if len(h.entries) != navigationHistoryLimit {
		t.Fatalf("history length = %d", len(h.entries))
	}
	if h.index != len(h.entries)-1 {
		t.Fatalf("history index = %d", h.index)
	}
}
