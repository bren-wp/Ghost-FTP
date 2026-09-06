package config

import (
	"errors"
	"sync"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

const (
	DefaultParallelism              = 2
	MinParallelism                  = 1
	MaxParallelism                  = 8
	DefaultAutoRetryCount           = 0
	MinAutoRetryCount               = 0
	MaxAutoRetryCount               = 3
	DefaultRetryDelaySeconds        = 3
	MinRetryDelaySeconds            = 1
	MaxRetryDelaySeconds            = 30
	DefaultConnectionTimeoutSeconds = 15
	MinConnectionTimeoutSeconds     = 5
	MaxConnectionTimeoutSeconds     = 60
)

type SettingsStore struct {
	store  *Store
	mu     sync.Mutex
	loaded bool
	value  model.Settings
}

func NewSettings(s *Store) *SettingsStore { return &SettingsStore{store: s} }

// DefaultSettings is the single safe runtime fallback for settings. Runtime
// callers use the same values as settings migration instead of carrying their
// own copies of timeout, retry and parallelism defaults. Classic Light is the
// primary appearance for fresh installs; an explicitly persisted Dark choice
// remains canonical and is never overwritten by normalization.
func DefaultSettings() model.Settings {
	return model.Settings{
		Language:                 i18n.DefaultLanguage,
		Appearance:               model.AppearanceLight,
		Parallelism:              DefaultParallelism,
		ConflictPolicy:           model.ConflictPolicyReplaceBackup,
		BackupBeforeOverwrite:    true,
		ConfirmDelete:            true,
		AutoRetryCount:           DefaultAutoRetryCount,
		RetryDelaySeconds:        DefaultRetryDelaySeconds,
		ConnectionTimeoutSeconds: DefaultConnectionTimeoutSeconds,
	}
}

func validAppearance(appearance string) bool {
	switch appearance {
	case model.AppearanceDark, model.AppearanceLight:
		return true
	default:
		return false
	}
}

func validConflictPolicy(policy string) bool {
	switch policy {
	case model.ConflictPolicySkip, model.ConflictPolicyReplace, model.ConflictPolicyReplaceBackup:
		return true
	default:
		return false
	}
}

// migrateConflictPolicy converts the former pair of overwrite booleans into
// one explicit policy and then synchronizes the legacy fields. The legacy JSON
// fields remain present so older components can continue to read state while
// the canonical policy removes contradictory combinations for new code.
func migrateConflictPolicy(v model.Settings, persisted bool) model.Settings {
	if v.ConflictPolicy == "" {
		switch {
		case v.SkipExisting:
			v.ConflictPolicy = model.ConflictPolicySkip
		case v.BackupBeforeOverwrite:
			v.ConflictPolicy = model.ConflictPolicyReplaceBackup
		default:
			v.ConflictPolicy = model.ConflictPolicyReplace
		}
	} else if !validConflictPolicy(v.ConflictPolicy) && persisted {
		// Corrupt or unknown persisted state must not silently weaken overwrite
		// recovery. Fall back to the conservative policy.
		v.ConflictPolicy = model.ConflictPolicyReplaceBackup
	}

	switch v.ConflictPolicy {
	case model.ConflictPolicySkip:
		v.SkipExisting = true
		// Keep the legacy backup flag conservative for older components. It is
		// operationally irrelevant while SkipExisting is true because no
		// destination overwrite occurs.
		v.BackupBeforeOverwrite = true
	case model.ConflictPolicyReplace:
		v.SkipExisting = false
		v.BackupBeforeOverwrite = false
	case model.ConflictPolicyReplaceBackup:
		v.SkipExisting = false
		v.BackupBeforeOverwrite = true
	}
	return v
}

func normalizeSettings(v model.Settings) model.Settings {
	v.Language = i18n.Normalize(v.Language)
	if !validAppearance(v.Appearance) {
		v.Appearance = model.AppearanceLight
	}
	if v.Parallelism < MinParallelism || v.Parallelism > MaxParallelism {
		v.Parallelism = DefaultParallelism
	}
	if v.AutoRetryCount < MinAutoRetryCount || v.AutoRetryCount > MaxAutoRetryCount {
		v.AutoRetryCount = DefaultAutoRetryCount
	}
	if v.RetryDelaySeconds < MinRetryDelaySeconds || v.RetryDelaySeconds > MaxRetryDelaySeconds {
		v.RetryDelaySeconds = DefaultRetryDelaySeconds
	}
	if v.ConnectionTimeoutSeconds < MinConnectionTimeoutSeconds || v.ConnectionTimeoutSeconds > MaxConnectionTimeoutSeconds {
		v.ConnectionTimeoutSeconds = DefaultConnectionTimeoutSeconds
	}
	return migrateConflictPolicy(v, true)
}

func validateSettings(v model.Settings) error {
	if !i18n.IsSupported(v.Language) {
		return errors.New("unsupported language")
	}
	if !validAppearance(v.Appearance) {
		return errors.New("appearance must be dark or light")
	}
	if v.Parallelism < MinParallelism || v.Parallelism > MaxParallelism {
		return errors.New("parallel transfers must be between 1 and 8")
	}
	if !validConflictPolicy(v.ConflictPolicy) {
		return errors.New("conflict policy must be skip, replace, or replace_backup")
	}
	if v.AutoRetryCount < MinAutoRetryCount || v.AutoRetryCount > MaxAutoRetryCount {
		return errors.New("automatic retries must be between 0 and 3")
	}
	if v.RetryDelaySeconds < MinRetryDelaySeconds || v.RetryDelaySeconds > MaxRetryDelaySeconds {
		return errors.New("retry delay must be between 1 and 30 seconds")
	}
	if v.ConnectionTimeoutSeconds < MinConnectionTimeoutSeconds || v.ConnectionTimeoutSeconds > MaxConnectionTimeoutSeconds {
		return errors.New("connection timeout must be between 5 and 60 seconds")
	}
	return nil
}

func (s *SettingsStore) Get() (model.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loaded {
		return s.value, nil
	}
	var v model.Settings
	_, err := s.store.Read("settings.json", DefaultSettings(), &v)
	v = normalizeSettings(v)
	if err == nil {
		s.value = v
		s.loaded = true
	}
	return v, err
}

// Effective returns validated settings for runtime scheduling/connection code.
// If the state store is unavailable, safe defaults keep the client operational
// while preserving conservative overwrite/delete behavior.
func (s *SettingsStore) Effective() model.Settings {
	if s == nil || s.store == nil {
		return DefaultSettings()
	}
	v, err := s.Get()
	if err != nil {
		return DefaultSettings()
	}
	return v
}

func (s *SettingsStore) Set(v model.Settings) (model.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Missing values from older clients migrate to current safe defaults and
	// legacy overwrite booleans are converted into the canonical policy.
	if v.Language == "" {
		v.Language = i18n.DefaultLanguage
	}
	if v.Appearance == "" {
		v.Appearance = model.AppearanceLight
	}
	if v.ConnectionTimeoutSeconds == 0 {
		v.ConnectionTimeoutSeconds = DefaultConnectionTimeoutSeconds
	}
	if v.RetryDelaySeconds == 0 {
		v.RetryDelaySeconds = DefaultRetryDelaySeconds
	}
	v = migrateConflictPolicy(v, false)
	if err := validateSettings(v); err != nil {
		return v, err
	}
	v.Language = i18n.Normalize(v.Language)
	if err := s.store.Write("settings.json", v); err != nil {
		return v, err
	}
	s.value = v
	s.loaded = true
	return v, nil
}
