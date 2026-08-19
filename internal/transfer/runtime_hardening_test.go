package transfer

import "testing"

func TestSelectedIDsTrimsAndDeduplicates(t *testing.T) {
	got, err := selectedIDs([]string{"  one  ", "two", "one"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("selected ID count=%d want 2", len(got))
	}
	if _, ok := got["one"]; !ok {
		t.Fatal("trimmed ID one missing")
	}
	if _, ok := got["two"]; !ok {
		t.Fatal("ID two missing")
	}
}

func TestSelectedIDsRejectsInvalidSelection(t *testing.T) {
	if _, err := selectedIDs(nil); err == nil {
		t.Fatal("empty selection should fail")
	}
	if _, err := selectedIDs([]string{"ok", "   "}); err == nil {
		t.Fatal("blank transfer ID should fail")
	}
}

func TestWaitWorkersAcceptsNilContext(t *testing.T) {
	m := &Manager{}
	if err := m.waitWorkers(nil); err != nil {
		t.Fatalf("nil-context wait failed: %v", err)
	}
}
