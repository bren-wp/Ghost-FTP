package model

import (
	"encoding/json"
	"testing"
)

func TestProfileMarshalRejectsTransientLinuxPassword(t *testing.T) {
	_, err := json.Marshal(Profile{ID: "p1", PasswordBlob: "linux-secret-v1:socket.token"})
	if err == nil {
		t.Fatal("transient Linux password token was persisted")
	}
}

func TestProfileMarshalRejectsTransientLinuxPassphrase(t *testing.T) {
	_, err := json.Marshal(Profile{ID: "p1", PassphraseBlob: "linux-secret-v1:socket.token"})
	if err == nil {
		t.Fatal("transient Linux passphrase token was persisted")
	}
}

func TestProfileMarshalAllowsNonTransientCredentialBlob(t *testing.T) {
	data, err := json.Marshal(Profile{ID: "p1", PasswordBlob: "opaque-protected-value"})
	if err != nil {
		t.Fatalf("non-transient protected credential rejected: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("profile JSON was empty")
	}
}
