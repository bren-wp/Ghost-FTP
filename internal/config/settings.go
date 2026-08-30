package config

import (
	"errors"
	"sync"

	"github.com/bren-wp/by-ftp/internal/i18n"
	"github.com/bren-wp/by-ftp/internal/model"
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
// own copies of timeout, retry and parallelism defaults.
func DefaultSettings() model.Settings {
	return model.Settings{
		Language:                 i18n.DefaultLanguage,
		Parallelism:              DefaultParallelism,
		BackupBeforeOverwrite:    true,
		ConfirmDelete:            true,
		AutoRetryCount:           DefaultAutoRetryCount,
		RetryDelaySeconds:        DefaultRetryDelaySeconds,
		ConnectionTimeoutSeconds: DefaultConnectionTimeoutSeconds,
	}
}

func normalizeSettings(v model.Settings) model.Settings {
	v.Language = i18n.Normalize(v.Language)
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
	return v
}

func validateSettings(v model.Settings) error {
	if !i18n.IsSupported(v.Language) {
		return errors.New("unsupported language")
	}
	if v.Parallelism < MinParallelism || v.Parallelism > MaxParallelism {
		return errors.New("parallel transfers must be between 1 and 8")
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
	// Missing values from older settings files migrate to current safe defaults.
	if v.Language == "" {
		v.Language = i18n.DefaultLanguage
	}
	if v.ConnectionTimeoutSeconds == 0 {
		v.ConnectionTimeoutSeconds = DefaultConnectionTimeoutSeconds
	}
	if v.RetryDelaySeconds == 0 {
		v.RetryDelaySeconds = DefaultRetryDelaySeconds
	}
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
