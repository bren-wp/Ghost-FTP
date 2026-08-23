package config

import (
	"os"
	"path/filepath"
	"testing"
)

type partialRecoveryState struct {
	CurrentOnly int `json:"currentOnly,omitempty"`
	Previous    int `json:"previous,omitempty"`
}

func TestStoreRecoveryDoesNotMixCorruptCurrentWithPrevious(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(`{"currentOnly":99,"previous":`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state.json.previous"), []byte(`{"previous":7}`), 0600); err != nil {
		t.Fatal(err)
	}

	got := partialRecoveryState{CurrentOnly: 123, Previous: 456}
	gen, err := New(dir).Read("state.json", partialRecoveryState{}, &got)
	if err != nil {
		t.Fatal(err)
	}
	if gen != "previous" {
		t.Fatalf("generation=%q want previous", gen)
	}
	if got.CurrentOnly != 0 || got.Previous != 7 {
		t.Fatalf("recovery mixed generations: %+v", got)
	}
}

func TestDecodeStateDoesNotMutateDestinationOnError(t *testing.T) {
	got := partialRecoveryState{CurrentOnly: 10, Previous: 20}
	if err := decodeState([]byte(`{"currentOnly":77,"previous":`), &got); err == nil {
		t.Fatal("corrupt JSON unexpectedly decoded")
	}
	if got.CurrentOnly != 10 || got.Previous != 20 {
		t.Fatalf("destination mutated after failed decode: %+v", got)
	}
}

func TestDecodeStateRejectsNonPointerDestination(t *testing.T) {
	if err := decodeState([]byte(`{"previous":1}`), partialRecoveryState{}); err == nil {
		t.Fatal("non-pointer state destination unexpectedly accepted")
	}
}
