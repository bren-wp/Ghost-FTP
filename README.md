<p align="center">
  <img src="docs/slike/byftp-zaglavlje.png" alt="ByFTP — siguran prijenos datoteka" width="900">
</p>

<p align="center">
  <strong>Privatan FTP / FTPS / SFTP klijent za Windows, Linux i macOS.</strong><br>
  ByFTP je fokusirani alat tvrtke Brendigo bez telemetrije aplikacije, oglasa, obaveznog cloud računa, browser upravljanja ili skrivenog mrežnog servisa.
</p>

<p align="center">
  <a href="https://github.com/bren-wp/by-ftp/releases"><strong>Izdanja</strong></a> ·
  <a href="docs/INSTALACIJA.md"><strong>Instalacija</strong></a> ·
  <a href="docs/SIGURNOST.md"><strong>Sigurnost</strong></a> ·
  <a href="docs/PRIVATNOST.md"><strong>Privatnost</strong></a> ·
  <a href="LICENSE"><strong>Licenca</strong></a>
</p>

<p align="center">
  <a href="../../actions/workflows/ci.yml"><img alt="Provjere" src="../../actions/workflows/ci.yml/badge.svg"></a>
</p>

# ByFTP

ByFTP koristi jedan tipizirani Go `Engine`, zajedničke FTP/FTPS/SFTP adaptere, isti transfer queue i iste sigurnosne granice na svim podržanim sustavima. Windows izdanje ima puni izvorni Win32 dvopanelni GUI. Linux i macOS koriste funkcionalno terminalno sučelje nad istim engineom i stvarnim mrežnim adapterima.

**Trenutačno izdanje: 2.16.2**

## Produkcijska podrška

| Platforma | Službeni paket | Arhitekture | Sučelje |
|---|---|---|---|
| Windows 10/11 | Setup EXE, Portable EXE, ZIP | x64, x86 | puni Win32 GUI |
| Linux | DEB | amd64, arm64, i386 | terminalni klijent |
| macOS | Universal PKG | Intel x86_64 + Apple Silicon arm64 | Finder launcher + terminalni klijent |

### Podržana autentikacija

| Način | Windows | Linux / macOS |
|---|---|---|
| FTP lozinka | da | da |
| FTPS lozinka | da | da |
| SFTP privatni ključ bez passphrasea | da | da |
| SFTP lozinka | da | trenutačno nije omogućeno |
| SFTP privatni ključ s passphraseom | da | trenutačno nije omogućeno |
| SFTP host-key provjera i potvrda | da | da |

Linux/macOS namjerno odbijaju nepodržani SFTP način prije mrežnog pokušaja. ByFTP ne prikazuje lažno stanje uspješnog spajanja i ne šalje SFTP tajnu kroz nesigurni argument naredbenog retka ili običan environment samo radi pariteta značajki.

## Preuzimanje

Službeni distribucijski kanal su [GitHub izdanja](https://github.com/bren-wp/by-ftp/releases).

### Windows x64

Za većinu modernih Windows računala:

- `ByFTP-<verzija>-Setup-x64.exe` — preporučena instalacija
- `ByFTP-<verzija>-Portable-x64.exe` — rad bez instalacije
- `ByFTP-<verzija>-Windows-x64.zip` — Setup + Portable + dokumentacija i bundle checksumovi

### Windows x86 / 32-bitni

- `ByFTP-<verzija>-Setup-x86.exe`
- `ByFTP-<verzija>-Portable-x86.exe`
- `ByFTP-<verzija>-Windows-x86.zip`

### Linux

- `ByFTP-<verzija>-Linux-amd64.deb`
- `ByFTP-<verzija>-Linux-arm64.deb`
- `ByFTP-<verzija>-Linux-i386.deb`

### macOS

- `ByFTP-<verzija>-macOS-Universal.pkg`

### Integritet izdanja

Svako izdanje dodatno sadrži:

- `SHA256.txt` — SHA-256 svih javnih paketa i zajedničkih release metapodataka
- `RELEASE-NOTES.txt` — bilješke generirane iz odgovarajućeg CHANGELOG odjeljka
- `BUILD-METADATA.txt` — verzija, release commit i podaci produkcijskog GitHub Actions runa

## Koji paket odabrati?

**Windows korisnik koji želi normalnu instalaciju:** odaberite Setup za svoju arhitekturu. Na većini računala to je x64.

**Windows korisnik koji ne želi instalirati aplikaciju:** odaberite Portable. Postavke i profilni podaci i dalje koriste ByFTP korisničku podatkovnu mapu, ali sama aplikacija ne prolazi Setup lifecycle.

**Linux korisnik:** odaberite DEB prema `dpkg --print-architecture` rezultatu.

**macOS korisnik:** koristite Universal PKG; isti paket sadrži Intel i Apple Silicon binarij.

Detaljni postupci nalaze se u [INSTALACIJA.md](docs/INSTALACIJA.md).

## Instalacija, nadogradnja i uklanjanje

### Windows

Setup instalira ByFTP u korisnički Windows kontekst i registrira aplikaciju u standardnom Windows popisu instaliranih aplikacija. Nadogradnja koristi isti Setup paket novije verzije i čuva korisničke podatke u predviđenoj ByFTP podatkovnoj lokaciji.

Za uklanjanje otvorite **Postavke → Aplikacije → Instalirane aplikacije → ByFTP** i odaberite standardnu Windows opciju za uklanjanje. Korisnik ne treba tražiti ili pokretati dodatnu release datoteku.

Portable paket pokreće se izravno i nije registriran kao klasična instalacija.

### Linux

Primjer za amd64:

```bash
sudo apt install ./ByFTP-<verzija>-Linux-amd64.deb
```

Paket instalira `/usr/bin/byftp`, desktop launcher i ikonu. Ovisnosti paketa su `ca-certificates`, `curl` i `openssh-client`.

### macOS

Otvorite `ByFTP-<verzija>-macOS-Universal.pkg` i slijedite standardni macOS Installer. Paket instalira `/Applications/ByFTP.app` i `/usr/local/bin/byftp`.

`ByFTP.app` otvara stvarni terminalni ByFTP klijent u Terminal aplikaciji; nije lažni web launcher.

## Prvi spoj

Za povezivanje trebate:

1. odabrati FTP, eksplicitni FTPS, implicitni FTPS ili SFTP;
2. upisati poslužitelj, port i korisničko ime;
3. upisati lozinku ili odabrati privatni ključ prema protokolu;
4. kod prvog SFTP spajanja provjeriti prikazani SHA-256 fingerprint host ključa;
5. potvrditi host ključ samo ako odgovara vrijednosti koju očekujete od administratora poslužitelja.

ByFTP ne smatra vezu uspješnom samim pokretanjem `curl` ili OpenSSH procesa. `remote.Manager.Connect()` mora dovršiti autentikaciju i uspješno izvršiti početni udaljeni `List` probe. Tek tada engine vraća `Connected=true` i Windows sučelje prikazuje **POVEZANO**.

## Povezivanje i pouzdanost

### FTP i FTPS

FTP/FTPS koristi sistemski `curl` s konfiguracijom preko standardnog ulaza. Vjerodajnica se ne stavlja u command line. Proxy i TLS override varijable iz vanjskog okruženja se sanitiziraju, a transfer koristi kontrolirane timeout i staging putove.

### SFTP

SFTP koristi sistemski OpenSSH. ByFTP:

- skenira host ključ i prikazuje SHA-256 fingerprint prije prvog povjerenja;
- veže prihvaćeni pin uz točan protokol, host i port;
- koristi privatni kratkotrajni `known_hosts` i session config;
- ne koristi `sftp -b`, jer taj način uključuje `BatchMode=yes` i može blokirati password/passphrase AskPass;
- koristi eksplicitni `BatchMode=no` na Windows AskPass putu;
- blokira naslijeđeni SSH agent, ProxyCommand, ProxyJump, PKCS#11 provider, forwarding i slične implicitne helper putove.

### Timeout, prekid i reconnect

Session lifecycle je referentno brojan. Disconnect prvo sprječava nove operacije i otkazuje session context, zatim čeka aktivne operacije. Adapter se ne zatvara ispod aktivnog transfera ili listinga. Ako caller deadline istekne, cleanup se završava odvojeno, a reconnect ostaje blokiran dok prethodna sesija nije sigurno zatvorena.

## Upravljanje datotekama

### Windows GUI

- dvopanelni lokalni/udaljeni prikaz
- višestruki odabir
- upload i download
- stvaranje mapa
- preimenovanje
- brisanje
- osvježavanje
- CHMOD gdje ga protokol/poslužitelj podržava
- transfer queue s 1–8 paralelnih radnika
- pause/resume, cancel i retry u engineu

### Linux/macOS terminal

Terminalni frontend koristi isti engine i podržava udaljene naredbe i transfere, uključujući `ls`, `cd`, `mkdir`, `rename`, `delete`, `chmod`, `get`, `put` i `pwd`.

## Sigurnost transfera

ByFTP koristi transakcijski pristup gdje je moguće:

- kriptografski nasumične staging nazive;
- provjeru da download staging ostaje regularna datoteka;
- no-replace aktivaciju i rollback;
- zabranu symlink/junction/reparse traversal izlaza;
- ponovnu validaciju lokalnog root-a prije queued prijenosa;
- connection-generation identitet koji sprječava retry starog posla na drugom serveru/računu;
- ograničenu rekurziju i broj stavki;
- kontroliranu enumeraciju do javnog limita od 50.000 stavki.

## Vjerodajnice i privatnost

### Windows

Spremljene profilne tajne koriste Windows DPAPI. Lozinka se automatski nasljeđuje samo za isti endpoint i korisničko ime. SFTP passphrase dodatno mora pripadati istom privatnom ključu. Promjena identiteta profila čisti vrijednost koja više ne pripada novoj konfiguraciji.

Windows AskPass je fail-closed: spremljenu tajnu daje samo jasno prepoznatom `password` ili `passphrase` promptu. MFA, OTP, security-key i nepoznati promptovi ne dobivaju spremljenu tajnu.

### Linux/macOS

FTP/FTPS aktivna lozinka čuva se samo u memoriji ByFTP procesa iza kriptografski nasumičnog tokena i briše se pri zatvaranju sesije. Terminalni frontend ne sprema profilne vjerodajnice na disk.

## Bez telemetrije aplikacije

ByFTP runtime nema:

- analitiku korištenja;
- oglašavanje;
- vanjski crash-reporting servis;
- obavezni cloud račun;
- skriveni update API;
- browser/localhost upravljački server;
- trajni runtime activity/error log.

Produkcijski CI dodatno eksplicitno postavlja **Go toolchain telemetry mode na `off`** prije testova i builda te ga provjerava. Produkcijske build skripte fail-closed odbijaju rad ako `go telemetry` nije `off`; ne oslanjaju se na neučinkovitu istoimenu OS env varijablu.

Detalji su u [PRIVATNOST.md](docs/PRIVATNOST.md).

## Provjera SHA-256

### Windows PowerShell

```powershell
Get-FileHash .\ByFTP-<verzija>-Setup-x64.exe -Algorithm SHA256
```

### Linux

```bash
sha256sum ByFTP-<verzija>-Linux-amd64.deb
```

### macOS

```bash
shasum -a 256 ByFTP-<verzija>-macOS-Universal.pkg
```

Usporedite rezultat s odgovarajućim retkom u službenom `SHA256.txt` prije distribucije ili instalacije.

## Potpisivanje i produkcijska ograničenja

Kod, build, bundle i SHA-256 provjere mogu biti potpuno automatizirane, ali identitet izdavača ne smije se fabricirati.

- Windows binariji nemaju status **Verified Publisher** dok nije dostupan stvarni Brendigo Authenticode certifikat.
- macOS paket nije **Developer ID** potpisan/notariziran dok nije dostupan stvarni Apple certifikat i odgovarajući secrets.
- Linux/macOS imaju terminalno, a ne puni Windows GUI sučelje.
- SFTP password i privatni ključ s passphraseom trenutačno su omogućeni na Windowsu; Linux/macOS ostaju fail-closed na SFTP ključu bez passphrasea dok se ne dovrši siguran Unix credential broker.

Ova ograničenja su namjerna i dokumentirana; ne prikazuju se kao dovršene značajke.

## Što donosi 2.16.2

2.16.2 je produkcijski hardening patch nad 2.16 linijom:

- release workflow više se ne pokreće ponovno zbog taga koji je sam publisher upravo izradio;
- svi release runovi koriste jednu serijaliziranu concurrency grupu;
- release ima zaseban quality job s auditima, Python regresijama, `go test`, `go test -race` i `go vet` prije objave;
- release Linux i macOS jobovi ponovno verificiraju paketnu strukturu prije uploada;
- publish staging mora sadržavati točno 10 očekivanih platformskih paketa prije izrade zajedničkih metapodataka;
- CI/release stvarno izvršava `go telemetry off` i potvrđuje rezultat;
- produkcijske build skripte odbijaju build ako Go telemetry mode nije `off`;
- Linux/macOS buildovi zahtijevaju Go 1.26.5+ kao i Windows build;
- lokalni build razdvaja javne izlaze od internih tehničkih build dokaza;
- javna dokumentacija opisuje samo aktualne distribucijske pakete i standardni OS lifecycle.

## Izgradnja iz izvornog koda

### Zahtjevi

- Go **1.26.5+**
- Go telemetry mode `off` za produkcijski build
- Python 3 za Windows/release audite
- Windows za puni `BUILD-WINDOWS.ps1`
- Linux s `dpkg-deb` za DEB pakete
- macOS s `lipo`, `pkgbuild`, `sips` i `iconutil` za Universal PKG

Prvo eksplicitno isključite Go toolchain telemetriju:

```bash
go telemetry off
```

Windows:

```powershell
.\BUILD-WINDOWS.ps1
```

Linux:

```bash
bash scripts/BUILD-LINUX.sh
```

macOS:

```bash
bash scripts/BUILD-MACOS.sh
```

Kanonski broj verzije nalazi se samo u [`VERSION`](VERSION). Produkcijski buildovi koriste lokalni Go toolchain, `GOPROXY=off`, `GOSUMDB=off`, `CGO_ENABLED=0` i ne preuzimaju vanjske Go module.

## Automatizirane provjere

Glavni quality skup:

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
```

GitHub Actions zatim dodatno:

1. izvršava process-level FTP/SFTP connect smoke regresije;
2. gradi i verificira Windows x64 i x86;
3. testira Linux i gradi amd64/arm64/i386 DEB;
4. testira macOS i gradi Universal PKG;
5. u release workflowu ponovno izvršava zaseban production quality/race gate;
6. objavljuje izdanje samo nakon prolaska svih platformskih gateova.

## Release integritet

`scripts/publish_release.ps1` radi fail-closed:

- tag mora pokazivati na točan release commit;
- postojeći asset uspoređuje se po nazivu, veličini i SHA-256 digestu;
- nedostajući potvrđeni asset može se nadopuniti;
- drugačiji sadržaj pod istim nazivom zaustavlja izdanje;
- neočekivani asset zaustavlja izdanje;
- slijepi overwrite nije dopušten.

Release workflow ima jedan autoritativni okidač za promjenu `VERSION` na `main` i manualni rerun. Tag koji publisher stvori ne pokreće drugi paralelni release.

## Struktura repozitorija

```text
README.md              glavni produkcijski pregled
LICENSE                vlasnička ByFTP licenca
CHANGELOG.md            povijest izdanja
VERSION                 jedini izvor produkcijske verzije
BUILD-WINDOWS.*         kanonski Windows build ulazi
cmd/                    aplikacija i Windows instalacijski lifecycle
internal/               engine, remote, transfer, config i sigurnosni moduli
build/                  službeni PNG/ICO resursi
scripts/                build, audit, bundle i release alati
docs/                   detaljna hrvatska dokumentacija i slike
.github/                CI, release workflow i GitHub predlošci
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

ByFTP je **vlasnički/source-available softver, nije open-source softver**. Objavljeni izvorni kod može se pregledavati radi transparentnosti, sigurnosne provjere i evaluacije, ali bez prethodnog pisanog dopuštenja tvrtke Brendigo ne daje se opće pravo izmjene, redistribucije, rebrandinga, prodaje, sublicenciranja ili izrade izvedenih distribucija.

Pročitajte cijelu [ByFTP vlasničku licencu](LICENSE).

## Podrška i sigurnosne prijave

Za uobičajene probleme pogledajte [PODRSKA.md](docs/PODRSKA.md). Za osjetljive sigurnosne probleme slijedite [SIGURNOST.md](docs/SIGURNOST.md). U javni issue nikada ne stavljajte lozinke, privatne ključeve, stvarne produkcijske hostove, korisnička imena ili povjerljive podatke klijenata.
