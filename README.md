<p align="center">
  <img src="docs/slike/byftp-zaglavlje.png" alt="ByFTP — siguran prijenos datoteka" width="900">
</p>

<p align="center">
  <strong>Brz, privatan i izvorni FTP / FTPS / SFTP klijent za Windows.</strong><br>
  ByFTP je fokusirani desktop alat tvrtke Brendigo bez browser sučelja, telemetrije ili obaveznog cloud računa.
</p>

<p align="center">
  <a href="https://github.com/bren-wp/by-ftp/releases"><strong>Izdanja</strong></a> ·
  <a href="https://github.com/users/bren-wp/packages?repo_name=by-ftp"><strong>Paketi</strong></a> ·
  <a href="LICENSE"><strong>Licenca</strong></a> ·
  <a href="docs/PRIVATNOST.md"><strong>Privatnost</strong></a> ·
  <a href="docs/SIGURNOST.md"><strong>Sigurnost</strong></a>
</p>

<p align="center">
  <a href="../../actions/workflows/ci.yml"><img alt="CI" src="../../actions/workflows/ci.yml/badge.svg"></a>
</p>

## Prenosite datoteke. Zadržite kontrolu.

ByFTP je izvorni Windows x64 klijent za **FTP, FTPS i SFTP**. Spaja dvopanelni upravitelj datotekama s učvršćenim transfer engineom, skupnim operacijama i arhitekturom usmjerenom na privatnost.

**Trenutačno izdanje: 2.14.3**

## Preuzimanje

Preporučeni kanal su [GitHub izdanja](https://github.com/bren-wp/by-ftp/releases). Službeno izdanje sadrži:

- `ByFTP-<verzija>-Setup-x64.exe`
- `ByFTP-<verzija>-Portable-x64.exe`
- `ByFTP-<verzija>-Uninstall-x64.exe`
- `ByFTP-<verzija>-Windows-x64.zip`
- `ByFTP-<verzija>-Source.zip`
- `SHA256.txt`
- `verification.txt`
- `RELEASE-NOTES.txt`
- `BUILD-METADATA.txt`

GitHub Packages je dodatni paketni/arhivski kanal. Službena su samo neizmijenjena izdanja koja Brendigo objavi kroz službeni ByFTP/Brendigo kanal.

## Mogućnosti

### Veze

- FTP
- FTPS — eksplicitni i implicitni način
- SFTP
- autentikacija lozinkom ili privatnim ključem
- potvrda i pinning SFTP host ključa
- podesivo vrijeme čekanja veze
- lokalni profili zaštićeni Windows DPAPI mehanizmom

### Upravitelj datotekama

- dvopanelni lokalni/udaljeni prikaz
- višestruki odabir, dvoklik i brzi prijenos
- pojedinačne datoteke i cijela stabla mapa
- stvaranje mapa, preimenovanje, brisanje, osvježavanje i CHMOD
- zaštita od traversal putanja, Windows rezerviranih naziva, symlinkova, junctiona i reparse-point izlaza
- kontrolirana lokalna enumeracija do 50.000 stavki

### Red prijenosa

- 1–8 paralelnih prijenosa
- pause/resume, skupni cancel i retry
- automatski retry samo prolaznih mrežnih grešaka
- preskakanje postojećih datoteka
- cross-server retry blokada
- runtime revalidacija lokalnog korijena
- zaštita rekurzivnog uploada od kasne zamjene roota
- autoritativni završni status koji late cancel ne može prepisati nakon stvarnog uspjeha

## Što donosi 2.14.3

- GitHub Release objava je idempotentna i fail-closed: tag mora pokazivati na točan commit, a postojeći asset mora imati istu veličinu i SHA-256 digest
- djelomično izdanje može se sigurno dovršiti ponovnim runom bez slijepog `--clobber` prepisivanja potvrđenih asseta
- konačni Windows ZIP ponovno se provjerava nakon kompresije, uključujući putanje, duplikate, obavezne datoteke i svaki `BUNDLE-SHA256.txt` zapis
- uvedeni su zasebni docs, security i release auditi te Python regresije release alata
- hrvatski i version auditi više ne trebaju ručno sinkronizirane produkcijske verzije
- CI, Windows build i lokalni build koriste isti prošireni skup quality gateova
- Windows ZIP i GitHub Package uključuju kompletnu Markdown dokumentaciju
- detaljna dokumentacija postala je verzijski neutralna i automatski se provjerava za pokvarene poveznice i nepotpun indeks

## Sigurnost i privatnost

ByFTP namjerno nema telemetriju, analitiku, oglašavanje, automatski update API, trajni runtime log ni browser/localhost upravljački server. Normalan mrežni promet usmjeren je prema FTP/FTPS/SFTP poslužitelju koji korisnik odabere.

Ključne zaštite uključuju:

- SFTP host-key pinning i izolirani session trust
- Windows curl/OpenSSH iz System32
- DPAPI zaštitu spremljenih osjetljivih podataka
- sanitizirani AskPass bez credential datoteke
- state safe-open provjeru stvarno otvorenog regularnog objekta
- kriptografski nasumične staging nazive
- download `Lstat`/regular-file/reparse provjeru prije atomske aktivacije
- no-follow rekurzivno brisanje s filesystem-root, depth i item granicama
- transakcijski staging/rollback i no-replace rename
- connection-generation i cross-server retry izolaciju
- duboke kopije transfer događaja
- offline produkcijske buildove bez vanjskih Go modula

Detalji: [Sigurnost](docs/SIGURNOST.md) i [Privatnost](docs/PRIVATNOST.md).

## Zahtjevi

### Pokretanje

- Windows 10 ili Windows 11 x64
- sistemski `curl.exe` za FTP/FTPS
- Windows OpenSSH Client za SFTP

### Izgradnja

- Go **1.26.5+**
- Python 3
- Windows x64 za puni produkcijski build

Kanonska verzija nalazi se isključivo u [`VERSION`](VERSION).

```powershell
.\BUILD-WINDOWS.ps1
```

Lokalna Unix/Linux Windows cross-build provjera:

```bash
./scripts/BUILD-LOCAL.sh
```

Produkcijski build koristi `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off` i `GOTELEMETRY=off`.

## Provjere kvalitete

```text
python scripts/generate_brand_assets.py --check
python scripts/audit_croatian.py
python scripts/audit_version.py
python scripts/audit_docs.py
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_release.py
python -m unittest discover -s scripts -p 'test_*.py'
go test ./...
go test -race ./...
go vet ./...
python scripts/release_notes.py --version 2.14.3 --output RELEASE-NOTES.test.txt
```

GitHub Actions dodatno izvršava puni Windows produkcijski build. Release workflow nakon kompresije verificira i konačni Windows ZIP.

## Struktura repozitorija

```text
README.md              glavni hrvatski pregled projekta
LICENSE                vlasnička ByFTP licenca
CHANGELOG.md            povijest izdanja
VERSION                 jedini izvor release verzije
BUILD-WINDOWS.*         produkcijski Windows entrypointi
cmd/                    aplikacija, instalacija i uklanjanje
internal/               tipizirani runtime moduli
build/                  službeni PNG/ICO resursi
scripts/                build, audit, bundle, release i PE alati
docs/                   detaljna dokumentacija i slike
.github/                CI, release workflow i hrvatski GitHub predlošci
```

## Dokumentacija

- [Indeks dokumentacije](docs/README.md)
- [Arhitektura](docs/ARHITEKTURA.md)
- [Sigurnost](docs/SIGURNOST.md)
- [Privatnost](docs/PRIVATNOST.md)
- [Testiranje](docs/TESTIRANJE.md)
- [Provjera izdanja](docs/PROVJERA-IZDANJA.md)
- [Potpisivanje](docs/POTPISIVANJE.md)
- [Plan razvoja](docs/PLAN-RAZVOJA.md)
- [Podrška](docs/PODRSKA.md)
- [Doprinos](docs/DOPRINOS.md)
- [Obavijesti trećih strana](docs/OBAVIJESTI-TRECIH-STRANA.md)
- [Izdavanje na GitHubu](docs/IZDAVANJE-NA-GITHUBU.md)
- [Povijest promjena](CHANGELOG.md)

## Licenca

**Copyright © 2026 Brendigo. Sva prava pridržana.**

ByFTP je **vlasnički/source-available softver, nije open-source softver**. Objavljeni izvorni kod može se pregledavati radi transparentnosti, sigurnosne provjere i evaluacije, ali se bez prethodnog pisanog dopuštenja tvrtke Brendigo ne daje opće pravo izmjene, redistribucije, rebrandinga, prodaje, sublicenciranja, izrade izvedenica ili ponovne uporabe koda u drugom projektu.

Pročitajte cijelu [ByFTP vlasničku licencu](LICENSE).

## Sigurnosne prijave

Za osjetljive sigurnosne probleme slijedite [SIGURNOST.md](docs/SIGURNOST.md). U javni issue nikada ne stavljajte lozinke, privatne ključeve, stvarne produkcijske hostove, korisnička imena ili povjerljive podatke klijenata.