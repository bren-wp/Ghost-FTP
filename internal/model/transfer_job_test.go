package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTransferJobRuntimeClockIsNotSerialized(t *testing.T) {
	job := TransferJob{
		ID:               "job",
		Status:           "running",
		Progress:         50,
		BytesTransferred: 512,
		BytesTotal:       1024,
		BytesPerSecond:   256,
		ETASeconds:       2,
		StartedAt:        "2026-09-06T19:00:00Z",
	}
	data, err := json.Marshal(job)
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(data)
	if strings.Contains(encoded, "startedAt") || strings.Contains(encoded, job.StartedAt) {
		t.Fatalf("runtime clock leaked into transfer JSON: %s", encoded)
	}
	for _, field := range []string{"bytesTransferred", "bytesTotal", "bytesPerSecond", "etaSeconds"} {
		if !strings.Contains(encoded, `"`+field+`"`) {
			t.Fatalf("runtime metric %q missing from transfer JSON: %s", field, encoded)
		}
	}
}
