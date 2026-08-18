package usererror

import (
	"context"
	"errors"
	"strings"
)

// Message converts low-level protocol/OS/tooling errors into concise end-user
// messages. Low-level tooling details are intentionally not exposed in the
// normal desktop UI.
func Message(err error, fallback string) string {
	if err == nil {
		return ""
	}

	// Specifični lokalni lifecycle status mora imati prednost nad generičkim
	// context errorom. Engine može errors.Join-ati transfer deadline i remote
	// close timeout; u tom slučaju korisnik treba vidjeti što se stvarno događa
	// sa sesijom, a ne pogrešnu poruku da poslužitelj nije odgovorio.
	s := strings.ToLower(strings.Join(strings.Fields(err.Error()), " "))
	if strings.Contains(s, "prethodna veza se još sigurno zatvara") {
		return "Prethodna veza još se sigurno zatvara. Pokušajte ponovno za nekoliko trenutaka."
	}
	if strings.Contains(s, "sigurno zatvaranje veze još traje") {
		return "Prekid veze još se sigurno dovršava. Ponovno povezivanje bit će dostupno čim se stara sesija zatvori."
	}

	if errors.Is(err, context.Canceled) {
		return "Povezivanje ili operacija je otkazana."
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "Poslužitelj nije odgovorio na vrijeme. Provjerite adresu, port i mrežnu vezu pa pokušajte ponovno."
	}

	switch {
	case strings.Contains(s, "otisak sftp host ključa se promijenio") || strings.Contains(s, "fingerprint se promijenio") || strings.Contains(s, "host key verification failed"):
		return "Sigurnosni ključ poslužitelja promijenio se. Veza je blokirana radi vaše zaštite."
	case containsAny(s, "sftp podrška nije dostupna u sustavu windows", "sftp komponenta nije pronađena", "openssh client nije instaliran", "nedostaje sftp.exe", "nedostaje ssh-keyscan.exe", "nedostaje ssh-keygen.exe"):
		return "SFTP podrška nije dostupna. U postavkama sustava Windows uključite značajku OpenSSH Client."
	case containsAny(s, "nije moguće dohvatiti sftp host ključ", "poslužitelj nije vratio ssh host ključ"):
		return "SFTP poslužitelj nije vratio sigurnosni host ključ. Provjerite adresu, port i je li SSH/SFTP servis pokrenut."
	case containsAny(s, "authentication failed", "permission denied (publickey", "permission denied (password", "permission denied, please try again", "login incorrect", "access denied", "530 login", "530 user", "530 not logged", "authentication rejected"):
		return "Prijava nije prihvaćena. Provjerite korisničko ime, lozinku, SSH ključ ili zaporku ključa."
	case containsAny(s, "could not resolve host", "name or service not known", "temporary failure in name resolution", "no such host", "host not found"):
		return "Poslužitelj nije pronađen. Provjerite adresu poslužitelja."
	case containsAny(s, "connection refused", "actively refused"):
		return "Poslužitelj odbija vezu na odabranom portu. Provjerite protokol i port."
	case containsAny(s, "timed out", "timeout", "operation timed out"):
		return "Poslužitelj nije odgovorio na vrijeme. Provjerite adresu, port i mrežnu vezu pa pokušajte ponovno."
	case containsAny(s, "connection reset", "connection closed", "broken pipe", "connection aborted", "network is unreachable", "no route to host"):
		return "Veza s poslužiteljem je prekinuta. Povežite se ponovno i ponovite operaciju."
	case containsAny(s, "certificate", "ssl certificate", "tls", "schannel"):
		return "Sigurnu vezu nije moguće potvrditi. Provjerite certifikat i FTPS postavke poslužitelja."
	case containsAny(s, "disk full", "no space left", "insufficient disk space", "quota exceeded", "552"):
		return "Nema dovoljno prostora za dovršetak operacije."
	case containsAny(s, "permission denied", "access is denied", "550 permission", "553 permission"):
		return "Nemate dopuštenje za ovu datoteku ili mapu."
	case containsAny(s, "no such file", "not found", "550 file unavailable", "550 failed to open"):
		return "Datoteka ili mapa više nije dostupna. Osvježite prikaz i pokušajte ponovno."
	case containsAny(s, "already exists", "file exists", "ciljna stavka već postoji"):
		return "Stavka s tim nazivom već postoji."
	case strings.Contains(s, "nije uspostavljena veza"):
		return "Niste povezani s poslužiteljem."
	case strings.Contains(s, "red prijenosa je prevelik"):
		return "Red prijenosa je pun. Očistite završene prijenose pa pokušajte ponovno."
	case strings.Contains(s, "previše stavki") || strings.Contains(s, "predubok"):
		return "Odabrana struktura je prevelika za jednu sigurnu operaciju."
	case strings.Contains(s, "neispravan port") || strings.Contains(s, "port mora biti"):
		return "Port mora biti broj između 1 i 65535."
	case strings.Contains(s, "neispravan poslužitelj"):
		return "Adresa poslužitelja nije ispravna."
	case strings.Contains(s, "neispravno korisničko ime"):
		return "Korisničko ime nije ispravno."
	case strings.Contains(s, "neispravan naziv datoteke ili mape"):
		return "Naziv datoteke ili mape nije dopušten."
	}

	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		fallback = "Operacija nije uspjela. Provjerite vezu i pokušajte ponovno."
	}
	return fallback
}

func containsAny(s string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(s, value) {
			return true
		}
	}
	return false
}
