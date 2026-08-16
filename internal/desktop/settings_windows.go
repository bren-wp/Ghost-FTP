//go:build windows

package desktop

import (
	"brendigo.com/byftp/internal/brand"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/usererror"
	"strconv"
	"strings"
)

func (a *app) loadSettings() {
	// Osiguraj hrvatski cue tekst i kod nadogradnje sa starijeg UI resursa.
	cue(a.host, "Poslužitelj")
	a.goSafe(func() {
		settings, err := a.engine.Settings()
		if err != nil {
			return
		}
		a.dispatch(func() { a.settings = settings })
	})
}

func promptNumber(title, instruction string, current, min, max int) (int, bool) {
	v, ok := platform.PromptDialog(title, instruction, strconv.Itoa(current))
	if !ok {
		return current, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < min || n > max {
		platform.ErrorDialog(title, "Neispravna vrijednost", "Unesite broj od "+strconv.Itoa(min)+" do "+strconv.Itoa(max)+".")
		return current, false
	}
	return n, true
}

func (a *app) openSettings() {
	settings := a.settings
	if settings.Parallelism < 1 {
		settings.Parallelism = 2
	}
	if settings.RetryDelaySeconds < 1 {
		settings.RetryDelaySeconds = 3
	}
	if settings.ConnectionTimeoutSeconds < 5 {
		settings.ConnectionTimeoutSeconds = 15
	}

	parallel, ok := promptNumber("ByFTP — postavke", "Broj paralelnih prijenosa (1–8):", settings.Parallelism, 1, 8)
	if !ok {
		return
	}
	settings.Parallelism = parallel

	connectTimeout, ok := promptNumber("ByFTP — postavke", "Vrijeme čekanja pri spajanju (5–60 sekundi):", settings.ConnectionTimeoutSeconds, 5, 60)
	if !ok {
		return
	}
	settings.ConnectionTimeoutSeconds = connectTimeout

	retries, ok := promptNumber("ByFTP — postavke", "Automatski ponoviti neuspjeli prijenos (0–3 puta):", settings.AutoRetryCount, 0, 3)
	if !ok {
		return
	}
	settings.AutoRetryCount = retries
	if retries > 0 {
		delay, ok := promptNumber("ByFTP — postavke", "Pauza između automatskih pokušaja (1–30 sekundi):", settings.RetryDelaySeconds, 1, 30)
		if !ok {
			return
		}
		settings.RetryDelaySeconds = delay
	}

	settings.BackupBeforeOverwrite = platform.ConfirmDialog(
		"ByFTP — postavke",
		"Sigurnosna kopija prije prepisivanja?",
		"Da = ByFTP zadržava sigurnosnu kopiju postojeće datoteke.\nNe = privremena zaštitna kopija uklanja se nakon uspješnog prijenosa.",
	)
	settings.SkipExisting = platform.ConfirmDialog(
		"ByFTP — postavke",
		"Preskočiti datoteke koje već postoje?",
		"Da = postojeće odredišne datoteke neće se prepisivati.\nNe = ByFTP će ih sigurno zamijeniti prema postavci sigurnosne kopije.",
	)
	settings.ConfirmDelete = platform.ConfirmDialog(
		"ByFTP — postavke",
		"Tražiti potvrdu prije brisanja?",
		"Preporučeno je ostaviti ovu opciju uključenu za lokalne i udaljene datoteke.",
	)
	a.goSafe(func() {
		saved, err := a.engine.SetSettings(settings)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("ByFTP — postavke", "Postavke nisu spremljene", usererror.Message(err, "Postavke trenutačno nije moguće spremiti."))
				return
			}
			a.settings = saved
			retryText := "bez automatskog ponavljanja"
			if saved.AutoRetryCount > 0 {
				retryText = "automatska ponavljanja: " + strconv.Itoa(saved.AutoRetryCount)
			}
			overwriteText := "prepisivanje uključeno"
			if saved.SkipExisting {
				overwriteText = "postojeće datoteke se preskaču"
			}
			a.setStatus("Postavke spremljene. Paralelni prijenosi: " + strconv.Itoa(saved.Parallelism) + " • spajanje: " + strconv.Itoa(saved.ConnectionTimeoutSeconds) + " s • " + retryText + " • " + overwriteText)
		})
	})
}

func (a *app) openAbout() {
	platform.InfoDialog(
		brand.ProductName+" — O programu",
		brand.ProductFull+" "+a.version,
		"Siguran i jednostavan prijenos datoteka putem FTP, FTPS i SFTP veze.\n\n"+
			"ByFTP ne prati korisnika i ne šalje unesene podatke trećim stranama.\n"+
			"Spremljeni profili ostaju samo na ovom računalu i zaštićeni su sustavom Windows.\n\n"+
			"Brendigo\n"+brand.Website+"\n"+brand.Support,
	)
}
