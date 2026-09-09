package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestQueuePriorityStateRequiresOneQueuedSelection(t *testing.T) {
	jobs := []model.TransferJob{
		{ID: "running", Status: "running"},
		{ID: "a", Status: "queued"},
		{ID: "done", Status: "done"},
		{ID: "b", Status: "queued"},
		{ID: "c", Status: "queued"},
	}

	cases := []struct {
		name     string
		selected []int
		up       bool
		down     bool
	}{
		{name: "none", selected: nil},
		{name: "multiple", selected: []int{1, 3}},
		{name: "running", selected: []int{0}},
		{name: "done", selected: []int{2}},
		{name: "first queued", selected: []int{1}, down: true},
		{name: "middle queued", selected: []int{3}, up: true, down: true},
		{name: "last queued", selected: []int{4}, up: true},
		{name: "out of range", selected: []int{99}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveQueuePriorityState(jobs, tc.selected)
			if got.MoveUp != tc.up || got.MoveDown != tc.down {
				t.Fatalf("deriveQueuePriorityState(%v) = %#v, want up=%v down=%v", tc.selected, got, tc.up, tc.down)
			}
		})
	}
}

func TestQueuePriorityWordsCoverEverySupportedLanguage(t *testing.T) {
	for _, language := range i18n.Languages() {
		text, ok := queuePriorityTranslations[language.Code]
		if !ok {
			t.Fatalf("missing queue-priority translation for %s", language.Code)
		}
		if text.MoveUp == "" || text.MoveDown == "" || text.MovedUp == "" || text.MovedDown == "" {
			t.Fatalf("incomplete queue-priority translation for %s: %#v", language.Code, text)
		}
		if got := queuePriorityWords(language.Code); got != text {
			t.Fatalf("queuePriorityWords(%s) = %#v, want %#v", language.Code, got, text)
		}
	}
}

func TestQueuePriorityWordsNormalizeRegionalLanguage(t *testing.T) {
	if got := queuePriorityWords("hr-HR"); got.MoveUp != "Pomakni gore" || got.MoveDown != "Pomakni dolje" {
		t.Fatalf("Croatian regional normalization failed: %#v", got)
	}
	if got := queuePriorityWords("unknown"); got != queuePriorityTranslations["en"] {
		t.Fatalf("unknown language did not fall back to English: %#v", got)
	}
}
