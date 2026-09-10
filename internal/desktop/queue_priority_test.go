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
		name                  string
		selected              []int
		top, up, down, bottom bool
	}{
		{name: "none", selected: nil},
		{name: "multiple", selected: []int{1, 3}},
		{name: "running", selected: []int{0}},
		{name: "done", selected: []int{2}},
		{name: "first queued", selected: []int{1}, down: true, bottom: true},
		{name: "middle queued", selected: []int{3}, top: true, up: true, down: true, bottom: true},
		{name: "last queued", selected: []int{4}, top: true, up: true},
		{name: "out of range", selected: []int{99}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveQueuePriorityState(jobs, tc.selected)
			if got.MoveTop != tc.top || got.MoveUp != tc.up || got.MoveDown != tc.down || got.MoveBottom != tc.bottom {
				t.Fatalf("deriveQueuePriorityState(%v) = %#v, want top=%v up=%v down=%v bottom=%v", tc.selected, got, tc.top, tc.up, tc.down, tc.bottom)
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
		if text.MoveTop == "" || text.MoveUp == "" || text.MoveDown == "" || text.MoveBottom == "" || text.MovedTop == "" || text.MovedUp == "" || text.MovedDown == "" || text.MovedBottom == "" {
			t.Fatalf("incomplete queue-priority translation for %s: %#v", language.Code, text)
		}
		if got := queuePriorityWords(language.Code); got != text {
			t.Fatalf("queuePriorityWords(%s) = %#v, want %#v", language.Code, got, text)
		}
	}
}

func TestQueuePriorityWordsNormalizeRegionalLanguage(t *testing.T) {
	if got := queuePriorityWords("hr-HR"); got.MoveTop != "Na vrh" || got.MoveUp != "Pomakni gore" || got.MoveDown != "Pomakni dolje" || got.MoveBottom != "Na dno" {
		t.Fatalf("Croatian regional normalization failed: %#v", got)
	}
	if got := queuePriorityWords("unknown"); got != queuePriorityTranslations["en"] {
		t.Fatalf("unknown language did not fall back to English: %#v", got)
	}
}
