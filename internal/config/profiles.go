package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/profilebinding"
	"github.com/bren-wp/Ghost-FTP/internal/security"
)

type Profiles struct {
	store *Store
	mu    sync.Mutex
}

func NewProfiles(store *Store) *Profiles { return &Profiles{store: store} }
func randomID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type secureProfileEnvelope struct {
	Version   int    `json:"version"`
	Protected string `json:"protected"`
}

const secureProfilesFile = "profiles.secure.json"

func (p *Profiles) loadSecure() ([]model.Profile, bool, error) {
	var envelope secureProfileEnvelope
	source, err := p.store.Read(secureProfilesFile, secureProfileEnvelope{}, &envelope)
	if err != nil {
		return nil, false, err
	}
	if source == "fallback" || envelope.Protected == "" {
		for _, name := range []string{secureProfilesFile, secureProfilesFile + ".previous"} {
			if _, statErr := os.Lstat(filepath.Join(p.store.Dir(), name)); statErr == nil {
				return nil, true, errors.New("spremljene profile nije moguće pročitati")
			} else if !errors.Is(statErr, os.ErrNotExist) {
				return nil, true, statErr
			}
		}
		return nil, false, nil
	}
	if envelope.Version != 1 {
		return nil, true, errors.New("spremljeni profili koriste nepodržan format")
	}
	plain, err := unprotectProfileData(envelope.Protected, p.store.Dir())
	if err != nil {
		return nil, true, errors.New("spremljene profile nije moguće otključati")
	}
	defer func() {
		for i := range plain {
			plain[i] = 0
		}
	}()
	var items []model.Profile
	if err := json.Unmarshal(plain, &items); err != nil {
		return nil, true, errors.New("spremljeni profili su oštećeni")
	}
	if profileDataNeedsMigration(envelope.Protected) {
		if err := p.saveAll(items); err != nil {
			return nil, true, errors.New("spremljene profile nije moguće sigurno migrirati")
		}
	}
	return items, true, nil
}

func (p *Profiles) load() ([]model.Profile, error) {
	if items, found, err := p.loadSecure(); found || err != nil {
		return items, err
	}

	var legacy []model.Profile
	_, err := p.store.Read("profiles.json", []model.Profile{}, &legacy)
	if err != nil {
		return nil, err
	}
	if len(legacy) != 0 {
		if err := p.saveAll(legacy); err != nil {
			return nil, err
		}
	}
	p.removeLegacyProfileFiles()
	return legacy, nil
}

func (p *Profiles) removeLegacyProfileFiles() {
	for _, name := range []string{"profiles.json", "profiles.json.previous"} {
		_ = os.Remove(filepath.Join(p.store.Dir(), name))
	}
}

func (p *Profiles) saveAll(items []model.Profile) error {
	plain, err := json.Marshal(items)
	if err != nil {
		return err
	}
	defer func() {
		for i := range plain {
			plain[i] = 0
		}
	}()
	protected, err := protectProfileData(plain, p.store.Dir())
	if err != nil {
		return err
	}
	if err := p.store.Write(secureProfilesFile, secureProfileEnvelope{Version: 1, Protected: protected}); err != nil {
		return err
	}
	p.removeLegacyProfileFiles()
	return nil
}

func publicProfile(x model.Profile) model.PublicProfile {
	return model.PublicProfile{ID: x.ID, Name: x.Name, Protocol: x.Protocol, Host: x.Host, Port: x.Port, Username: x.Username, HasPassword: x.PasswordBlob != "", PrivateKeyPath: x.PrivateKeyPath, HasPassphrase: x.PassphraseBlob != "", Fingerprint: x.Fingerprint, RemotePath: x.RemotePath, LocalPath: x.LocalPath}
}

func sameSFTPEndpoint(a, b model.Profile) bool {
	return strings.EqualFold(strings.TrimSpace(a.Protocol), "sftp") &&
		strings.EqualFold(strings.TrimSpace(b.Protocol), "sftp") &&
		profilebinding.EndpointMatches(a.Protocol, a.Host, a.Port, b.Protocol, b.Host, b.Port)
}

func sameProfileAccount(a, b model.Profile) bool {
	return profilebinding.AccountMatches(
		a.Protocol, a.Host, a.Port, a.Username,
		b.Protocol, b.Host, b.Port, b.Username,
	)
}

func sameProfilePrivateKey(a, b model.Profile) bool {
	return profilebinding.PrivateKeyMatches(
		a.Protocol, a.Host, a.Port, a.Username, a.PrivateKeyPath,
		b.Protocol, b.Host, b.Port, b.Username, b.PrivateKeyPath,
	)
}

func (p *Profiles) List() ([]model.PublicProfile, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	items, err := p.load()
	if err != nil {
		return nil, err
	}
	out := make([]model.PublicProfile, 0, len(items))
	for _, x := range items {
		out = append(out, publicProfile(x))
	}
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out, nil
}

func (p *Profiles) Get(id string) (model.Profile, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	items, err := p.load()
	if err != nil {
		return model.Profile{}, err
	}
	for _, x := range items {
		if x.ID == id {
			return x, nil
		}
	}
	return model.Profile{}, errors.New("profil nije pronađen")
}

func (p *Profiles) Save(in model.ProfileInput) (model.PublicProfile, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	items, err := p.load()
	if err != nil {
		return model.PublicProfile{}, err
	}

	in.ID = strings.TrimSpace(in.ID)
	var x model.Profile
	idx := -1
	for i, v := range items {
		if v.ID == in.ID && in.ID != "" {
			x = v
			idx = i
			break
		}
	}
	if in.ID != "" && idx < 0 {
		return model.PublicProfile{}, errors.New("profil za izmjenu nije pronađen")
	}
	previous := x
	if x.ID == "" {
		x.ID, err = randomID()
		if err != nil {
			return model.PublicProfile{}, err
		}
	}

	requestedFingerprint := in.Fingerprint
	x.Name = strings.TrimSpace(in.Name)
	x.Protocol = in.Protocol
	x.Host = in.Host
	x.Port = in.Port
	x.Username = in.Username
	x.PrivateKeyPath = in.PrivateKeyPath
	x.RemotePath = in.RemotePath
	x.LocalPath = in.LocalPath
	if x.RemotePath == "" {
		if x.Protocol == "sftp" {
			x.RemotePath = "."
		} else {
			x.RemotePath = "/"
		}
	}
	if x.Name == "" || len(x.Name) > 120 || !utf8.ValidString(x.Name) || strings.ContainsAny(x.Name, "\x00\r\n") {
		return model.PublicProfile{}, errors.New("naziv profila je neispravan")
	}
	if err := security.ValidateRemotePath(x.RemotePath); err != nil {
		return model.PublicProfile{}, err
	}
	if len(x.LocalPath) > 32767 || !utf8.ValidString(x.LocalPath) || strings.ContainsAny(x.LocalPath, "\x00\r\n") {
		return model.PublicProfile{}, errors.New("lokalna putanja profila je neispravna")
	}
	if len(x.PrivateKeyPath) > 32767 || !utf8.ValidString(x.PrivateKeyPath) || strings.ContainsAny(x.PrivateKeyPath, "\x00\r\n") {
		return model.PublicProfile{}, errors.New("putanja privatnog ključa je neispravna")
	}
	if requestedFingerprint != "" {
		if err := security.ValidateSFTPFingerprint(requestedFingerprint); err != nil {
			return model.PublicProfile{}, err
		}
	}
	if err := security.ValidateConnection(x.Protocol, x.Host, x.Username, x.Port); err != nil {
		return model.PublicProfile{}, err
	}

	// Stare profilne tajne ne smiju se automatski preseliti na novi endpoint,
	// korisnički račun ili privatni ključ. Za promijenjeni identitet korisnik
	// mora ponovno upisati tajnu ako je želi spremiti.
	if idx >= 0 && !sameProfileAccount(previous, x) {
		x.PasswordBlob = ""
	}
	if idx >= 0 && !sameProfilePrivateKey(previous, x) {
		x.PassphraseBlob = ""
	}

	// Host-key pin pripada endpointu, ne ostalim poljima profila. Obično uređivanje
	// istog SFTP hosta/porta mora sačuvati postojeći pin. Promjena endpointa ili
	// protokola resetira ga i prisiljava novu trust potvrdu. Eksplicitno predani
	// fingerprint (npr. nakon potvrde host ključa) ima prednost.
	switch {
	case x.Protocol != "sftp":
		x.Fingerprint = ""
	case requestedFingerprint != "":
		x.Fingerprint = requestedFingerprint
	case idx >= 0 && sameSFTPEndpoint(previous, x):
		x.Fingerprint = previous.Fingerprint
	default:
		x.Fingerprint = ""
	}

	if x.Protocol == "sftp" && x.PrivateKeyPath == "" && in.Passphrase != "" {
		return model.PublicProfile{}, errors.New("zaporka privatnog ključa zahtijeva odabran privatni ključ")
	}
	if in.ClearPassword {
		x.PasswordBlob = ""
	} else if in.Password != "" {
		if err = security.ValidateSecret(in.Password); err != nil {
			return model.PublicProfile{}, err
		}
		x.PasswordBlob, err = security.ProtectString(in.Password)
		if err != nil {
			return model.PublicProfile{}, err
		}
	}
	if in.ClearPassphrase {
		x.PassphraseBlob = ""
	} else if in.Passphrase != "" {
		if err = security.ValidateSecret(in.Passphrase); err != nil {
			return model.PublicProfile{}, err
		}
		x.PassphraseBlob, err = security.ProtectString(in.Passphrase)
		if err != nil {
			return model.PublicProfile{}, err
		}
	}
	if x.Protocol == "sftp" && x.PrivateKeyPath == "" {
		x.PassphraseBlob = ""
	}
	if x.Protocol != "sftp" {
		x.PrivateKeyPath = ""
		x.PassphraseBlob = ""
		x.Fingerprint = ""
	}
	if idx >= 0 {
		items[idx] = x
	} else {
		items = append(items, x)
	}
	if err := p.saveAll(items); err != nil {
		return model.PublicProfile{}, err
	}
	return publicProfile(x), nil
}

func (p *Profiles) Remove(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	items, err := p.load()
	if err != nil {
		return err
	}
	out := items[:0]
	found := false
	for _, x := range items {
		if x.ID == id {
			found = true
			continue
		}
		out = append(out, x)
	}
	if !found {
		return errors.New("profil nije pronađen")
	}
	return p.saveAll(out)
}

func (p *Profiles) UpdateFingerprint(id, fp string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	items, err := p.load()
	if err != nil {
		return err
	}
	if err := security.ValidateSFTPFingerprint(fp); err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == id {
			if items[i].Protocol != "sftp" {
				return errors.New("fingerprint je dopušten samo za SFTP profil")
			}
			items[i].Fingerprint = fp
			return p.saveAll(items)
		}
	}
	return errors.New("profil nije pronađen")
}
