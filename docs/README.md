# ByFTP dokumentacija

Ova mapa sadrži detaljnu produkcijsku dokumentaciju za novu ByFTP 1.x liniju. Root repozitorija ostaje fokusiran na korisnički pregled, kanonsku verziju, licencu, CHANGELOG i glavne build ulaze.

## Sadržaj

- [INSTALACIJA.md](INSTALACIJA.md) — Windows x64/x86 Setup i Portable, instalacijski lifecycle, nadogradnja, standardno uklanjanje i rješavanje instalacijskih problema
- [KLIJENTI.md](KLIJENTI.md) — ByFTP All-in-One, FTP Client, SFTP Client, SSH Client i S3 Client; granice i zajednička jezgra
- [ARHITEKTURA.md](ARHITEKTURA.md) — slojevi aplikacije, client mode, Engine/remote/transfer, SSH/S3 adapteri, state i release arhitektura
- [SIGURNOST.md](SIGURNOST.md) — threat model, FTP/FTPS/SFTP/SSH/S3 sigurnosne granice, vjerodajnice, filesystem i release zaštite
- [PRIVATNOST.md](PRIVATNOST.md) — mrežna politika, lokalne tajne, DPAPI, S3/SSH tajne, build telemetrija i odsutnost aplikacijske telemetrije
- [TESTIRANJE.md](TESTIRANJE.md) — unit/race/vet, process-level connect smoke, S3/SSH regresije, Windows PE provjere i produkcijski CI
- [PROVJERA-IZDANJA.md](PROVJERA-IZDANJA.md) — fail-closed kontrolna lista za 12 javnih EXE datoteka, tag, Release i pet GitHub Packages paketa
- [IZDAVANJE-NA-GITHUBU.md](IZDAVANJE-NA-GITHUBU.md) — reset stare release linije za 1.0.0, serijalizirani publisher i Packages objava
- [GITHUB-PACKAGES.md](GITHUB-PACKAGES.md) — pet package linija, verzijska sinkronizacija i obavezna provjera najnovije objavljene verzije
- [POTPISIVANJE.md](POTPISIVANJE.md) — Authenticode granice i što je potrebno za stvarni Verified Publisher
- [PODRSKA.md](PODRSKA.md) — podržani klijenti, preduvjeti, povezivanje, dijagnostika i sigurna prijava greške
- [PLAN-RAZVOJA.md](PLAN-RAZVOJA.md) — završeno u 1.0.0 i prioriteti sljedećih 1.x izdanja
- [DOPRINOS.md](DOPRINOS.md) — pravila ovlaštenih doprinosa i obavezni production gateovi
- [OBAVIJESTI-TRECIH-STRANA.md](OBAVIJESTI-TRECIH-STRANA.md) — sistemski curl/OpenSSH, WinSCP referenca/licenca i komponente trećih strana
- [slike/](slike/) — službeni dokumentacijski vizualni resursi

## Produkcijska načela

Dokumentacija mora opisivati ono što stvarni kod, build i javni Release zaista rade. Posebno:

- aktivna javna verzijska linija počinje s `1.0.0`;
- GitHub Release sadrži točno 12 ByFTP EXE asseta, dok GitHub automatski prikazuje standardne source archive poveznice;
- interna komponenta uklanjanja pripada Setup payloadu i nije javni Release asset;
- FTP i SFTP klijenti imaju zasebne profile/state i strogi protokolni allowlist;
- SSH Client koristi sistemski OpenSSH i ne sprema interaktivnu SSH lozinku/MFA/passphrase;
- S3 Client koristi vlastiti standard-library SigV4 core i u 1.0.0 ne predstavlja multipart upload kao dovršenu funkciju;
- stanje `POVEZANO` u file-transfer GUI-ju znači uspješnu autentikaciju i stvarni početni udaljeni probe;
- digitalni publisher identitet se ne tvrdi bez stvarnog certifikata;
- Go build telemetrija mora biti stvarno `off` u produkcijskom build/release okruženju;
- svaki službeni release mora objaviti istu verziju u svih pet GitHub Packages paketa;
- GitHub Packages nisu zasebna verzijska linija: njihova najnovija objavljena verzija mora odgovarati `VERSION` i najnovijem službenom Release tagu.

## Automatizirana konzistentnost

`scripts/audit_docs.py` provjerava lokalne Markdown/HTML poveznice, zahtijeva da svaki detaljni dokument bude indeksiran ovdje i u glavnom README-u te blokira verzionirane naslove za dokumente koji moraju ostati aktualni između izdanja.

`scripts/audit_security.py`, `scripts/audit_privacy.py`, `scripts/audit_version.py` i `scripts/audit_release.py` zaključavaju dokumentirani sigurnosni, privatnosni, verzijski i distribucijski ugovor prema stvarnom kodu/workflowu.

Povijest aktivne 1.x linije je u [`CHANGELOG.md`](../CHANGELOG.md), a licencni uvjeti u [`LICENSE`](../LICENSE).
