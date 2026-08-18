//go:build windows

package desktop

import (
	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/profilebinding"
	"brendigo.com/byftp/internal/usererror"
	"context"
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
		a.keyPath, a.chooseKey, a.passphrase,
	} {
		enableWindow.Call(h, 0)
	}
	a.setRemoteControls(false)
}

func (a *app) setConnectionUI(connected bool) {
	a.connected = connected
	a.connectionBusy = false
	connectedVal, disconnectedVal := uintptr(0), uintptr(1)
	if connected {
		connectedVal, disconnectedVal = 1, 0
	}
	enableWindow.Call(a.connect, disconnectedVal)
	enableWindow.Call(a.disconnect, connectedVal)
	for _, h := range []uintptr{a.profilesCombo, a.protocol, a.host, a.port, a.user, a.pass, a.keyPath, a.chooseKey, a.passphrase} {
		enableWindow.Call(h, disconnectedVal)
	}
	a.setRemoteControls(connected)
	if connected {
		setText(a.connectionBadge, "● POVEZANO")
	} else {
		setText(a.connectionBadge, "● NIJE POVEZANO")
		a.updateProtocolControls()
	}
	invalidateRect.Call(a.hwnd, 0, 1)
}

func (a *app) connectNow() {
	host := strings.TrimSpace(getText(a.host))
	user := strings.TrimSpace(getText(a.user))
	password := getText(a.pass)
	port, err := strconv.Atoi(strings.TrimSpace(getText(a.port)))
	if err != nil || port < 1 || port > 65535 {
		platform.ErrorDialog("ByFTP", "Neispravan port", "Port mora biti broj između 1 i 65535.")
		return
	}
	cfg := model.ConnectionConfig{
		Protocol:       a.protocolValue(),
		Host:           host,
		Port:           port,
		Username:       user,
		Password:       password,
		PrivateKeyPath: strings.TrimSpace(getText(a.keyPath)),
		Passphrase:     getText(a.passphrase),
	}
	if host == "" || user == "" {
		platform.ErrorDialog("ByFTP", "Nedostaju podaci", "Upišite poslužitelj i korisničko ime.")
		return
	}
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
			a.connectionCancel = nil
			if err != nil {
				a.setConnectionUI(false)
				a.setStatus(usererror.Message(err, "Povezivanje nije uspjelo. Provjerite podatke i pokušajte ponovno."))
				platform.ErrorDialog("ByFTP — povezivanje", "Povezivanje nije uspjelo", usererror.Message(err, "Provjerite podatke za prijavu i mrežnu vezu."))
				return
			}
			if r.RequiresTrust {
				if !platform.ConfirmDialog("ByFTP — SFTP sigurnost", "Novi SFTP ključ poslužitelja", "Provjerite otisak sigurnosnog ključa prije prihvaćanja:\n\n"+r.Fingerprint+"\n\nVjerujete li ovom poslužitelju?") {
					a.engine.CancelPendingTrust()
					a.setConnectionUI(false)
					a.setStatus("SFTP povezivanje otkazano.")
					return
				}
				a.connectTrusted(profileID, cfg, r.Fingerprint)
				return
			}
			a.onConnected(host)
		})
	})
}

func (a *app) connectTrusted(profileID string, cfg model.ConnectionConfig, fingerprint string) {
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
		_, err := a.engine.Connect(ctx, profileID, cfg, fingerprint, profileID != "")
		cfg.Password = ""
		cfg.Passphrase = ""
		a.dispatch(func() {
			a.connectionCancel = nil
			if err != nil {
				a.setConnectionUI(false)
				a.setStatus(usererror.Message(err, "SFTP povezivanje nije uspjelo."))
				platform.ErrorDialog("ByFTP — SFTP", "Povezivanje nije uspjelo", usererror.Message(err, "Provjerite podatke za prijavu i SFTP postavke."))
				return
			}
			a.onConnected(cfg.Host)
		})
	})
}

func (a *app) currentEndpointMatchesProfile(p model.PublicProfile) bool {
	port, err := strconv.Atoi(strings.TrimSpace(getText(a.port)))
	if err != nil {
		return false
	}
	return profilebinding.EndpointMatches(
		p.Protocol, p.Host, p.Port,
		a.protocolValue(), strings.TrimSpace(getText(a.host)), port,
	)
}

func (a *app) onConnected(host string) {
	setText(a.pass, "")
	setText(a.passphrase, "")
	a.setConnectionUI(true)
	a.setStatus("Povezano: " + host)
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
}

func (a *app) disconnectNow() {
	if a.hasActiveTransfers() {
		if !platform.ConfirmDialog("ByFTP — prekid veze", "Prekinuti vezu i aktivne prijenose?", "Svi prijenosi koji su na čekanju ili u tijeku bit će otkazani prije prekida veze.") {
			return
		}
	}
	a.setConnectionBusy(true)
	a.setStatus("Prekid veze…")
	a.goSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err := a.engine.Disconnect(ctx)
		a.dispatch(func() {
			if a.remoteNavCancel != nil {
				a.remoteNavCancel()
				a.remoteNavCancel = nil
			}
			a.remoteNavSeq++
			a.setConnectionUI(false)
			clearList(a.remoteList)
			a.remoteItems = nil
			if err != nil {
				a.setStatus(usererror.Message(err, "Veza je zatvorena uz upozorenje."))
			} else {
				a.setStatus("Veza prekinuta.")
			}
		})
	})
}

func (a *app) choosePrivateKey() {
	p, err := a.engine.ChoosePrivateKey()
	if err != nil {
		platform.ErrorDialog("ByFTP", "Odabir privatnog ključa nije uspio", usererror.Message(err, "Privatni ključ trenutačno nije moguće odabrati."))
		return
	}
	if p != "" {
		setText(a.keyPath, p)
	}
}

func (a *app) resetProfileCredentialCues() {
	cue(a.pass, "Lozinka")
	cue(a.passphrase, "Zaporka privatnog ključa")
}

func (a *app) setProfileCredentialCues(p model.PublicProfile) {
	if p.HasPassword {
		cue(a.pass, "Spremljena lozinka — prazno koristi spremljenu")
	} else {
		cue(a.pass, "Lozinka")
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
	idx, _, _ := sendMessageW.Call(a.profilesCombo, cbGetCurSel, 0, 0)
	if idx == 0 || int(idx) > len(a.profiles) {
		a.selectedProfileID = ""
		setText(a.pass, "")
		setText(a.passphrase, "")
		a.resetProfileCredentialCues()
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
}

func (a *app) saveCurrentProfile() {
	if a.connected {
		platform.InfoDialog("ByFTP", "Profil nije moguće mijenjati tijekom veze", "Prekinite vezu pa spremite promjene profila.")
		return
	}
	existing, editing := a.currentProfile()
	defaultName := strings.TrimSpace(getText(a.host))
	if editing {
		defaultName = existing.Name
	}
	name, ok := platform.PromptDialog("ByFTP — profil", "Naziv profila:", defaultName)
	if !ok {
		return
	}
	protocol := a.protocolValue()
	host := strings.TrimSpace(getText(a.host))
	username := strings.TrimSpace(getText(a.user))
	keyPath := strings.TrimSpace(getText(a.keyPath))
	port, _ := strconv.Atoi(strings.TrimSpace(getText(a.port)))
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
			"ByFTP — privatnost",
			"Spremiti vjerodajnice na ovom računalu?",
			"Da = upisane vjerodajnice spremit će se u ByFTP spremište zaštićeno sustavom Windows.\nNe = profil će se spremiti bez spremljene lozinke i zaporke privatnog ključa.",
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
			message := "Profil već sadrži spremljene vjerodajnice koje još pripadaju ovom identitetu.\n\nDa = zadrži ih.\nNe = ukloni ih iz ByFTP spremišta."
			if autoRemoved {
				message = "Vjerodajnice koje više ne pripadaju novom poslužitelju, korisniku ili privatnom ključu bit će automatski uklonjene.\n\n" + message
			}
			if !platform.ConfirmDialog("ByFTP — privatnost", "Zadržati spremljene vjerodajnice?", message) {
				if retainPassword {
					clearPassword = true
				}
				if retainPassphrase {
					clearPassphrase = true
				}
			}
		} else if autoRemoved {
			platform.InfoDialog(
				"ByFTP — sigurnost profila",
				"Stare vjerodajnice neće se prenijeti",
				"Promijenili ste poslužitelj, port, korisničko ime ili privatni ključ. Radi zaštite stare spremljene vjerodajnice uklanjaju se iz ovog profila. Ako ih želite spremiti za novi identitet, upišite ih ponovno.",
			)
		}
	}

	payload := model.ProfileInput{
		ID:              a.selectedProfileID,
		Name:            strings.TrimSpace(name),
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
				platform.ErrorDialog("ByFTP — profil", "Profil nije spremljen", usererror.Message(err, "Profil trenutačno nije moguće spremiti."))
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
	p, ok := a.currentProfile()
	if !ok {
		platform.InfoDialog("ByFTP", "Nije odabran spremljeni profil", "Odaberite profil koji želite obrisati.")
		return
	}
	if a.connected {
		platform.InfoDialog("ByFTP", "Profil se ne briše tijekom veze", "Prvo prekinite vezu.")
		return
	}
	if !platform.ConfirmDialog("ByFTP — profili", "Obrisati profil?", p.Name+"\n\nSpremljene vjerodajnice bit će uklonjene iz ByFTP spremišta.") {
		return
	}
	id := p.ID
	a.goSafe(func() {
		err := a.engine.RemoveProfile(id)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("ByFTP — profili", "Profil nije obrisan", usererror.Message(err, "Profil trenutačno nije moguće obrisati."))
				return
			}
			a.selectedProfileID = ""
			a.resetProfileCredentialCues()
			a.loadProfiles()
			a.setStatus("Profil obrisan.")
		})
	})
}
