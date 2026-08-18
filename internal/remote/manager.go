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

var (
	ErrSessionClosing    = errors.New("prethodna veza se još sigurno zatvara")
	ErrDisconnectTimeout = errors.New("sigurno zatvaranje veze još traje")
)

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
	// Vjerodajnice u čistom tekstu namjerno se ovdje ne spajaju. Obrađuju se
	// odvojeno kako bi spremljeni DPAPI blob ostao šifriran sve do stvarne uporabe.
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

	// Privremeni SFTP trust podaci smiju preživjeti samo prvi korak u kojem
	// korisnik treba potvrditi novi ključ. Svaki uspjeh, otkaz ili greška u
	// sljedećem koraku briše DPAPI-zaštićene privremene vjerodajnice iz memorije.
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
	connectTimeout := 15
	if m.settings != nil {
		if settings, settingsErr := m.settings.Get(); settingsErr == nil && settings.ConnectionTimeoutSeconds >= 5 && settings.ConnectionTimeoutSeconds <= 60 {
			connectTimeout = settings.ConnectionTimeoutSeconds
		}
	}
	if trust == "" {
		// Novi pokušaj povezivanja poništava svaku prethodno neodgovorenu potvrdu povjerenja.
		m.clearPendingTrustLocked()
	}
	var s Session
	if cfg.Protocol == "sftp" {
		knownHostsDir := filepath.Join(m.dataDir, "known_hosts")
		if err := security.EnsureNoRedirectDirectory(m.dataDir, knownHostsDir); err != nil {
			return ConnectResult{}, errors.New("mapa SFTP sesije nije sigurna")
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

	// Nakon predaje protokolarnom adapteru vjerodajnice u čistom tekstu ne ostaju u stanju veze.
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
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-state.done:
		return state.err
	case <-ctx.Done():
		return ErrDisconnectTimeout
	}
}

func (m *Manager) finishSessionClose(state *sessionCloseState, s Session) {
	m.activeOps.Wait()
	if s != nil {
		state.err = s.Close()
	}
	close(state.done)

	m.mu.Lock()
	if m.closing == state {
		m.closing = nil
	}
	m.mu.Unlock()
}

func (m *Manager) Disconnect(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	m.opMu.Lock()
	defer m.opMu.Unlock()
	m.clearPendingTrustLocked()

	// Ako je prethodni poziv već odvojio sesiju, ali je timeout vratio kontrolu
	// pozivatelju prije završnog Close(), samo čekaj isti close-state. Time jedan
	// adapter nikada ne dobiva dvostruki Close i reconnect ne može prijeći preko
	// stare sesije koja još koristi svoje privremene SFTP datoteke.
	m.mu.Lock()
	if m.session == nil {
		state := m.closing
		m.mu.Unlock()
		return waitForSessionClose(ctx, state)
	}

	// Najprije atomarno zatvori ulaz za nove operacije. Operation registrira svoj
	// WaitGroup ref dok još drži RLock, pa nakon ovog write locka nijedna nova
	// operacija ne može početi na sesiji koju ćemo zatvoriti.
	s := m.session
	cancel := m.sessionCancel
	m.session = nil
	m.sessionCtx = nil
	m.sessionCancel = nil
	m.cfg = model.ConnectionConfig{}
	state := &sessionCloseState{done: make(chan struct{})}
	m.closing = state
	m.mu.Unlock()

	// Otkazivanje budi aktivne protokolarne pozive. Čekanje i završni Close rade
	// u odvojenom cleanup putu kako deadline UI-a/shutdowna nikada ne bi blokirao
	// pozivatelja neograničeno. Adapter se ipak ne zatvara prije zadnjeg releasea.
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
	probePath := "/"
	if s.Protocol() == "sftp" {
		probePath = "."
	}
	_, err = s.List(opCtx, probePath)
	return err
}

// Operation vraća aktivnu sesiju s kontekstom koji se otkazuje kada ga
// otkaže pozivatelj ili kada se prekine aktivna ByFTP veza. Svaka uspješna
// registracija mora pozvati release; release je namjerno idempotentan.
func (m *Manager) Operation(ctx context.Context) (Session, context.Context, func(), error) {
	m.mu.RLock()
	s := m.session
	sessionCtx := m.sessionCtx
	if s == nil || sessionCtx == nil {
		m.mu.RUnlock()
		return nil, nil, func() {}, errors.New("nije uspostavljena veza")
	}
	// Add mora biti pod istim RLockom pod kojim je pročitana aktivna sesija.
	// Disconnect tako ne može početi Wait dok je nova operacija između provjere
	// sesije i registracije svog reference counta.
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
	material := fmt.Sprintf("%s\x00%s\x00%d\x00%s\x00%s", strings.ToLower(strings.TrimSpace(cfg.Protocol)), strings.ToLower(strings.TrimSpace(cfg.Host)), cfg.Port, cfg.Username, cfg.Fingerprint)
	sum := sha256.Sum256([]byte(material))
	return hex.EncodeToString(sum[:])
}

// ConnectionIdentity vraća neprozirni memorijski identitet aktivnog odredišta.
// Ne sadrži tajne podatke i sprječava da se prijenos stvoren za jedan
// poslužitelj/račun ponovno pokuša izvršiti prema drugome.
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
