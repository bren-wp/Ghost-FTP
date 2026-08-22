package config

import (
	"brendigo.com/byftp/internal/i18n"
	"brendigo.com/byftp/internal/model"
	"errors"
	"sync"
)

type SettingsStore struct {
	store  *Store
	mu     sync.Mutex
	loaded bool
	value  model.Settings
}

func NewSettings(s *Store) *SettingsStore { return &SettingsStore{store: s} }

func defaults() model.Settings {
	return model.Settings{
		Language:                 i18n.DefaultLanguage,
		Parallelism:              2,
		BackupBeforeOverwrite:    true,
		ConfirmDelete:            true,
		AutoRetryCount:           0,
		RetryDelaySeconds:        3,
		ConnectionTimeoutSeconds: 15,
	}
}

func normalizeSettings(v model.Settings) model.Settings {
	v.Language = i18n.Normalize(v.Language)
	if v.Parallelism < 1 || v.Parallelism > 8 {
		v.Parallelism = 2
	}
	if v.AutoRetryCount < 0 || v.AutoRetryCount > 3 {
		v.AutoRetryCount = 0
	}
	if v.RetryDelaySeconds < 1 || v.RetryDelaySeconds > 30 {
		v.RetryDelaySeconds = 3
	}
	if v.ConnectionTimeoutSeconds < 5 || v.ConnectionTimeoutSeconds > 60 {
		v.ConnectionTimeoutSeconds = 15
	}
	return v
}

func validateSettings(v model.Settings) error {
	if !i18n.IsSupported(v.Language) {
		return errors.New("unsupported language")
	}
	if v.Parallelism < 1 || v.Parallelism > 8 {
		return errors.New("parallel transfers must be between 1 and 8")
	}
	if v.AutoRetryCount < 0 || v.AutoRetryCount > 3 {
		return errors.New("automatic retries must be between 0 and 3")
	}
	if v.RetryDelaySeconds < 1 || v.RetryDelaySeconds > 30 {
		return errors.New("retry delay must be between 1 and 30 seconds")
	}
	if v.ConnectionTimeoutSeconds < 5 || v.ConnectionTimeoutSeconds > 60 {
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
	_, err := s.store.Read("settings.json", defaults(), &v)
	v = normalizeSettings(v)
	if err == nil {
		s.value = v
		s.loaded = true
	}
	return v, err
}

func (s *SettingsStore) Set(v model.Settings) (model.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Missing values from older settings files migrate to current safe defaults.
	if v.Language == "" {
		v.Language = i18n.DefaultLanguage
	}
	if v.ConnectionTimeoutSeconds == 0 {
		v.ConnectionTimeoutSeconds = 15
	}
	if v.RetryDelaySeconds == 0 {
		v.RetryDelaySeconds = 3
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
