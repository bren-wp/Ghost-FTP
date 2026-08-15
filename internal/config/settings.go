package config

import (
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
		Parallelism:              2,
		BackupBeforeOverwrite:    true,
		ConfirmDelete:            true,
		AutoRetryCount:           0,
		RetryDelaySeconds:        3,
		ConnectionTimeoutSeconds: 15,
	}
}

func normalizeSettings(v model.Settings) model.Settings {
	if v.Parallelism < 1 || v.Parallelism > 8 {
		v.Parallelism = 2
	}
	if v.AutoRetryCount < 0 || v.AutoRetryCount > 3 {
		v.AutoRetryCount = 0
	}
	// Zero is a legacy/missing value for retry delay, not a useful runtime delay.
	if v.RetryDelaySeconds < 1 || v.RetryDelaySeconds > 30 {
		v.RetryDelaySeconds = 3
	}
	if v.ConnectionTimeoutSeconds < 5 || v.ConnectionTimeoutSeconds > 60 {
		v.ConnectionTimeoutSeconds = 15
	}
	return v
}

func validateSettings(v model.Settings) error {
	if v.Parallelism < 1 || v.Parallelism > 8 {
		return errors.New("paralelni prijenosi moraju biti između 1 i 8")
	}
	if v.AutoRetryCount < 0 || v.AutoRetryCount > 3 {
		return errors.New("automatska ponavljanja moraju biti između 0 i 3")
	}
	if v.RetryDelaySeconds < 1 || v.RetryDelaySeconds > 30 {
		return errors.New("pauza između pokušaja mora biti između 1 i 30 sekundi")
	}
	if v.ConnectionTimeoutSeconds < 5 || v.ConnectionTimeoutSeconds > 60 {
		return errors.New("vrijeme čekanja spajanja mora biti između 5 i 60 sekundi")
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
	// Zero is accepted only for newly introduced fields so callers compiled
	// against an older settings shape migrate safely to the production default.
	if v.ConnectionTimeoutSeconds == 0 {
		v.ConnectionTimeoutSeconds = 15
	}
	if v.RetryDelaySeconds == 0 {
		v.RetryDelaySeconds = 3
	}
	if err := validateSettings(v); err != nil {
		return v, err
	}
	if err := s.store.Write("settings.json", v); err != nil {
		return v, err
	}
	s.value = v
	s.loaded = true
	return v, nil
}
