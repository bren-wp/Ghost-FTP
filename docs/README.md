# ByFTP dokumentacija

Ova mapa je jedino mjesto za detaljnu projektnu dokumentaciju. Root repozitorija ostaje fokusiran na produkcijski pregled, licencu, povijest promjena, kanonsku verziju i glavne build ulaze.

## Sadržaj

- [INSTALACIJA.md](INSTALACIJA.md) — odabir paketa, Windows x64/x86, Linux DEB, macOS Universal PKG, instalacija, nadogradnja, uklanjanje, prvi spoj i SHA-256 provjera
- [ARHITEKTURA.md](ARHITEKTURA.md) — zajednički Engine, platformni frontendovi, remote/transfer/session lifecycle, build i release arhitektura
- [SIGURNOST.md](SIGURNOST.md) — model povezivanja, SFTP/AskPass, vjerodajnice, filesystem, build i release sigurnosne granice
- [PRIVATNOST.md](PRIVATNOST.md) — runtime mrežna politika, lokalne tajne, Go build telemetrija i release provenance podaci
- [TESTIRANJE.md](TESTIRANJE.md) — unit/race/vet, process-level connect smoke, auditi, platformni CI i production release quality gate
- [PROVJERA-IZDANJA.md](PROVJERA-IZDANJA.md) — fail-closed kontrolna lista za produkcijsku matricu, staging, 13 javnih custom asseta, checksumove i potpise
- [POTPISIVANJE.md](POTPISIVANJE.md) — Windows Authenticode i macOS Developer ID/notarization granice
- [PLAN-RAZVOJA.md](PLAN-RAZVOJA.md) — završene cjeline i sljedeći prioriteti bez lažnog označavanja nedovršenih platformnih značajki
- [PODRSKA.md](PODRSKA.md) — podržane platforme, auth matrica, dijagnostika, instalacijski problemi i sigurna prijava greške
- [DOPRINOS.md](DOPRINOS.md) — pravila ovlaštenih doprinosa i obavezni quality gateovi
- [OBAVIJESTI-TRECIH-STRANA.md](OBAVIJESTI-TRECIH-STRANA.md) — sistemski curl/OpenSSH i ostale komponente trećih strana
- [IZDAVANJE-NA-GITHUBU.md](IZDAVANJE-NA-GITHUBU.md) — jedan release okidač, serijalizacija, production quality/race gate, platformski artefakti i fail-closed publisher
- [slike/](slike/) — službeni dokumentacijski vizualni resursi

## Produkcijska načela dokumentacije

Dokumentacija mora opisivati ono što stvarni kod, CI i release ugovor doista podržavaju. Posebno:

- Windows, Linux i macOS imaju različite UI/auth granice koje se ne skrivaju
- stanje `Connected` znači uspješnu autentikaciju i početni udaljeni probe
- interni build dokazi nisu predstavljeni kao korisnički distribucijski paketi
- digitalni publisher identitet se ne tvrdi bez stvarnog certifikata
- Go build telemetrija mora biti stvarno `off` u produkcijskom build/release okruženju
- javni release nastaje tek nakon vlastitog production quality/race gatea i svih platformskih buildova

## Automatizirana konzistentnost

`scripts/audit_docs.py` provjerava lokalne Markdown/HTML poveznice, zahtijeva da svaki detaljni dokument bude indeksiran ovdje i u glavnom README-u te blokira verzionirane naslove za dokumente koji moraju ostati aktualni između izdanja.

`scripts/audit_security.py`, `scripts/audit_privacy.py` i `scripts/audit_release.py` dodatno zaključavaju da dokumentirani runtime, build-privacy i release model ne mogu neprimjetno odstupiti od stvarnog koda i workflowa.

Povijest izdanja ostaje u root datoteci [`CHANGELOG.md`](../CHANGELOG.md), a licenca u [`LICENSE`](../LICENSE).
