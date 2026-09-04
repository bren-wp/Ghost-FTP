//go:build windows

package desktop

import (
	"context"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
	"github.com/bren-wp/Ghost-FTP/internal/profilebinding"
	"github.com/bren-wp/Ghost-FTP/internal/remote"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

func (a *app) cancelConnectionAttempt() {
	if a.connectionCancel != nil {
		a.connectionCancel()
		a.connectionCancel = nil
	}
}

func (a *app) cancelHealthCheck() {
	if a.healthCheckCancel != nil {
		a.healthCheckCancel()
		a.healthCheckCancel = nil
	}
	a.healthCheckRunning = false
}

// beginConnectionTransition invalidates every asynchronous callback that was
// created for an older session. The counter is UI-thread owned and is checked
// again when asynchronous work dispatches its result back to the window.
func (a *app) beginConnectionTransition() uint64 {
	a.connectionGeneration++
	a.cancelHealthCheck()
	return a.connectionGeneration
}

func (a *app) connectionContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	a.cancelConnectionAttempt()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	a.connectionCancel = cancel
	return ctx, cancel
}

func (a *app) syncDefaultPort() {
	idx, _, _ := sendMessageW.Call(a.protocol, cbGetCurSel, 0, 0)
	setText(a.port, protocolAt(idx).Port)
}

func (a *app) protocolValue() string {
	idx, _, _ := sendMessageW.Call(a.protocol, cbGetCurSel, 0, 0)
	return protocolAt(idx).Value
}

func (a *app) setProtocolValue(protocol string) {
	sendMessageW.Call(a.protocol, cbSetCurSel, protocolIndex(protocol), 0)
	a.updateProtocolControls()
}

func (a *app) setConnectionBusy(busy bool) {
	a.connectionBusy = busy
	if !busy {
		a.setConnectionUI(a.connected)
		return
	}
	for _, h := range []uintptr{
		a.connect, a.disconnect, a.profilesCombo, a.protocol, a.host, a.port, a.user, a.pass,
		a.keyPath, a.chooseKey, a.passphrase, a.saveProfile, a.removeProfile, a.settingsBtn,
	} {
		setControlEnabled(h, false)
	}
	a.setRemoteControls(false)
	a.updateActionControls()
}

func (a *app) setConnectionUI(connected bool) {
	a.connected = connected
	a.connectionBusy = false
	setControlEnabled(a.connect, !connected)
	setControlEnabled(a.disconnect, connected)
	for _, h := range []uintptr{a.profilesCombo, a.protocol, a.host, a.port, a.user, a.pass} {
		setControlEnabled(h, !connected)
	}
	if connected {
		for _, h := range []uintptr{a.keyPath, a.chooseKey, a.passphrase} {
			setControlEnabled(h, false)
		}
		setText(a.connectionBadge, "● POVEZANO")
	} else {
		setText(a.connectionBadge, "● NIJE POVEZANO")
		a.updateProtocolControls()
	}
	a.setRemoteControls(connected)
	a.updateActionControls()
	invalidateRect.Call(a.hwnd, 0, 1)
}

func (a *app) connectNow() {
	if a.connectionBusy || a.connected {
		return
	}
	host := getText(a.host)
	user := getText(a.user)
	password := getText(a.pass)
	protocol := a.protocolValue()
	port, err := validateRawConnectionInput(protocol, host, getText(a.port), user)
	if err != nil {
		if err == errInvalidConnectionPort {
			platform.ErrorDialog("GhostFTP", "Neispravan port", "Port mora biti broj između 1 i 65535.")
		} else {
			platform.ErrorDialog("GhostFTP — povezivanje", "Neispravni podaci veze", usererror.Message(err, "Provjerite poslužitelj, port i korisničko ime."))
		}
		return
	}
	cfg := model.ConnectionConfig{
		Protocol:       protocol,
		Host:           host,
		Port:           port,
		Username:       user,
		Password:       password,
		PrivateKeyPath: strings.TrimSpace(getText(a.keyPath)),
		Passphrase:     getText(a.passphrase),
	}
	generation := a.beginConnectionTransition()
	// Unesene tajne ostaju u zaključanim edit kontrolama dok veza stvarno ne
	// uspije. Time korisnik nakon mrežne/port greške može ponoviti pokušaj bez
	// ponovnog upisa, a onConnected ih briše odmah nakon potvrđenog spajanja.
	a.setConnectionBusy(true)
	a.setStatus("Povezivanje s " + host + "…")
	profileID := a.selectedProfileID
	ctx, cancel := a.connectionContext(75 * time.Second)
	a.goSafe(func() {
		defer cancel()
		r, err := a.engine.Connect(ctx, profileID, cfg, "", false)
		cfg.Password = ""
		cfg.Passphrase = ""
		a.dispatch(func() {
			if generation != a.connectionGeneration {
				return
			}
			a.connectionCancel = nil
			if err != nil {
				a.setConnectionUI(false)
				a.setStatus(usererror.Message(err, "Povezivanje nije uspjelo. Provjerite podatke i pokušajte ponovno."))
				platform.ErrorDialog("GhostFTP — povezivanje", "Povezivanje nije uspjelo", usererror.Message(err, "Provjerite podatke za prijavu i mrežnu vezu."))
				return
			}
			if r.RequiresTrust {
				if !platform.ConfirmDialog("GhostFTP — SFTP sigurnost", "Novi SFTP ključ poslužitelja", "Provjerite otisak sigurnosnog ključa prije prihvaćanja:\n\n"+r.Fingerprint+"\n\nVjerujete li ovom poslužitelju?") {
					a.engine.CancelPendingTrust()
					a.setConnectionUI(false)
					a.setStatus("SFTP povezivanje otkazano.")
					return
				}
				a.connectTrusted(profileID, cfg, r.Fingerprint, generation)
				return
			}
			a.onConnected(host, r.Diagnostics)
		})
	})
}

func (a *app) connectTrusted(profileID string, cfg model.ConnectionConfig, fingerprint string, generation uint64) {
	if generation != a.connectionGeneration {
		return
	}
	// Brzi SFTP spoj ne ovisi samo o kratkotrajnom pending-trust cacheu.
	// Kontrole su tijekom pokušaja zaključane pa se upravo unesena tajna može
	// sigurno ponovno uzeti. Za spremljeni profil prazno polje i dalje dopušta
	// Resolveu korištenje pravilno vezane spremljene vjerodajnice.
	if cfg.Password == "" {
		cfg.Password = getText(a.pass)
	}
	if cfg.Passphrase == "" {
		cfg.Passphrase = getText(a.passphrase)
	}
	a.setStatus("Provjera SFTP ključa i povezivanje…")
	ctx, cancel := a.connectionContext(75 * time.Second)
	a.goSafe(func() {
		defer cancel()
		r, err := a.engine.Connect(ctx, profileID, cfg, fingerprint, profileID != "")
		cfg.Password = ""
		cfg.Passphrase = ""
		a.dispatch(func() {
			if generation != a.connectionGeneration {
				return
			}
			a.connectionCancel = nil
			if err != nil {
				a.setConnectionUI(false)
				a.setStatus(usererror.Message(err, "SFTP povezivanje nije uspjelo."))
				platform.ErrorDialog("GhostFTP — SFTP", "Povezivanje nije uspjelo", usererror.Message(err, "Provjerite podatke za prijavu i SFTP postavke."))
				return
			}
			a.onConnected(cfg.Host, r.Diagnostics)
		})
	})
}

func (a *app) currentEndpointMatchesProfile(p model.PublicProfile) bool {
	port, err := strconv.Atoi(getText(a.port))
	if err != nil {
		return false
	}
	return profilebinding.EndpointMatches(
		p.Protocol, p.Host, p.Port,
		a.protocolValue(), getText(a.host), port,
	)
}

func connectionDiagnosticStatus(host string, diagnostics remote.ConnectionDiagnostics) string {
	parts := []string{"Povezano: " + host}
	if diagnostics.Secure {
		parts = append(parts, "siguran prijenos")
	} else {
		parts = append(parts, "FTP bez enkripcije")
	}
	if diagnostics.WebRootDetected {
		parts = append(parts, "web root: "+diagnostics.WebRoot)
	} else if diagnostics.RootMode == "home" {
		parts = append(parts, "SFTP home spreman")
	} else {
		parts = append(parts, "account root spreman")
	}
	return strings.Join(parts, " • ")
}

func (a *app) onConnected(host string, diagnostics remote.ConnectionDiagnostics) {
	setText(a.pass, "")
	setText(a.passphrase, "")
	a.queuePaused = false
	a.setConnectionUI(true)
	a.setStatus(connectionDiagnosticStatus(host, diagnostics))
	remoteStart := "/"
	if a.protocolValue() == "sftp" {
		remoteStart = "."
	}
	localStart := ""
	if p, ok := a.currentProfile(); ok && a.currentEndpointMatchesProfile(p) {
		if strings.TrimSpace(p.RemotePath) != "" {
			remoteStart = p.RemotePath
		}
		localStart = p.LocalPath
	}
	if localStart != "" {
		a.refreshLocal(localStart)
	}
	a.refreshRemote(remoteStart)
	a.refreshTransfers()
}

func (a *app) finishDisconnected(status string) {
	if a.remoteNavCancel != nil {
		a.remoteNavCancel()
		a.remoteNavCancel = nil
	}
	a.remoteNavSeq++
	a.queuePaused = true
	a.setConnectionUI(false)
	clearList(a.remoteList)
	a.remoteItems = nil
	a.updateActionControls()
	if strings.TrimSpace(status) != "" {
		a.setStatus(status)
	}
}

func (a *app) disconnectNow() {
	if a.connectionBusy || !a.connected {
		return
	}
	if a.hasActiveTransfers() {
		if !platform.ConfirmDialog("GhostFTP — prekid veze", "Prekinuti vezu i aktivne prijenose?", "Svi prijenosi koji su na čekanju ili u tijeku bit će otkazani prije prekida veze.") {
			return
		}
	}
	generation := a.beginConnectionTransition()
	a.setConnectionBusy(true)
	a.setStatus("Prekid veze…")
	a.goSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err := a.engine.Disconnect(ctx)
		a.dispatch(func() {
			if generation != a.connectionGeneration {
				return
			}
			if err != nil {
				a.finishDisconnected(usererror.Message(err, "Veza je zatvorena uz upozorenje."))
			} else {
				a.finishDisconnected("Veza prekinuta.")
			}
		})
	})
}

func (a *app) choosePrivateKey() {
	p, err := a.engine.ChoosePrivateKey()
	if err != nil {
		platform.ErrorDialog("GhostFTP", "Odabir privatnog ključa nije uspio", usererror.Message(err, "Privatni ključ trenutačno nije moguće odabrati."))
		return
	}
	if p != "" {
		setText(a.keyPath, p)
	}
}

func (a *app) resetProfileCredentialCues() {
	cue(a.pass, "FTP / SFTP lozinka")
	cue(a.passphrase, "Zaporka privatnog ključa")
}

func (a *app) setProfileCredentialCues(p model.PublicProfile) {
	if p.HasPassword {
		cue(a.pass, "Spremljena lozinka — prazno koristi spremljenu")
	} else {
		cue(a.pass, "FTP / SFTP lozinka")
	}
	if p.HasPassphrase && p.PrivateKeyPath != "" {
		cue(a.passphrase, "Spremljena zaporka ključa — prazno koristi spremljenu")
	} else {
		cue(a.passphrase, "Zaporka privatnog ključa")
	}
}

func (a *app) loadProfiles() {
	a.goSafe(func() {
		profiles, err := a.engine.Profiles()
		a.dispatch(func() {
			a.applyProfiles(profiles, err)
		})
	})
}

func (a *app) applyProfiles(profiles []model.PublicProfile, loadErr error) {
	if loadErr != nil {
		a.setStatus(usererror.Message(loadErr, "Spremljene profile trenutačno nije moguće učitati."))
		return
	}
	a.profiles = profiles
	const cbResetContent = 0x014B
	sendMessageW.Call(a.profilesCombo, cbResetContent, 0, 0)
	sendMessageW.Call(a.profilesCombo, cbAddString, 0, uintptr(unsafe.Pointer(wstr("Brzi spoj (bez profila)"))))
	for _, p := range profiles {
		label := p.Name + " — " + p.Host
		sendMessageW.Call(a.profilesCombo, cbAddString, 0, uintptr(unsafe.Pointer(wstr(label))))
	}
	selected := 0
	var selectedProfile model.PublicProfile
	if a.selectedProfileID != "" {
		for i, p := range profiles {
			if p.ID == a.selectedProfileID {
				selected = i + 1
				selectedProfile = p
				break
			}
		}
	}
	if selected == 0 {
		a.selectedProfileID = ""
		a.resetProfileCredentialCues()
	} else {
		a.setProfileCredentialCues(selectedProfile)
	}
	sendMessageW.Call(a.profilesCombo, cbSetCurSel, uintptr(selected), 0)
	a.updateActionControls()
}

func (a *app) currentProfile() (model.PublicProfile, bool) {
	if a.selectedProfileID == "" {
		return model.PublicProfile{}, false
	}
	for _, p := range a.profiles {
		if p.ID == a.selectedProfileID {
			return p, true
		}
	}
	return model.PublicProfile{}, false
}

func (a *app) selectProfile() {
	if a.connectionBusy || a.connected {
		return
	}
	idx, _, _ := sendMessageW.Call(a.profilesCombo, cbGetCurSel, 0, 0)
	if idx == 0 || int(idx) > len(a.profiles) {
		a.selectedProfileID = ""
		setText(a.pass, "")
		setText(a.passphrase, "")
		a.resetProfileCredentialCues()
		a.updateActionControls()
		return
	}
	p := a.profiles[int(idx)-1]
	a.selectedProfileID = p.ID
	a.setProtocolValue(p.Protocol)
	setText(a.host, p.Host)
	setText(a.port, strconv.Itoa(p.Port))
	setText(a.user, p.Username)
	setText(a.pass, "")
	setText(a.keyPath, p.PrivateKeyPath)
	setText(a.passphrase, "")
	if p.LocalPath != "" {
		a.refreshLocal(p.LocalPath)
	}
	if p.RemotePath != "" {
		setText(a.remotePath, p.RemotePath)
	}
	a.setProfileCredentialCues(p)
	a.updateActionControls()
}

func (a *app) saveCurrentProfile() {
	if a.connected || a.connectionBusy {
		platform.InfoDialog("GhostFTP", "Profil nije moguće mijenjati tijekom veze", "Pričekajte završetak povezivanja ili prekinite vezu pa spremite promjene profila.")
		return
	}
	existing, editing := a.currentProfile()
	defaultName := strings.TrimSpace(getText(a.host))
	if editing {
		defaultName = existing.Name
	}
	name, ok := platform.PromptDialog("GhostFTP — profil", "Naziv profila:", defaultName)
	if !ok {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		platform.ErrorDialog("GhostFTP — profil", "Naziv profila nedostaje", "Upišite naziv po kojem ćete prepoznati ovu vezu.")
		return
	}
	protocol := a.protocolValue()
	host := getText(a.host)
	username := getText(a.user)
	port, err := validateRawConnectionInput(protocol, host, getText(a.port), username)
	if err != nil {
		if err == errInvalidConnectionPort {
			platform.ErrorDialog("GhostFTP — profil", "Neispravan port", "Port mora biti broj između 1 i 65535.")
		} else {
			platform.ErrorDialog("GhostFTP — profil", "Neispravni podaci veze", usererror.Message(err, "Provjerite poslužitelj, port i korisničko ime."))
		}
		return
	}
	keyPath := strings.TrimSpace(getText(a.keyPath))
	password := getText(a.pass)
	passphrase := getText(a.passphrase)
	clearPassword := false
	clearPassphrase := false

	if editing {
		sameAccount := profilebinding.AccountMatches(
			existing.Protocol, existing.Host, existing.Port, existing.Username,
			protocol, host, port, username,
		)
		samePrivateKey := profilebinding.PrivateKeyMatches(
			existing.Protocol, existing.Host, existing.Port, existing.Username, existing.PrivateKeyPath,
			protocol, host, port, username, keyPath,
		)
		clearPassword = existing.HasPassword && !sameAccount
		clearPassphrase = existing.HasPassphrase && !samePrivateKey
	}

	if password != "" || passphrase != "" {
		if !platform.ConfirmDialog(
			"GhostFTP — privatnost",
			"Spremiti vjerodajnice na ovom računalu?",
			"Da = upisane vjerodajnice spremit će se u GhostFTP spremište zaštićeno sustavom Windows.\nNe = profil će se spremiti bez spremljene lozinke i zaporke privatnog ključa.",
		) {
			password = ""
			passphrase = ""
			clearPassword = true
			clearPassphrase = true
		} else {
			if password != "" {
				clearPassword = false
			}
			if passphrase != "" {
				clearPassphrase = false
			}
		}
	} else if editing {
		retainPassword := existing.HasPassword && !clearPassword
		retainPassphrase := existing.HasPassphrase && !clearPassphrase
		autoRemoved := clearPassword || clearPassphrase
		if retainPassword || retainPassphrase {
			message := "Profil već sadrži spremljene vjerodajnice koje još pripadaju ovom identitetu.\n\nDa = zadrži ih.\nNe = ukloni ih iz GhostFTP spremišta."
			if autoRemoved {
				message = "Vjerodajnice koje više ne pripadaju novom poslužitelju, korisniku ili privatnom ključu bit će automatski uklonjene.\n\n" + message
			}
			if !platform.ConfirmDialog("GhostFTP — privatnost", "Zadržati spremljene vjerodajnice?", message) {
				if retainPassword {
					clearPassword = true
				}
				if retainPassphrase {
					clearPassphrase = true
				}
			}
		} else if autoRemoved {
			platform.InfoDialog(
				"GhostFTP — sigurnost profila",
				"Stare vjerodajnice neće se prenijeti",
				"Promijenili ste poslužitelj, port, korisničko ime ili privatni ključ. Radi zaštite stare spremljene vjerodajnice uklanjaju se iz ovog profila. Ako ih želite spremiti za novi identitet, upišite ih ponovno.",
			)
		}
	}

	payload := model.ProfileInput{
		ID:              a.selectedProfileID,
		Name:            name,
		Protocol:        protocol,
		Host:            host,
		Port:            port,
		Username:        username,
		Password:        password,
		ClearPassword:   clearPassword,
		PrivateKeyPath:  keyPath,
		Passphrase:      passphrase,
		ClearPassphrase: clearPassphrase,
		LocalPath:       a.localCurrent,
		RemotePath:      optionalRemotePath(getText(a.remotePath)),
	}
	setText(a.pass, "")
	setText(a.passphrase, "")
	password = ""
	passphrase = ""
	a.goSafe(func() {
		saved, err := a.engine.SaveProfile(payload)
		payload.Password = ""
		payload.Passphrase = ""
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("GhostFTP — profil", "Profil nije spremljen", usererror.Message(err, "Profil trenutačno nije moguće spremiti."))
				return
			}
			a.selectedProfileID = saved.ID
			setText(a.pass, "")
			setText(a.passphrase, "")
			a.setProfileCredentialCues(saved)
			a.setStatus("Profil spremljen: " + saved.Name)
			a.loadProfiles()
		})
	})
}

func (a *app) removeCurrentProfile() {
	if a.connectionBusy {
		return
	}
	p, ok := a.currentProfile()
	if !ok {
		platform.InfoDialog("GhostFTP", "Nije odabran spremljeni profil", "Odaberite profil koji želite obrisati.")
		return
	}
	if a.connected {
		platform.InfoDialog("GhostFTP", "Profil se ne briše tijekom veze", "Prvo prekinite vezu.")
		return
	}
	if !platform.ConfirmDialog("GhostFTP — profili", "Obrisati profil?", p.Name+"\n\nSpremljene vjerodajnice bit će uklonjene iz GhostFTP spremišta.") {
		return
	}
	id := p.ID
	a.goSafe(func() {
		err := a.engine.RemoveProfile(id)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("GhostFTP — profili", "Profil nije obrisan", usererror.Message(err, "Profil trenutačno nije moguće obrisati."))
				return
			}
			a.selectedProfileID = ""
			a.resetProfileCredentialCues()
			a.loadProfiles()
			a.setStatus("Profil obrisan.")
			a.updateActionControls()
		})
	})
}
