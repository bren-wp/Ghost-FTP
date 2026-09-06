package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestTransferProgressTextUsesRuntimeBytes(t *testing.T) {
	job := model.TransferJob{Progress: 3, BytesTransferred: 5 * 1024 * 1024, BytesTotal: 10 * 1024 * 1024}
	if got := transferProgressText(job); got != "50% · 5.00 MB" {
		t.Fatalf("transferProgressText()=%q", got)
	}
}

func TestTransferProgressTextDoesNotInventPercentWithoutTotal(t *testing.T) {
	job := model.TransferJob{Status: "running", BytesTransferred: 5 * 1024 * 1024}
	if got := transferProgressText(job); got != "5.00 MB" {
		t.Fatalf("transferProgressText()=%q", got)
	}
}

func TestTransferProgressTextKeepsLegacyFractionCompatibility(t *testing.T) {
	job := model.TransferJob{Progress: 0.42}
	if got := transferProgressText(job); got != "42%" {
		t.Fatalf("transferProgressText()=%q", got)
	}
}

func TestTransferRuntimeSuffixShowsSpeedAndETAOnlyWhileUseful(t *testing.T) {
	running := model.TransferJob{Status: "running", BytesPerSecond: 2 * 1024 * 1024, ETASeconds: 65}
	if got := transferRuntimeSuffix(running); got != " · 2.00 MB/s · 01:05" {
		t.Fatalf("running suffix=%q", got)
	}
	done := running
	done.Status = "done"
	if got := transferRuntimeSuffix(done); got != " · 2.00 MB/s" {
		t.Fatalf("done suffix=%q", got)
	}
	failed := running
	failed.Status = "failed"
	if got := transferRuntimeSuffix(failed); got != "" {
		t.Fatalf("failed suffix=%q", got)
	}
}

func TestFormatTransferETAHours(t *testing.T) {
	if got := formatTransferETA(3661); got != "1:01:01" {
		t.Fatalf("formatTransferETA()=%q", got)
	}
}
