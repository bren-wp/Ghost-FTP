<p align="center">
  <img src="docs/slike/byftp-zaglavlje.png" alt="ByFTP — siguran prijenos datoteka" width="900">
</p>

<p align="center">
  <strong>Privatan FTP / FTPS / SFTP klijent za Windows, Linux i macOS.</strong><br>
  ByFTP je fokusirani alat tvrtke Brendigo bez telemetrije, oglašavanja, obaveznog cloud računa ili skrivenog mrežnog servisa.
</p>

<p align="center">
  <a href="https://github.com/bren-wp/by-ftp/releases"><strong>Izdanja</strong></a> ·
  <a href="docs/INSTALACIJA.md"><strong>Instalacija</strong></a> ·
  <a href="LICENSE"><strong>Licenca</strong></a> ·
  <a href="docs/PRIVATNOST.md"><strong>Privatnost</strong></a> ·
  <a href="docs/SIGURNOST.md"><strong>Sigurnost</strong></a>
</p>

<p align="center">
  <a href="../../actions/workflows/ci.yml"><img alt="Provjere" src="../../actions/workflows/ci.yml/badge.svg"></a>
</p>

## Prenosite datoteke. Zadržite kontrolu.

ByFTP koristi isti tipizirani `Engine`, remote adaptere, transfer queue i sigurnosne granice na svim podržanim sustavima. Windows izdanje ima puni Win32 dvopanelni GUI; Linux i macOS izdanje imaju funkcionalno terminalno sučelje nad istim engineom i stvarnim mrežnim adapterima.

**Trenutačno izdanje: 2.16.1**

## Platforme

| Platforma | Paket | Arhitekture | Sučelje |
|---|---|---|---|
| Windows 10/11 | Setup EXE, Portable EXE, ZIP | x64, x86 | puni Win32 GUI |
| Linux | DEB | amd64, arm64, i386 | terminalni klijent |
| macOS | Universal PKG | Intel x86_64 + Apple Silicon arm64 | Finder launcher + terminalni klijent |

### Autentikacija

| Način | Windows | Linux / macOS |
|---|---|---|
| FTP/FTPS lozinka | da | da |
| SFTP privatni ključ bez passphrasea | da | da |
| SFTP lozinka | da | trenutačno nije omogućeno |
| SFTP privatni ključ s passphraseom | da | trenutačno nije omogućeno |
| SFTP host-key provjera i potvrda | da | da |

Linux/macOS terminalno izdanje namjerno odbija nepodržani SFTP način **prije mrežnog pokušaja** umjesto da prikaže lažno stanje „povezano”. Unix AskPass broker za password/passphrase SFTP ostaje zaseban sigurnosni korak.

## Preuzimanje

Preporučeni kanal su [GitHub izdanja](https://github.com/bren-wp/by-ftp/releases). ByFTP 2.16.1 javno objavljuje samo korisne instalacijske/portable pakete i zajedničke metapodatke.

### Windows

- `ByFTP-<verzija>-Setup-x64.exe`
- `ByFTP-<verzija>-Portable-x64.exe`
- `ByFTP-<verzija>-Windows-x64.zip`
- `ByFTP-<verzija>-Setup-x86.exe`
- `ByFTP-<verzija>-Portable-x86.exe`
- `ByFTP-<verzija>-Windows-x86.zip`

### Linux

- `ByFTP-<verzija>-Linux-amd64.deb`
- `ByFTP-<verzija>-Linux-arm64.deb`
- `ByFTP-<verzija>-Linux-i386.deb`

### macOS

- `ByFTP-<verzija>-macOS-Universal.pkg`

### Zajedničko

- `SHA256.txt`
- `RELEASE-NOTES.txt`
- `BUILD-METADATA.txt`

**Standalone Uninstaller, interni `verification.txt` i dodatni `ByFTP-<verzija>-Source.zip` nisu javni ByFTP release asseti.** Windows uninstaller ostaje ugrađen u Setup paket. GitHub automatski prikazuje vlastite „Source code (zip)” i „Source code (tar.gz)” poveznice za svaki tag; to je ugrađena GitHub funkcija, a ne dodatni ByFTP asset.

Detaljne upute: [Instalacija](docs/INSTALACIJA.md).

## Što donosi 2.16.1

### Stvarno provjeren connect put

- 2.16.1 zadržava 2.16.0 ispravku OpenSSH `sftp -b` problema: `-b` se ne koristi jer uključuje `BatchMode=yes` i može blokirati password/passphrase AskPass
- dodani su **process-level connect smoke testovi** koji stvarno pokreću child procese preko istog `exec.CommandContext`, stdin/config i parser puta koji koristi produkcija
- FTP/FTPS smoke potvrđuje da unesena lozinka ide kroz kratkotrajni runtime secret, zatim u curl config preko standardnog ulaza, da se listing stvarno parsira i da `Close()` uklanja runtime tajnu
- SFTP smoke potvrđuje da child proces ne dobiva `-b`, da dobiva `ls` naredbu preko stdin-a i da se vraćeni listing stvarno parsira
- Linux CI sada izvršava Go testove i `go vet` na Linux runneru **prije** izrade amd64/arm64/i386 DEB paketa
- macOS CI sada izvršava Go testove i `go vet` na stvarnom macOS runneru **prije** izrade Universal PKG paketa
- sigurnosni audit zahtijeva postojanje process-level connect smoke regresija pa se taj dokaz ne može slučajno ukloniti budućim refaktorom

### Pouzdanije povezivanje iz 2.16 linije

- Windows unesena lozinka/passphrase ostaje u zaključanom polju dok spajanje stvarno ne uspije; tek potvrđeni `Connected` briše osjetljivi unos
- dvostupanjska SFTP host-key potvrda ponovno koristi upravo unesenu vjerodajnicu bez traženja ponovnog unosa
- `remote.Manager.Connect()` ne vraća `Connected=true` dok adapter ne napravi stvarni udaljeni `List()` probe
- curl/OpenSSH timeout i korisničko otkazivanje propagiraju se kao pravi `context` uzroci
- bracketirani IPv6 host pravilno se normalizira za OpenSSH `HostName` i `ssh-keyscan`
- AskPass je fail-closed i ne šalje tajnu nepoznatom MFA/OTP/security-key promptu
- ne-Windows FTP/FTPS aktivna lozinka čuva se samo u procesnoj memoriji iza kriptografski nasumičnog tokena i briše se pri zatvaranju sesije

## Mogućnosti

### Veze

- FTP
- FTPS — eksplicitni i implicitni način
- SFTP
- potvrda i pinning SFTP host ključa
- podesivo vrijeme čekanja veze
- Windows profilne vjerodajnice zaštićene DPAPI mehanizmom
- profilne tajne vezane uz točan endpoint, korisnika i privatni ključ
- fail-closed AskPass i sanitizirano okruženje vanjskih mrežnih alata

### Upravitelj i prijenosi

- Windows: dvopanelni lokalni/udaljeni prikaz i višestruki odabir
- Linux/macOS: terminalne remote i transfer naredbe nad istim engineom
- pojedinačne datoteke i cijela stabla mapa u zajedničkom transfer sloju
- stvaranje mapa, preimenovanje, brisanje, osvježavanje i CHMOD
- 1–8 paralelnih prijenosa
- pause/resume, skupni cancel i retry u engineu
- automatski retry samo prolaznih mrežnih grešaka
- preskakanje postojećih datoteka
- transakcijski staging/backup/rollback
- zaštita od traversal putanja, symlinkova, junctiona i reparse-point izlaza
- kontrolirana enumeracija do 50.000 stavki

## Sigurnost i privatnost

ByFTP namjerno nema telemetriju, analitiku, oglašavanje, skriveni update API, trajni runtime log ni browser/localhost upravljački server. Normalan mrežni promet usmjeren je prema FTP/FTPS/SFTP poslužitelju koji korisnik odabere.

Ključne zaštite uključuju:

- profilne vjerodajnice vezane uz točan endpoint/račun/ključ prije automatskog korištenja
- SFTP host-key pin vezan uz protokol, host i port
- OpenSSH `-b`/BatchMode regresijski guard
- process-level FTP/SFTP connect smoke testove na Unix runnerima
- AskPass koji odbija nepoznate autentikacijske promptove
- Windows DPAPI za spremljene profilne tajne i aktivne Windows credential blobove
- procesno memorijsko runtime spremište aktivne FTP/FTPS tajne na Linuxu/macOS-u
- sanitizirano okruženje curl/OpenSSH procesa bez naslijeđenog proxyja, SSH agenta ili TLS overridea
- regular-file/symlink/reparse provjeru privatnog ključa i download staging datoteke
- session lifecycle zaštitu s bounded disconnectom i reconnect blokadom tijekom sigurnog zatvaranja
- no-follow rekurzivno brisanje s filesystem-root, depth i item granicama
- connection-generation i cross-server retry izolaciju
- offline Go build bez vanjskih Go modula

Detalji: [Sigurnost](docs/SIGURNOST.md) i [Privatnost](docs/PRIVATNOST.md).

## Zahtjevi

### Windows

- Windows 10 ili Windows 11
- x64 ili x86 paket koji odgovara sustavu
- sistemski `curl.exe` za FTP/FTPS
- Windows OpenSSH Client za SFTP

### Linux

- distribucija s `dpkg`/DEB paketima za službeni installer
- `curl`, `openssh-client`, `ca-certificates` i `stty`
- amd64, arm64 ili i386

### macOS

- Intel ili Apple Silicon Mac
- sistemski `curl`, OpenSSH i Terminal
- Universal PKG paket

### Izgradnja

- Go **1.26.5+**
- Python 3 za Windows/release audite
- Windows za `BUILD-WINDOWS.ps1`
- Linux s `dpkg-deb` za `scripts/BUILD-LINUX.sh`
- macOS s `lipo`, `pkgbuild`, `sips` i `iconutil` za `scripts/BUILD-MACOS.sh`

Kanonska verzija nalazi se isključivo u [`VERSION`](VERSION).

```powershell
.\BUILD-WINDOWS.ps1
```

```bash
bash scripts/BUILD-LINUX.sh
bash scripts/BUILD-MACOS.sh   # na macOS-u
```

Produkcijski buildovi koriste `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off` i `GOTELEMETRY=off`.

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
python scripts/release_notes.py --version 2.16.1 --output RELEASE-NOTES.test.txt
```

GitHub Actions dodatno izvršava Go testove na Linuxu i macOS-u, gradi puni Windows x64+x86 paket, sva tri Linux DEB paketa i macOS Universal PKG. Release workflow objavljuje izdanje tek nakon prolaska cijele platformne matrice.

## Struktura repozitorija

```text
README.md              glavni hrvatski pregled projekta
LICENSE                vlasnička ByFTP licenca
CHANGELOG.md            povijest izdanja
VERSION                 jedini izvor release verzije
BUILD-WINDOWS.*         produkcijski Windows entrypointi
cmd/                    aplikacija, instalacija i uklanjanje
internal/               tipizirani runtime moduli i sigurnosne granice
build/                  službeni PNG/ICO resursi
scripts/                Windows/Linux/macOS build, audit i release alati
docs/                   detaljna dokumentacija i slike
.github/                CI, cross-platform release workflow i GitHub predlošci
```

## Dokumentacija

- [Indeks dokumentacije](docs/README.md)
- [Instalacija](docs/INSTALACIJA.md)
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
