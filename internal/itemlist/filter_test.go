package itemlist

import (
	"reflect"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestFilterEmptyQueryReturnsIndependentCopy(t *testing.T) {
	source := []model.Item{{Name: "alpha.txt"}, {Name: "beta.txt"}}
	visible := Filter(source, "   ")
	if !reflect.DeepEqual(visible, source) {
		t.Fatalf("visible=%v source=%v", visible, source)
	}
	visible[0].Name = "changed.txt"
	if source[0].Name != "alpha.txt" {
		t.Fatal("filter result aliases source storage")
	}
}

func TestFilterMatchesCaseInsensitiveSubstring(t *testing.T) {
	source := []model.Item{{Name: "Quarterly-Report.PDF"}, {Name: "notes.txt"}}
	got := Filter(source, "report")
	if len(got) != 1 || got[0].Name != "Quarterly-Report.PDF" {
		t.Fatalf("got=%v", got)
	}
}

func TestFilterRequiresEveryWhitespaceToken(t *testing.T) {
	source := []model.Item{
		{Name: "client invoice 2026.pdf"},
		{Name: "client notes 2026.txt"},
		{Name: "invoice archive.pdf"},
	}
	got := Filter(source, "  CLIENT   2026 invoice ")
	if len(got) != 1 || got[0].Name != "client invoice 2026.pdf" {
		t.Fatalf("got=%v", got)
	}
}

func TestFilterHandlesUnicodeCaseConversion(t *testing.T) {
	source := []model.Item{{Name: "RAČUN-Rujan.pdf"}, {Name: "izvještaj.txt"}}
	got := Filter(source, "račun")
	if len(got) != 1 || got[0].Name != "RAČUN-Rujan.pdf" {
		t.Fatalf("got=%v", got)
	}
}

func TestFilterDoesNotMutateSource(t *testing.T) {
	source := []model.Item{{Name: "zeta.txt"}, {Name: "alpha.txt"}, {Name: "beta.txt"}}
	before := append([]model.Item(nil), source...)
	visible := Filter(source, "a")
	Sort(visible)
	if !reflect.DeepEqual(source, before) {
		t.Fatalf("source mutated: got=%v want=%v", source, before)
	}
}

func TestFilterNoMatchesReturnsEmptySlice(t *testing.T) {
	source := []model.Item{{Name: "alpha.txt"}}
	got := Filter(source, "missing")
	if got == nil || len(got) != 0 {
		t.Fatalf("expected non-nil empty result, got %#v", got)
	}
}
