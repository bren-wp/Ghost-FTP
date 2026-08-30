package remote

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/bren-wp/by-ftp/internal/config"
	"github.com/bren-wp/by-ftp/internal/model"
	"github.com/bren-wp/by-ftp/internal/profilebinding"
	"github.com/bren-wp/by-ftp/internal/security"
)

var (
	ErrSessionClosing    = errors.New("prethodna veza se još sigurno zatvara")
	ErrDisconnectTimeout = errors.New("sigurno zatvaranje veze još traje")
)

func nonNilContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func probePathForSession(s Session) string {
	if s != nil && s.Protocol() == "sftp" {
		return "."
	}
	return "/"
}

type sessionCloseState struct {
	done chan struct{}
	err  error
}

type Manager struct {
	mu            sync.RWMutex
	opMu          sync.Mutex
	activeOps     sync.WaitGroup
	session       Session
	sessionCtx    context.Context
	sessionCancel context.CancelFunc
	closing       *sessionCloseState
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
	return &Manager{profiles: p, settings: settings, dataDir: dataDir, exePath: exePath}
}

type resolvedConnection struct {
	Config         model.ConnectionConfig
	PasswordBlob   string
	PassphraseBlob string
}

// mergeConnection koristi spremljeni profil samo kao početne connection podatke.
// Polje privatnog ključa je autoritativno iz aktualnog UI unosa: prazna
// vrijednost znači "bez privatnog ključa" i ne smije vratiti stari ključ.
// Fingerprint je trust pin i obrađuje se zasebno prema identitetu endpointa.
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
	base.PrivateKeyPath = override.PrivateKeyPath
	base.Fingerprint = override.Fingerprint
	base.Password = override.Password
	base.Passphrase = override.Passphrase
	return base
}

func profileEndpointMatches(profile model.Profile, cfg model.ConnectionConfig) bool {
	return profile.ID != "" && profilebinding.EndpointMatches(
		profile.Protocol, profile.Host, profile.Port,
		cfg.Protocol, cfg.Host, cfg.Port,
	)
}

func profileAccountMatches(profile model.Profile, cfg model.ConnectionConfig) bool {
	return profile.ID != "" && profilebinding.AccountMatches(
		profile.Protocol, profile.Host, profile.Port, profile.Username,
		cfg.Protocol, cfg.Host, cfg.Port, cfg.Username,
	)
}

func profilePrivateKeyMatches(profile model.Profile, cfg model.ConnectionConfig) bool {
	return profile.ID != "" && profilebinding.PrivateKeyMatches(
		profile.Protocol, profile.Host, profile.Port, profile.Username, profile.PrivateKeyPath,
		cfg.Protocol, cfg.Host, cfg.Port, cfg.Username, cfg.PrivateKeyPath,
	)
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
		}, in)
		// Spremljene tajne nikada se ne prenose preko privremeno izmijenjenog
		// endpointa/računa. Korisnik tada mora izričito upisati vjerodajnicu.
		if in.Password == "" && profileAccountMatches(p, resolved.Config) {
			resolved.PasswordBlob = p.PasswordBlob
		}
		if in.Passphrase == "" && profilePrivateKeyMatches(p, resolved.Config) {
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
	if cfg.Fingerprint != "" {
		if err := security.ValidateSFTPFingerprint(cfg.Fingerprint); err != nil {
			return resolved, profile, err
		}
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
	ctx = nonNilContext(ctx)
	m.opMu.Lock()
	defer m.opMu.Unlock()

	preservePendingTrust := false
	defer func() {
		if !preservePendingTrust {
			m.clearPendingTrustLocked()
		}
	}()

	m.mu.RLock()
	alreadyConnected := m.session != nil
	closing := m.closing != nil
	m.mu.RUnlock()
	if alreadyConnected {
		return ConnectResult{}, errors.New("veza je već uspostavljena; prvo prekinite postojeću vezu")
	}
	if closing {
		return ConnectResult{}, ErrSessionClosing
	}

	resolved, profile, err := m.Resolve(profileID, in)
	if err != nil {
		m.clearPendingTrustLocked()
		return ConnectResult{}, err
	}
	cfg := resolved.Config
	profileEndpoint := profileEndpointMatches(profile, cfg)
	connectTimeout := m.settings.Effective().ConnectionTimeoutSeconds
	if trust == "" {
		m.clearPendingTrustLocked()
	}
	var s Session
	if cfg.Protocol == "sftp" {
		knownHostsDir := filepath.Join(m.dataDir, "known_hosts")
		if err := security.EnsureNoRedirectDirectory(m.dataDir, knownHostsDir); err != nil {
			return ConnectResult{}, errors.New("mapa SFTP sesije nije sigurna")
		}
		// Windows klijent je single-instance, pa se crash-ostatci mogu očistiti
		// tek nakon no-redirect provjere. Linux/macOS namjerno dopuštaju više
		// terminalskih procesa; ondje startup cleanup ne smije dirati artefakte
		// druge aktivne sesije.
		if runtime.GOOS == "windows" {
			cleanupStaleSFTPArtifacts(knownHostsDir)
		}
		fp, keyLine, keyAlgorithm, err := ScanFingerprint(ctx, cfg.Host, cfg.Port, knownHostsDir)
		if err != nil {
			return ConnectResult{}, err
		}
		expected := strings.TrimSpace(cfg.Fingerprint)
		if profileEndpoint && profile.Fingerprint != "" {
			expected = profile.Fingerprint
		}
		if expected != "" && expected != fp {
			return ConnectResult{}, errors.New("otisak SFTP host ključa se promijenio; veza je blokirana")
		}
		if expected == "" && trust == "" {
			if err := m.stashPendingTrust(cfg, resolved, fp); err != nil {
				return ConnectResult{}, err
			}
			preservePendingTrust = true
			return ConnectResult{RequiresTrust: true, Fingerprint: fp}, nil
		}
		if trust != "" && trust != fp {
			m.clearPendingTrustLocked()
			return ConnectResult{}, errors.New("potvrđeni otisak SFTP ključa ne odgovara poslužitelju")
		}
		if trust != "" {
			m.applyPendingTrust(cfg, &resolved, fp)
		}
		kh, err := writePrivateTempFile(knownHostsDir, ".byftp-known-*.txt", []byte(keyLine))
		if err != nil {
			return ConnectResult{}, err
		}
		// Privremena promjena hosta/porta smije vrijediti samo za ovu sesiju.
		// Nikada ne prepisuj spremljeni pin originalnog profila drugim endpointom.
		if remember && profileID != "" && profileEndpoint {
			if err := m.profiles.UpdateFingerprint(profileID, fp); err != nil {
				_ = os.Remove(kh)
				return ConnectResult{}, err
			}
		}
		hostKeyConstraint := hostKeyConstraintForScannedKey(keyAlgorithm)
		s, err = newSFTPWithProtectedSecrets(cfg.Host, cfg.Port, cfg.Username, cfg.Password, resolved.PasswordBlob, cfg.PrivateKeyPath, cfg.Passphrase, resolved.PassphraseBlob, kh, hostKeyConstraint, m.exePath, connectTimeout)
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
	if _, err = s.List(cctx, probePathForSession(s)); err != nil {
		_ = s.Close()
		return ConnectResult{}, err
	}

	publicCfg := cfg
	publicCfg.Password = ""
	publicCfg.Passphrase = ""

	sessionCtx, sessionCancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.session = s
	m.sessionCtx = sessionCtx
	m.sessionCancel = sessionCancel
	m.cfg = publicCfg
	m.mu.Unlock()
	return ConnectResult{Connected: true}, nil
}

func waitForSessionClose(ctx context.Context, state *sessionCloseState) error {
	if state == nil {
		return nil
	}
	ctx = nonNilContext(ctx)
	select {
	case <-state.done:
		return state.err
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			return context.Canceled
		}
		return ErrDisconnectTimeout
	}
}

func (m *Manager) finishSessionClose(state *sessionCloseState, s Session) {
	m.activeOps.Wait()
	if s != nil {
		state.err = s.Close()
	}
	m.mu.Lock()
	if m.closing == state {
		m.closing = nil
	}
	m.mu.Unlock()
	close(state.done)
}

func (m *Manager) Disconnect(ctx context.Context) error {
	ctx = nonNilContext(ctx)
	m.opMu.Lock()
	defer m.opMu.Unlock()
	m.clearPendingTrustLocked()

	m.mu.Lock()
	if m.session == nil {
		state := m.closing
		m.mu.Unlock()
		return waitForSessionClose(ctx, state)
	}
	s := m.session
	cancel := m.sessionCancel
	m.session = nil
	m.sessionCtx = nil
	m.sessionCancel = nil
	m.cfg = model.ConnectionConfig{}
	state := &sessionCloseState{done: make(chan struct{})}
	m.closing = state
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	go m.finishSessionClose(state, s)
	return waitForSessionClose(ctx, state)
}

func (m *Manager) Probe(ctx context.Context) error {
	s, opCtx, release, err := m.Operation(ctx)
	if err != nil {
		return err
	}
	defer release()
	_, err = s.List(opCtx, probePathForSession(s))
	return err
}

func (m *Manager) Operation(ctx context.Context) (Session, context.Context, func(), error) {
	ctx = nonNilContext(ctx)
	m.mu.RLock()
	s := m.session
	sessionCtx := m.sessionCtx
	if s == nil || sessionCtx == nil {
		m.mu.RUnlock()
		return nil, nil, func() {}, errors.New("nije uspostavljena veza")
	}
	m.activeOps.Add(1)
	m.mu.RUnlock()

	opCtx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(sessionCtx, cancel)
	var once sync.Once
	release := func() {
		once.Do(func() {
			stop()
			cancel()
			m.activeOps.Done()
		})
	}
	return s, opCtx, release, nil
}

func connectionIdentity(cfg model.ConnectionConfig) string {
	material := fmt.Sprintf("%s\x00%s\x00%s", profilebinding.EndpointKey(cfg.Protocol, cfg.Host, cfg.Port), cfg.Username, cfg.Fingerprint)
	sum := sha256.Sum256([]byte(material))
	return hex.EncodeToString(sum[:])
}

func (m *Manager) ConnectionIdentity() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.session == nil {
		return "", errors.New("nije uspostavljena veza")
	}
	return connectionIdentity(m.cfg), nil
}

func (m *Manager) Config() (model.ConnectionConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg, m.session != nil
}
