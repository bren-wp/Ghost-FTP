package model

import (
	"encoding/json"
	"errors"
	"strings"
)

const linuxTransientSecretPrefix = "linux-secret-v1:"

// MarshalJSON enforces a persistence boundary for Linux AskPass credentials.
// The linux-secret-v1 token references a short-lived, process-owned memory
// broker and is intentionally unsuitable for disk persistence. Windows DPAPI
// blobs do not use this prefix and continue through the normal profile path.
func (p Profile) MarshalJSON() ([]byte, error) {
	if strings.HasPrefix(p.PasswordBlob, linuxTransientSecretPrefix) || strings.HasPrefix(p.PassphraseBlob, linuxTransientSecretPrefix) {
		return nil, errors.New("transient Linux credentials cannot be persisted in a profile")
	}
	type profileAlias Profile
	return json.Marshal(profileAlias(p))
}
