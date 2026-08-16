package remote

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"brendigo.com/byftp/internal/config"
	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/security"
)

type Manager struct {
	mu            sync.RWMutex
	opMu          sync.Mutex
	session       Session
	sessionCtx    context.Context
	sessionCancel context.CancelFunc
	cfg           model.ConnectionConfig
	profiles      *config.Profiles
	settings      *config.SettingsStore
	dataDir       string
	exePath       string
	pendingTrust  pendingTrustState
}

type pendingTrustState struct {
	host, username, keyPath, fingerprint string
	port                                 int
	passwordBlob, passphraseBlob         string
	expires                              time.Time
}

func NewManager(p *config.Profiles, settings *config.SettingsStore, dataDir, exePath string) *Manager {
	cleanupStaleSFTPArtifacts(filepath.Join(dataDir, "known_hosts"))
	return &Manager{profiles: p, settings: settings, dataDir: dataDir, exePath: exePath}
}

type resolvedConnection struct {
	Config         model.ConnectionConfig
	PasswordBlob   string
	PassphraseBlob string
}

func mergeConnection(base model.ConnectionConfig, override model.ConnectionConfig) model.ConnectionConfig {
	if override.Protocol != "" {
		base.Protocol = override.Protocol
	}
	if override.Host != "" {
		base.Host = override.Host
	}
	if override.Port != 0 {
		base.Port = override.Port
	}
	if override.Username != "" {
		base.Username = override.Username
	}
	if override.PrivateKeyPath != "" {
		base.PrivateKeyPath = override.PrivateKeyPath
	}
	if override.Fingerprint != "" {
		base.Fingerprint = override.Fingerprint
	}
	// Plaintext secrets are intentionally not merged here. They are handled
	// separately so a saved DPAPI blob can remain encrypted until actual use.
	base.Password = override.Password
	base.Passphrase = override.Passphrase
	return base
}

func (m *Manager) Resolve(profileID string, in model.ConnectionConfig) (resolvedConnection, model.Profile, error) {
	var profile model.Profile
	resolved := resolvedConnection{Config: in}
	if profileID != "" {
		p, err := m.profiles.Get(profileID)
		if err != nil {
			return resolved, profile, err
		}
		profile = p
		resolved.Config = mergeConnection(model.ConnectionConfig{
			Protocol: p.Protocol, Host: p.Host, Port: p.Port, Username: p.Username,
			PrivateKeyPath: p.PrivateKeyPath, Fingerprint: p.Fingerprint,
		}, in)
		if in.Password == "" {
			resolved.PasswordBlob = p.PasswordBlob
		}
		if in.Passphrase == "" {
			resolved.PassphraseBlob = p.PassphraseBlob
		}
	}
	cfg := resolved.Config
	if err := security.ValidateConnection(cfg.Protocol, cfg.Host, cfg.Username, cfg.Port); err != nil {
		return resolved, profile, err
	}
	if err := security.ValidateSecret(cfg.Password); err != nil {
		return resolved, profile, err
	}
	if err := security.ValidateSecret(cfg.Passphrase); err != nil {
		return resolved, profile, err
	}
	return resolved, profile, nil
}

type ConnectResult struct {
	Connected     bool   `json:"connected"`
	RequiresTrust bool   `json:"requiresTrust,omitempty"`
	Fingerprint   string `json:"fingerprint,omitempty"`
}

func (m *Manager) clearPendingTrustLocked() {
	m.pendingTrust = pendingTrustState{}
}

func (m *Manager) CancelPendingTrust() {
	m.opMu.Lock()
	defer m.opMu.Unlock()
	m.clearPendingTrustLocked()
}

func (m *Manager) stashPendingTrust(cfg model.ConnectionConfig, resolved resolvedConnection, fingerprint string) error {
	passwordBlob := resolved.PasswordBlob
	passphraseBlob := resolved.PassphraseBlob
	var err error
	if cfg.Password != "" {
		passwordBlob, err = security.ProtectString(cfg.Password)
		if err != nil {
			return err
		}
	}
	if cfg.Passphrase != "" {
		passphraseBlob, err = security.ProtectString(cfg.Passphrase)
		if err != nil {
			return err
		}
	}
	m.pendingTrust = pendingTrustState{
		host: cfg.Host, port: cfg.Port, username: cfg.Username, keyPath: cfg.PrivateKeyPath, fingerprint: fingerprint,
		passwordBlob: passwordBlob, passphraseBlob: passphraseBlob, expires: time.Now().Add(2 * time.Minute),
	}
	return nil
}

func (m *Manager) applyPendingTrust(cfg model.ConnectionConfig, resolved *resolvedConnection, fingerprint string) {
	p := m.pendingTrust
	m.clearPendingTrustLocked()
	if time.Now().After(p.expires) || p.host != cfg.Host || p.port != cfg.Port || p.username != cfg.Username || p.keyPath != cfg.PrivateKeyPath || p.fingerprint != fingerprint {
		return
	}
	if cfg.Password == "" && resolved.PasswordBlob == "" {
		resolved.PasswordBlob = p.passwordBlob
	}
	if cfg.Passphrase == "" && resolved.PassphraseBlob == "" {
		resolved.PassphraseBlob = p.passphraseBlob
	}
}

func (m *Manager) Connect(ctx context.Context, profileID string, in model.ConnectionConfig, trust string, remember bool) (ConnectResult, error) {
	m.opMu.Lock()
	defer m.opMu.Unlock()

	m.mu.RLock()
	alreadyConnected := m.session != nil
	m.mu.RUnlock()
	if alreadyConnected {
		return ConnectResult{}, errors.New("veza je već uspostavljena; prvo prekinite postojeću vezu")
	}

	resolved, profile, err := m.Resolve(profileID, in)
	if err != nil {
		m.clearPendingTrustLocked()
		return ConnectResult{}, err
	}
	cfg := resolved.Config
	connectTimeout := 15
	if m.settings != nil {
		if settings, settingsErr := m.settings.Get(); settingsErr == nil && settings.ConnectionTimeoutSeconds >= 5 && settings.ConnectionTimeoutSeconds <= 60 {
			connectTimeout = settings.ConnectionTimeoutSeconds
		}
	}
	if trust == "" {
		// A fresh connect attempt supersedes any previously unanswered trust prompt.
		m.clearPendingTrustLocked()
	}
	var s Session
	if cfg.Protocol == "sftp" {
		knownHostsDir := filepath.Join(m.dataDir, "known_hosts")
		if err := security.EnsureNoRedirectDirectory(m.dataDir, knownHostsDir); err != nil {
			return ConnectResult{}, errors.New("SFTP session mapa nije sigurna")
		}
		fp, keyLine, keyAlgorithm, err := ScanFingerprint(ctx, cfg.Host, cfg.Port, knownHostsDir)
		if err != nil {
			return ConnectResult{}, err
		}
		expected := cfg.Fingerprint
		if profile.Fingerprint != "" {
			expected = profile.Fingerprint
		}
		if expected != "" && expected != fp {
			return ConnectResult{}, errors.New("SFTP host fingerprint se promijenio; veza je blokirana")
		}
		if expected == "" && trust == "" {
			if err := m.stashPendingTrust(cfg, resolved, fp); err != nil {
				return ConnectResult{}, err
			}
			return ConnectResult{RequiresTrust: true, Fingerprint: fp}, nil
		}
		if trust != "" && trust != fp {
			m.clearPendingTrustLocked()
			return ConnectResult{}, errors.New("potvrđeni SFTP fingerprint ne odgovara poslužitelju")
		}
		if trust != "" {
			m.applyPendingTrust(cfg, &resolved, fp)
		}
		kh, err := writePrivateTempFile(knownHostsDir, ".byftp-known-*.txt", []byte(keyLine))
		if err != nil {
			return ConnectResult{}, err
		}
		if remember && profileID != "" {
			if err := m.profiles.UpdateFingerprint(profileID, fp); err != nil {
				_ = os.Remove(kh)
				return ConnectResult{}, err
			}
		}
		s, err = newSFTPWithProtectedSecrets(cfg.Host, cfg.Port, cfg.Username, cfg.Password, resolved.PasswordBlob, cfg.PrivateKeyPath, cfg.Passphrase, resolved.PassphraseBlob, kh, keyAlgorithm, m.exePath, connectTimeout)
		if err != nil {
			_ = os.Remove(kh)
			return ConnectResult{}, err
		}
		cfg.Fingerprint = fp
	} else {
		s, err = newCurlFTPWithProtectedSecret(cfg.Protocol, cfg.Host, cfg.Port, cfg.Username, cfg.Password, resolved.PasswordBlob, connectTimeout)
		if err != nil {
			return ConnectResult{}, err
		}
	}

	cctx, cancel := context.WithTimeout(ctx, time.Duration(connectTimeout+5)*time.Second)
	defer cancel()
	probePath := "/"
	if s.Protocol() == "sftp" {
		probePath = "."
	}
	if _, err = s.List(cctx, probePath); err != nil {
		_ = s.Close()
		return ConnectResult{}, err
	}

	// Do not keep plaintext secrets in connection state after the protocol adapter owns them.
	publicCfg := cfg
	publicCfg.Password = ""
	publicCfg.Passphrase = ""

	sessionCtx, sessionCancel := context.WithCancel(context.Background())
	m.mu.Lock()
	old := m.session
	oldCancel := m.sessionCancel
	m.session = s
	m.sessionCtx = sessionCtx
	m.sessionCancel = sessionCancel
	m.cfg = publicCfg
	m.mu.Unlock()
	if oldCancel != nil {
		oldCancel()
	}
	if old != nil {
		_ = old.Close()
	}
	return ConnectResult{Connected: true}, nil
}

func (m *Manager) Disconnect() error {
	m.opMu.Lock()
	defer m.opMu.Unlock()
	m.clearPendingTrustLocked()
	m.mu.Lock()
	s := m.session
	cancel := m.sessionCancel
	m.session = nil
	m.sessionCtx = nil
	m.sessionCancel = nil
	m.cfg = model.ConnectionConfig{}
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if s != nil {
		return s.Close()
	}
	return nil
}

func (m *Manager) Probe(ctx context.Context) error {
	s, opCtx, release, err := m.Operation(ctx)
	if err != nil {
		return err
	}
	defer release()
	probePath := "/"
	if s.Protocol() == "sftp" {
		probePath = "."
	}
	_, err = s.List(opCtx, probePath)
	return err
}

// Operation returns the active session with a context that is cancelled when
// either the caller cancels or the active ByFTP connection is disconnected.
func (m *Manager) Operation(ctx context.Context) (Session, context.Context, func(), error) {
	m.mu.RLock()
	s := m.session
	sessionCtx := m.sessionCtx
	m.mu.RUnlock()
	if s == nil || sessionCtx == nil {
		return nil, nil, func() {}, errors.New("nije uspostavljena veza")
	}
	opCtx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(sessionCtx, cancel)
	release := func() {
		stop()
		cancel()
	}
	return s, opCtx, release, nil
}

func connectionIdentity(cfg model.ConnectionConfig) string {
	material := fmt.Sprintf("%s\x00%s\x00%d\x00%s\x00%s", strings.ToLower(strings.TrimSpace(cfg.Protocol)), strings.ToLower(strings.TrimSpace(cfg.Host)), cfg.Port, cfg.Username, cfg.Fingerprint)
	sum := sha256.Sum256([]byte(material))
	return hex.EncodeToString(sum[:])
}

// ConnectionIdentity returns an opaque in-memory identity for the active
// endpoint. It contains no secret and lets the transfer queue prevent a job
// created for one server/account from being retried against another.
func (m *Manager) ConnectionIdentity() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.session == nil {
		return "", errors.New("nije uspostavljena veza")
	}
	return connectionIdentity(m.cfg), nil
}

func (m *Manager) Session() (Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.session == nil {
		return nil, errors.New("nije uspostavljena veza")
	}
	return m.session, nil
}

func (m *Manager) Config() (model.ConnectionConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg, m.session != nil
}
