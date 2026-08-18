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
	if got != "Poslužitelj nije odgovorio na vrijeme. Pokušajte ponovno." {
		t.Fatalf("unexpected message: %q", got)
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
