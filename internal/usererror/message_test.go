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

func TestMessageFallback(t *testing.T) {
	got := Message(errors.New("opaque-internal-tool-error"), "Radnja nije uspjela.")
	if got != "Radnja nije uspjela." {
		t.Fatalf("unexpected message: %q", got)
	}
}
