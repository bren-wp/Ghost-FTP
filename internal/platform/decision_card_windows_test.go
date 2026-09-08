//go:build windows

package platform

import (
	"strings"
	"testing"
)

func TestDecisionCardLayoutPreservesCompactMinimum(t *testing.T) {
	layout := decisionCardLayoutForText("Delete?", "example.txt")
	if layout.clientWidth != 680 {
		t.Fatalf("unexpected compact width: %d", layout.clientWidth)
	}
	if layout.clientHeight != 320 {
		t.Fatalf("compact dialog changed height: got %d want 320", layout.clientHeight)
	}
	if layout.bodyHeight != decisionBodyMinH {
		t.Fatalf("compact body changed minimum height: got %d want %d", layout.bodyHeight, decisionBodyMinH)
	}
}

func TestDecisionCardLayoutExpandsForLongLocalizedText(t *testing.T) {
	short := decisionCardLayoutForText("Security", "Short message")
	long := decisionCardLayoutForText(
		"Eine Verbindung, ein Verbindungsversuch oder eine Übertragung ist noch aktiv",
		strings.Repeat("Beim Schließen werden aktive Vorgänge sicher abgebrochen. ", 8),
	)
	if long.clientHeight <= short.clientHeight {
		t.Fatalf("long localized text did not expand dialog: short=%d long=%d", short.clientHeight, long.clientHeight)
	}
	if long.headingHeight <= short.headingHeight {
		t.Fatalf("long heading did not expand: short=%d long=%d", short.headingHeight, long.headingHeight)
	}
	if long.bodyHeight <= short.bodyHeight {
		t.Fatalf("long body did not expand: short=%d long=%d", short.bodyHeight, long.bodyHeight)
	}
	if long.bodyHeight > decisionBodyMaxH || long.headingHeight > decisionHeadingMaxH {
		t.Fatalf("layout exceeded safety caps: heading=%d body=%d", long.headingHeight, long.bodyHeight)
	}
}

func TestDecisionCardEstimatorAccountsForWideScripts(t *testing.T) {
	latin := decisionCardEstimatedLines(strings.Repeat("a", 84), 42)
	wide := decisionCardEstimatedLines(strings.Repeat("界", 84), 42)
	if wide <= latin {
		t.Fatalf("wide-script text should require more estimated lines: latin=%d wide=%d", latin, wide)
	}
}
