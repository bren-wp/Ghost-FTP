package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		// Never follow a link to an unrelated file while recovering local data.
		info, err := os.Lstat(candidate)
		if err != nil {
			return persistedState{}, err
		}
		if !info.Mode().IsRegular() {
			return persistedState{}, fmt.Errorf("non-regular fallback state file: %s", candidate)
		}
		data, err := os.ReadFile(candidate)
		if err != nil {
			return persistedState{}, err
		}
		// A JSON null, missing profile list, or malformed settings is not a
		// fresh install. Reject it and attempt recovery instead of silently
		// replacing the user's saved sites with an empty list.
		if !validJSONKind(json.RawMessage(data), '{') {
			return persistedState{}, fmt.Errorf("invalid fallback state root object")
		}
		var state persistedState
		if err := json.Unmarshal(data, &state); err != nil {
			return persistedState{}, fmt.Errorf("corrupted fallback state: %w", err)
		}
		if !validJSONKind(state.Sites, '[') || !validJSONKind(state.Settings, '{') {
			return persistedState{}, fmt.Errorf("invalid fallback state sites/settings shape")
		}
		if !validLocale(state.Locale) {
			state.Locale = "en"
		}
		state.Sites = sanitizeSensitiveSiteFields(state.Sites)
		return state, nil
	}

	current, currentErr := load(path)
	if currentErr == nil {
		return current, nil
	}
	backup, backupErr := load(path + ".bak")
	if backupErr == nil {
		// Preserve a corrupted regular primary file for diagnostics without
		// clobbering a same-second prior recovery artifact.
		if !errors.Is(currentErr, os.ErrNotExist) {
			preserveFallbackCorrupt(path)
		}
		return backup, nil
	}
	// Both files absent is the only valid first-launch empty-state scenario.
	// Existing but broken backups must never be silently treated as new data.
	if errors.Is(currentErr, os.ErrNotExist) && errors.Is(backupErr, os.ErrNotExist) {
		return persistedState{Locale: "en", Sites: json.RawMessage(`[]`), Settings: json.RawMessage(`{}`)}, nil
	}
	return persistedState{}, fmt.Errorf("fallback state recovery unavailable (primary: %v; backup: %w)", currentErr, backupErr)
}

func preserveFallbackCorrupt(path string) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	// Preserve evidence with a new private filename, not a timestamp-based
	// path that can overwrite another recovery within the same second.
	_, _ = writePrivateStagedFile(filepath.Dir(path), "fallback-state.corrupt.*", data)
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

// writePrivateStagedFile writes a fully synced, unpredictable temporary file
// in the destination directory. It cannot truncate an attacker-chosen .tmp path.
func writePrivateStagedFile(dir, pattern string, data []byte) (string, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	name := f.Name()
	if err := f.Chmod(0600); err != nil {
		_ = f.Close()
		_ = os.Remove(name)
		return "", err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
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

	// Never follow a pre-existing state symlink. Validate both paths before
	// writing anything; the backup must not be a directory or a symlink.
	for _, candidate := range []string{path, path + ".bak"} {
		info, statErr := os.Lstat(candidate)
		if statErr == nil {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("refusing non-regular fallback state path: %s", candidate)
			}
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
	}

	// Preserve the last known-good .bak until AFTER the replacement is
	// committed. Do not replace a valid backup with corrupt former state.
	var stagedBackup string
	previous, readErr := os.ReadFile(path)
	if readErr == nil {
		var old persistedState
		if json.Unmarshal(previous, &old) == nil &&
			validJSONKind(old.Sites, '[') && validJSONKind(old.Settings, '{') {
			stagedBackup, err = writePrivateStagedFile(dir, ".fallback-backup-*.tmp", previous)
			if err != nil {
				return err
			}
			defer os.Remove(stagedBackup)
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return readErr
	}

	stagedState, err := writePrivateStagedFile(dir, ".fallback-state-*.tmp", data)
	if err != nil {
		return err
	}
	defer os.Remove(stagedState)
	// The destination is never removed or renamed out of the way first:
	// failed replacements leave the previous current state and .bak intact.
	if err := os.Rename(stagedState, path); err != nil {
		return fmt.Errorf("failed to atomically replace fallback state: %w", err)
	}
	if stagedBackup != "" {
		if err := os.Rename(stagedBackup, path+".bak"); err != nil {
			// The new state is already committed, while the previous .bak
			// remains intact. Never delete it to make rotation appear green.
			return fmt.Errorf("state saved but backup rotation failed: %w", err)
		}
	}
	// Best-effort directory metadata flush on filesystems that support it.
	if directory, err := os.Open(dir); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
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
