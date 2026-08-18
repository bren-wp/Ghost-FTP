package usererror

import (
	"context"
	"errors"
	"testing"
)

func TestMessageHidesToolDetails(t *testing.T) {
	got := Message(errors.New(`curl: (67) Login denied; server said: 530 Login incorrect`), "Povezivanje nije uspjelo.")
	if got != "Prijava nije prihvaćena. Provjerite korisničko ime, lozinku, SSH ključ ili zaporku ključa." {
		t.Fatalf("unexpected message: %q", got)
	}
	if got == "" || got == `curl: (67) Login denied; server said: 530 Login incorrect` {
		t.Fatal("low-level details leaked")
	}
}

func TestMessageDeadline(t *testing.T) {
	got := Message(context.DeadlineExceeded, "x")
	if got != "Poslužitelj nije odgovorio na vrijeme. Provjerite adresu, port i mrežnu vezu pa pokušajte ponovno." {
		t.Fatalf("unexpected message: %q", got)
	}
}

func TestMessageMissingSFTPComponent(t *testing.T) {
	got := Message(errors.New("SFTP podrška nije dostupna u sustavu Windows"), "x")
	if got != "SFTP podrška nije dostupna. U postavkama sustava Windows uključite značajku OpenSSH Client." {
		t.Fatalf("unexpected SFTP component message: %q", got)
	}
}

func TestMessageSFTPHostKeyScanFailure(t *testing.T) {
	got := Message(errors.New("poslužitelj nije vratio SSH host ključ"), "x")
	if got != "SFTP poslužitelj nije vratio sigurnosni host ključ. Provjerite adresu, port i je li SSH/SFTP servis pokrenut." {
		t.Fatalf("unexpected host-key scan message: %q", got)
	}
}

func TestMessageSessionStillClosing(t *testing.T) {
	got := Message(errors.New("prethodna veza se još sigurno zatvara"), "x")
	if got != "Prethodna veza još se sigurno zatvara. Pokušajte ponovno za nekoliko trenutaka." {
		t.Fatalf("unexpected message: %q", got)
	}
}

func TestMessageDisconnectCleanupStillRunning(t *testing.T) {
	got := Message(errors.New("sigurno zatvaranje veze još traje"), "x")
	if got != "Prekid veze još se sigurno dovršava. Ponovno povezivanje bit će dostupno čim se stara sesija zatvori." {
		t.Fatalf("unexpected message: %q", got)
	}
}

func TestMessageDisconnectLifecycleWinsJoinedDeadline(t *testing.T) {
	err := errors.Join(context.DeadlineExceeded, errors.New("sigurno zatvaranje veze još traje"))
	got := Message(err, "x")
	if got != "Prekid veze još se sigurno dovršava. Ponovno povezivanje bit će dostupno čim se stara sesija zatvori." {
		t.Fatalf("unexpected joined-error message: %q", got)
	}
}

func TestMessageFallback(t *testing.T) {
	got := Message(errors.New("opaque-internal-tool-error"), "Radnja nije uspjela.")
	if got != "Radnja nije uspjela." {
		t.Fatalf("unexpected message: %q", got)
	}
}
