# Instalacija ByFTP-a

Ovaj dokument opisuje službene pakete, preduvjete, instalaciju, nadogradnju, uklanjanje i prvi spoj. Kanonska verzija nalazi se u [`../VERSION`](../VERSION), a službeni paketi objavljuju se kroz [GitHub Releases](https://github.com/bren-wp/by-ftp/releases).

## Prije preuzimanja

1. odaberite paket za svoj operacijski sustav i arhitekturu;
2. preuzmite i `SHA256.txt` iz istog izdanja;
3. nakon preuzimanja usporedite SHA-256 paketa sa službenim retkom;
4. ne koristite kopiju paketa čiji hash ne odgovara službenom izdanju.

## Windows

### Koju arhitekturu odabrati?

Za većinu Windows 10/11 računala koristite x64. x86 je namijenjen 32-bitnom Windowsu ili slučaju kada je izričito potreban 32-bitni proces.

### Windows x64

- `ByFTP-<verzija>-Setup-x64.exe` — preporučena instalacija
- `ByFTP-<verzija>-Portable-x64.exe` — pokretanje bez instalacije
- `ByFTP-<verzija>-Windows-x64.zip` — Setup + Portable + dokumentacija i bundle checksumovi

### Windows x86

- `ByFTP-<verzija>-Setup-x86.exe`
- `ByFTP-<verzija>-Portable-x86.exe`
- `ByFTP-<verzija>-Windows-x86.zip`

### Instalacija

1. provjerite SHA-256 Setup paketa;
2. zatvorite staru ByFTP instancu ako je pokrenuta;
3. pokrenite Setup paket za svoju arhitekturu;
4. dovršite standardni instalacijski postupak;
5. pokrenite ByFTP iz instalirane lokacije ili Windows prečaca.

Setup sam upravlja potrebnim internim instalacijskim lifecycleom. Korisnik ne treba tražiti dodatne release datoteke za održavanje instalacije.

### Nadogradnja

Za nadogradnju preuzmite Setup novije verzije iste arhitekture i pokrenite ga preko postojeće instalacije. Instalacijski kod koristi transakcijski payload/rollback model i ne smije zamijeniti postojeću instalaciju nepotpunim payloadom.

Profilni i korisnički podaci čuvaju se u ByFTP korisničkoj podatkovnoj lokaciji odvojeno od programskih binarija.

### Uklanjanje

Otvorite:

**Postavke → Aplikacije → Instalirane aplikacije → ByFTP**

i odaberite standardnu Windows opciju za uklanjanje aplikacije.

### Portable način

Portable EXE ne prolazi klasični Setup lifecycle. Možete ga držati u vlastitoj mapi i pokretati izravno. To ne znači da se osjetljivi profilni podaci spremaju uz EXE; ByFTP koristi predviđenu korisničku podatkovnu lokaciju i Windows DPAPI za spremljene profilne tajne.

### Windows preduvjeti

- Windows 10 ili Windows 11
- sistemski `curl.exe` za FTP/FTPS
- Windows OpenSSH Client za SFTP

Ako OpenSSH Client nedostaje, SFTP se ne smatra dostupnim i ByFTP prikazuje hrvatsku poruku umjesto lažnog uspješnog spajanja.

## Linux

### Paketi

- `ByFTP-<verzija>-Linux-amd64.deb`
- `ByFTP-<verzija>-Linux-arm64.deb`
- `ByFTP-<verzija>-Linux-i386.deb`

Arhitekturu možete provjeriti naredbom:

```bash
dpkg --print-architecture
```

### Instalacija

Primjer za amd64:

```bash
sudo apt install ./ByFTP-<verzija>-Linux-amd64.deb
```

Paket instalira:

- `/usr/bin/byftp`
- ByFTP desktop launcher s terminalnim načinom rada
- ByFTP ikonu

Deklarirane ovisnosti paketa su:

- `ca-certificates`
- `curl`
- `openssh-client`

### Nadogradnja

Instalirajte noviji DEB paket istom `apt install ./...` metodom. Verzija i arhitektura paketa provjeravaju se u GitHub Actions release jobu prije objave.

### Uklanjanje

```bash
sudo apt remove byftp
```

Linux izdanje koristi terminalno sučelje i isti `api.Engine`, remote adaptere, transfer queue i sigurnosne granice kao Windows izdanje.

## macOS

### Paket

- `ByFTP-<verzija>-macOS-Universal.pkg`

Universal paket sadrži Intel x86_64 i Apple Silicon arm64 binarij.

### Instalacija

1. provjerite SHA-256 PKG paketa;
2. otvorite PKG u standardnom macOS Installeru;
3. dovršite instalaciju;
4. pokrenite `/Applications/ByFTP.app` ili `/usr/local/bin/byftp`.

Paket instalira:

- `/Applications/ByFTP.app`
- `/usr/local/bin/byftp`

`ByFTP.app` je Finder launcher koji otvara stvarni ByFTP terminalni klijent u Terminal aplikaciji.

### Nadogradnja

Otvorite noviji Universal PKG i instalirajte ga preko postojeće verzije.

### Uklanjanje

Trenutačni PKG nema zaseban GUI removal paket. Ako je potrebno ručno uklanjanje, uklonite `/Applications/ByFTP.app` i `/usr/local/bin/byftp` uz administratorska prava, te korisničke ByFTP podatke samo ako ih namjerno više ne želite zadržati.

## Podržana autentikacija

| Način | Windows | Linux/macOS |
|---|---|---|
| FTP lozinka | da | da |
| FTPS lozinka | da | da |
| SFTP privatni ključ bez passphrasea | da | da |
| SFTP lozinka | da | trenutačno ne |
| SFTP ključ s passphraseom | da | trenutačno ne |
| SFTP host-key potvrda | da | da |

Linux/macOS odbija nepodržani SFTP auth prije mrežnog pokušaja. To sprječava lažno stanje uspješnog povezivanja i izbjegava nesigurne improvizirane credential putove.

## Prvi spoj

### FTP/FTPS

Potrebni su:

- protokol
- poslužitelj
- port
- korisničko ime
- lozinka

Za FTPS provjerite s administratorom poslužitelja koristi li se eksplicitni ili implicitni način i koji port treba koristiti.

### SFTP

Potrebni su:

- poslužitelj
- port, uobičajeno 22 ako administrator nije odredio drugačije
- korisničko ime
- podržani način autentikacije

Kod prvog spajanja ByFTP skenira host ključ i prikazuje SHA-256 fingerprint. Fingerprint potvrdite samo ako ga možete usporediti s pouzdanim podatkom administratora poslužitelja.

## Kada ByFTP prikazuje „Povezano”

ByFTP ne označava vezu uspješnom samo zato što je `curl` ili OpenSSH proces pokrenut. `remote.Manager.Connect()` mora dovršiti autentikaciju i uspješno pročitati početni udaljeni direktorij (`List` probe). Tek nakon toga engine vraća `Connected=true`, a Windows UI prikazuje stanje **POVEZANO**.

Kod neuspjelog Windows pokušaja unesena lozinka/passphrase ostaje dostupna u zaključanom polju za ponovni pokušaj i briše se nakon stvarno potvrđene veze.

## Provjera SHA-256

### Windows

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

Rezultat mora odgovarati retku u `SHA256.txt` iz istog GitHub izdanja.

## Digitalni potpis

Windows paketi nemaju status Verified Publisher dok nije dostupan stvarni Brendigo Authenticode certifikat. macOS PKG nije Developer ID potpisan/notariziran bez stvarnog Apple certifikata. Workflow ne smije fabricirati potpis ili identitet izdavača.

Zbog toga je SHA-256 provjera posebno važna kod distribucije paketa izvan službene GitHub Release stranice.

## Produkcijska build pravila

Ako ByFTP gradite sami:

1. koristite Go 1.26.5 ili noviji podržani sigurnosni patch;
2. pokrenite `go telemetry off` i provjerite da `go telemetry` vraća `off`;
3. koristite kanonsku `VERSION` datoteku;
4. ne uključujte vanjske Go module;
5. pokrenite propisane audite i testove;
6. za javni release koristite GitHub Actions workflow, a ne ručno preslagivanje paketa.

Detalji su u [TESTIRANJE.md](TESTIRANJE.md), [PROVJERA-IZDANJA.md](PROVJERA-IZDANJA.md) i [IZDAVANJE-NA-GITHUBU.md](IZDAVANJE-NA-GITHUBU.md).
