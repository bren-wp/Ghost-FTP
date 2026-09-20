package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type persistedState struct {
	Locale   string          `json:"locale"`
	Sites    json.RawMessage `json:"sites"`
	Settings json.RawMessage `json:"settings"`
}

func fallbackStatePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "GhostFTP", "fallback-state.json"), nil
}

func readPersistedState() (persistedState, error) {
	path, err := fallbackStatePath()
	if err != nil {
		return persistedState{}, err
	}
	load := func(candidate string) (persistedState, error) {
		data, err := os.ReadFile(candidate)
		if err != nil {
			return persistedState{}, err
		}
		var state persistedState
		if err := json.Unmarshal(data, &state); err != nil {
			return persistedState{}, fmt.Errorf("corrupted fallback state: %w", err)
		}
		if !validLocale(state.Locale) {
			state.Locale = "en"
		}
		if !validJSONKind(state.Sites, '[') {
			state.Sites = json.RawMessage(`[]`)
		} else {
			state.Sites = sanitizeSensitiveSiteFields(state.Sites)
		}
		if !validJSONKind(state.Settings, '{') {
			state.Settings = json.RawMessage(`{}`)
		}
		return state, nil
	}
	state, err := load(path)
	if err == nil {
		return state, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		if backup, backupErr := load(path + ".bak"); backupErr == nil {
			return backup, nil
		}
		return persistedState{Locale: "en", Sites: json.RawMessage(`[]`), Settings: json.RawMessage(`{}`)}, nil
	}
	// Recover from the last known-good atomic-write backup instead of failing
	// startup because one JSON file was interrupted or externally corrupted.
	if backup, backupErr := load(path + ".bak"); backupErr == nil {
		preserveFallbackCorrupt(path)
		return backup, nil
	}
	return persistedState{}, err
}

func preserveFallbackCorrupt(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	_ = os.WriteFile(path+".corrupt."+stamp, data, 0600)
}

func sanitizeSensitiveSiteFields(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || !json.Valid(raw) {
		return json.RawMessage(`[]`)
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return json.RawMessage(`[]`)
	}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for key, child := range x {
				n := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
				switch n {
				case "password", "pass", "passphrase", "privatekey", "privatekeypath", "keypath":
					delete(x, key)
					continue
				}
				walk(child)
			}
		case []any:
			for _, child := range x {
				walk(child)
			}
		}
	}
	walk(value)
	clean, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`[]`)
	}
	return clean
}

func rejectSensitiveSiteFields(raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return fmt.Errorf("invalid sites payload: %w", err)
	}
	var walk func(any) error
	walk = func(v any) error {
		switch x := v.(type) {
		case map[string]any:
			for key, child := range x {
				n := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
				switch n {
				case "password", "pass", "passphrase", "privatekey", "privatekeypath", "keypath":
					return fmt.Errorf("sensitive field %q must not be persisted by the compatibility runtime", key)
				}
				if err := walk(child); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range x {
				if err := walk(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(value)
}

func writePersistedState(state persistedState) error {
	if !validLocale(state.Locale) {
		return fmt.Errorf("unsupported locale")
	}
	if !validJSONKind(state.Sites, '[') || !validJSONKind(state.Settings, '{') {
		return fmt.Errorf("invalid state shape")
	}
	if err := rejectSensitiveSiteFields(state.Sites); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path, err := fallbackStatePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	bak := path + ".bak"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Remove(bak)
	if _, statErr := os.Stat(path); statErr == nil {
		if err := os.Rename(path, bak); err != nil {
			_ = os.Remove(tmp)
			return err
		}
	}
	if err := os.Rename(tmp, path); err != nil {
		if _, statErr := os.Stat(bak); statErr == nil {
			_ = os.Rename(bak, path)
		}
		return err
	}
	// Intentionally retain the previous known-good backup for startup recovery.
	return nil
}

func validJSONKind(raw json.RawMessage, first byte) bool {
	if len(raw) == 0 || !json.Valid(raw) {
		return false
	}
	for _, b := range raw {
		if b == ' ' || b == '\n' || b == '\r' || b == '\t' {
			continue
		}
		return b == first
	}
	return false
}

func validLocale(v string) bool {
	switch v {
	case "en", "hr", "de", "fr", "es", "it", "pt", "nl", "pl", "sl", "sr", "bs", "mk", "sq":
		return true
	default:
		return false
	}
}
