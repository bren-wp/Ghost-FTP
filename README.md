<p align="center">
  <img src="docs/slike/byftp-zaglavlje.png" alt="ByFTP — siguran prijenos datoteka" width="900">
</p>

<p align="center">
  <strong>ByFTP 1.0.0 — privatni Windows file-transfer suite tvrtke Brendigo.</strong><br>
  FTP/FTPS · SFTP · SSH · S3 · bez telemetrije aplikacije · bez obaveznog cloud računa.
</p>

<p align="center">
  <a href="https://github.com/bren-wp/by-ftp/releases"><strong>Preuzimanje</strong></a> ·
  <a href="docs/INSTALACIJA.md"><strong>Instalacija</strong></a> ·
  <a href="docs/KLIJENTI.md"><strong>Klijenti</strong></a> ·
  <a href="docs/SIGURNOST.md"><strong>Sigurnost</strong></a> ·
  <a href="docs/PRIVATNOST.md"><strong>Privatnost</strong></a> ·
  <a href="LICENSE"><strong>Licenca</strong></a>
</p>

# ByFTP

**Trenutačna stabilna linija: 1.0.0**

ByFTP je skup nativnih i terminalnih klijenata za prijenos datoteka, udaljeni SSH pristup i S3 objektnu pohranu. Projekt koristi jedan zajednički sigurnosni i transferni core gdje protokoli dijele iste primitive, ali korisnik može preuzeti i pokrenuti samo klijent koji mu treba.

Nova 1.x linija namjerno počinje od `1.0.0`. Buduća funkcionalna izdanja koriste 1.1.0, 1.2.0 i dalje; hitni kompatibilni popravci mogu koristiti patch verzije poput 1.0.1.

## Obitelj aplikacija

| Aplikacija | Namjena | Windows sučelje | Protokoli / API |
|---|---|---|---|
| **ByFTP** | All-in-One file manager | nativni dvopanelni Win32 GUI | FTP, FTPS, SFTP |
| **ByFTP FTP Client** | odvojeni FTP file manager | nativni dvopanelni Win32 GUI | FTP, eksplicitni FTPS, implicitni FTPS |
| **ByFTP SFTP Client** | odvojeni sigurni file manager | nativni dvopanelni Win32 GUI | SFTP preko OpenSSH |
| **ByFTP SSH Client** | puni SSH terminal | konzolni terminal nad sistemskim OpenSSH | SSH |
| **ByFTP S3 Client** | S3/object-storage upravljanje | terminalni file-manager frontend u 1.0.0 | S3 REST + AWS Signature Version 4 |

FTP i SFTP klijenti nisu samo preimenovani EXE-ovi. Imaju zaseban process identity, zasebni profil/state prostor i strogi allowlist protokola. FTP Client ne može otvoriti SFTP profil, a SFTP Client ne može prebaciti sesiju na FTP/FTPS.

## Preuzimanje — GitHub Release

Službeni distribucijski kanal je [GitHub Releases](https://github.com/bren-wp/by-ftp/releases).

Release 1.x namjerno ima mali i predvidljiv javni skup: **12 EXE datoteka**. Nema dodatnog custom Source ZIP-a, verification datoteke, standalone Uninstall EXE-a, Windows ZIP bundlea ili internih build izvještaja među javnim assetima.

### ByFTP All-in-One

| Arhitektura | Instalacija | Portable |
|---|---|---|
| x64 | `ByFTP-1.0.0-Setup-x64.exe` | `ByFTP-1.0.0-Portable-x64.exe` |
| x86 / 32-bitni proces | `ByFTP-1.0.0-Setup-x86.exe` | `ByFTP-1.0.0-Portable-x86.exe` |

### Odvojeni klijenti

| Klijent | x64 | x86 |
|---|---|---|
| FTP | `ByFTP-FTP-Client-1.0.0-Portable-x64.exe` | `ByFTP-FTP-Client-1.0.0-Portable-x86.exe` |
| SFTP | `ByFTP-SFTP-Client-1.0.0-Portable-x64.exe` | `ByFTP-SFTP-Client-1.0.0-Portable-x86.exe` |
| SSH | `ByFTP-SSH-Client-1.0.0-Portable-x64.exe` | `ByFTP-SSH-Client-1.0.0-Portable-x86.exe` |
| S3 | `ByFTP-S3-Client-1.0.0-Portable-x64.exe` | `ByFTP-S3-Client-1.0.0-Portable-x86.exe` |

GitHub uz svaki tag automatski prikazuje **Source code (zip)** i **Source code (tar.gz)**. To su jedine source arhive na Release stranici.

> Windows 11 je 64-bitni operacijski sustav. x86 paket postoji za 32-bitni ByFTP proces i kompatibilna Windows okruženja; za moderno Windows 11 računalo preporuka je x64.

## Instalacija

### Standardna Windows instalacija

1. Preuzmite `ByFTP-1.0.0-Setup-x64.exe` za uobičajeno 64-bitno Windows računalo.
2. Zatvorite prethodno pokrenuti ByFTP prije nadogradnje.
3. Pokrenite Setup.
4. Installer verificira ugrađeni payload prije upisa aplikacije.
5. ByFTP se instalira u korisnički Windows kontekst, bez potrebe za administrativnim servisom.
6. Aplikacija se registrira u standardnom Windows popisu instaliranih aplikacija.

Interna komponenta potrebna za standardni Windows removal lifecycle ugrađena je u Setup. **Nije javni GitHub Release asset i korisnik je ne treba ručno preuzimati ili pokretati.**

### Uklanjanje

Otvorite:

**Postavke → Aplikacije → Instalirane aplikacije → ByFTP → Deinstaliraj**

Portable EXE nije registriran kao instalirana aplikacija; dovoljno je zatvoriti ga i ukloniti samu EXE datoteku. Profilni/state podaci ne brišu se automatski samo zato što je obrisan portable EXE.

Detalji i troubleshooting: [docs/INSTALACIJA.md](docs/INSTALACIJA.md).

## Prvi spoj — FTP / FTPS / SFTP

U ByFTP, FTP Clientu ili SFTP Clientu:

1. odaberite dopušteni protokol;
2. upišite poslužitelj;
3. provjerite port;
4. upišite korisničko ime;
5. upišite lozinku ili za SFTP odaberite privatni ključ;
6. kliknite **Poveži**;
7. za prvi SFTP spoj provjerite prikazani SHA-256 host-key fingerprint prije prihvaćanja.

ByFTP ne prikazuje **POVEZANO** samo zato što je pokrenut `curl` ili OpenSSH proces. Engine mora završiti autentikaciju i izvršiti stvarni početni udaljeni `List` probe. Tek nakon uspješnog odgovora UI prelazi u povezano stanje.

Unesene tajne ostaju u zaključanom password polju samo tijekom neuspjelog/retry pokušaja kako ih korisnik ne bi morao ponovno tipkati nakon obične mrežne pogreške. Nakon uspješnog spajanja polja se prazne.

## FTP i FTPS

FTP/FTPS adapter koristi sistemski `curl` i kontrolirani config preko standardnog ulaza. Lozinka se ne stavlja u command line.

Podržano:

- FTP;
- eksplicitni FTPS;
- implicitni FTPS;
- listanje;
- upload/download;
- stvaranje mape;
- preimenovanje;
- brisanje;
- transfer queue;
- retry/cancel;
- sigurni lokalni staging i rollback gdje ga operacija dopušta.

FTP nije kriptiran protokol. Za osjetljive podatke koristite FTPS ili SFTP.

## SFTP

SFTP koristi sistemski OpenSSH, ali ByFTP stvara vlastiti kratkotrajni session config umjesto nekontroliranog oslanjanja na korisničke SSH postavke.

Sigurnosne granice uključuju:

- SHA-256 host-key fingerprint potvrdu;
- pin vezan uz točan endpoint;
- privatni `known_hosts`;
- `BatchMode=no` za podržani AskPass put;
- zabranu `sftp -b` regresije koja bi ugasila password/passphrase autentikaciju;
- blokiran ProxyCommand/ProxyJump, agent i forwarding na file-transfer putu;
- validaciju privatnog ključa kao regularne lokalne datoteke bez symlink/reparse preusmjeravanja;
- session lifecycle koji ne zatvara adapter ispod aktivne operacije.

## SSH Client

`ByFTP-SSH-Client-...exe` je zaseban program za puni terminalni SSH pristup. Ne koristi file-transfer `Session` interface samo zato što SFTP i SSH dijele transport.

ByFTP generira privatni kratkotrajni OpenSSH config koji postavlja:

- točan HostName, port i korisnika;
- vlastiti `known_hosts`;
- `StrictHostKeyChecking ask`;
- `BatchMode no`;
- `IdentityAgent none`;
- `ProxyCommand none` i `ProxyJump none`;
- `ClearAllForwardings yes`;
- `ForwardAgent no` i `ForwardX11 no`;
- `GSSAPIAuthentication no`;
- `IdentitiesOnly yes`;
- eksplicitni privatni ključ ili `IdentityFile none`.

**Lozinku, MFA/keyboard-interactive kod i passphrase unosi izravno OpenSSH u terminalu.** ByFTP ih ne presreće i ne sprema u svoj profilni model.

## S3 Client

`ByFTP-S3-Client-...exe` je zaseban S3/object-storage klijent. Ne koristi AWS SDK i ne uvodi nove Go module. Potpisivanje koristi vlastitu standard-library implementaciju AWS Signature Version 4.

Podržano u 1.0.0:

- S3 HTTPS endpoint;
- lokalni HTTP samo za loopback testni/S3-kompatibilni servis;
- regija;
- access key + secret key;
- opcionalni session token u coreu;
- bucket;
- ListObjectsV2 s paginacijom;
- prefix prikaz kao mape;
- streaming upload;
- streaming download u privatni `.part` staging;
- download no-replace lokalni commit;
- delete;
- stvaranje prefixa;
- server-side copy + delete rename uz provjeru postojećeg odredišta.

S3 single `PutObject` u 1.0.0 ograničen je na 5 GB. Multipart upload bit će zasebna funkcionalna nadogradnja; ByFTP za veći objekt vraća jasnu grešku umjesto pokušaja koji bi izgledao uspješno.

S3 secret key i session token ostaju samo u memoriji aktivnog procesa i ne zapisuju se u ByFTP profilno spremište u 1.0.0.

## Dvopanelni file-manager workflow

Windows All-in-One, FTP Client i SFTP Client koriste Commander-style raspored:

- lokalno računalo lijevo;
- udaljeni poslužitelj desno;
- adrese/putanje iznad panela;
- upload/download između panela;
- transfer queue pri dnu;
- status veze i aktivnosti;
- tipkovnički i mišem dostupne osnovne radnje;
- moderni tamni Brendigo izgled.

WinSCP je korišten kao **funkcionalna/UX referenca** za očekivani file-transfer workflow i podjelu FTP/SFTP/SSH/S3 mogućnosti. WinSCP izvorni kod je GPLv3 i **nije kopiran u ByFTP**. ByFTP zadržava vlastitu Go implementaciju, vlastiti dizajn i vlastiti licencni model.

## Profili i vjerodajnice

Windows profilne tajne za file-transfer klijente koriste DPAPI.

- spremljena lozinka vrijedi samo za isti protokol + host + port + korisnika;
- SFTP passphrase dodatno mora pripadati istom privatnom ključu;
- promjena endpointa/računa/ključa čisti tajnu koja više ne pripada identitetu;
- host-key pin vrijedi samo za isti SFTP endpoint;
- korisnik može izričito zadržati ili ukloniti spremljene vjerodajnice;
- password polje nikada ne prikazuje spremljenu plaintext vrijednost.

Odvojeni FTP/SFTP klijenti koriste zasebni profil/state prostor pod ByFTP podatkovnom mapom.

## Transfer engine i stabilnost

ByFTP transfer queue podržava:

- 1–8 paralelnih radnika;
- pause/resume;
- cancel;
- retry;
- connection-generation identitet;
- pravilno razlikovanje `done`, `skipped`, `cancelled` i `failed` završetaka;
- siguran disconnect koji čeka aktivne operacije bez zatvaranja adaptera ispod njih;
- bounded disconnect timeout s deferred cleanupom;
- no-follow lokalne sigurnosne provjere;
- zabranu filesystem-root brisanja;
- ograničenu rekurziju i maksimalni broj stavki.

## Privatnost

ByFTP runtime nema:

- analitiku korištenja;
- oglase;
- vanjski crash-reporting servis;
- obavezni cloud račun;
- browser/localhost kontrolni server;
- skriveni update servis;
- trajni activity/error telemetry log.

Produkcijski CI prije testiranja i builda izvršava `go telemetry off` i provjerava stvarni Go telemetry mode. Build skripte odbijaju produkcijski build ako toolchain telemetry nije stvarno `off`.

Više: [docs/PRIVATNOST.md](docs/PRIVATNOST.md).

## GitHub Packages

Svaki službeni 1.x release mora objaviti **istu verziju** u svih pet GitHub Packages paketa:

- `ByFTP.Suite`
- `ByFTP.FTP.Client`
- `ByFTP.SFTP.Client`
- `ByFTP.SSH.Client`
- `ByFTP.S3.Client`

Release workflow završava neuspjehom ako GitHub API nakon objave ne pronađe očekivanu verziju u svih pet paketa.

## Verzijska politika

Nova javna release linija počinje s `1.0.0`.

- `1.0.0` — početna stabilna 1.x linija;
- `1.1.0`, `1.2.0`, ... — nove kompatibilne mogućnosti;
- `1.0.1`, `1.1.1`, ... — hitni stabilizacijski/sigurnosni patch ako je potreban.

Prijelaz na 1.0.0 namjerno uklanja stare GitHub Release zapise i stare verzijske `v*` tagove. Git commit povijest se ne prepisuje.

## Produkcijski CI

Merge kandidat mora proći:

- brand asset audit;
- hrvatski audit;
- version audit;
- documentation audit;
- security audit;
- privacy audit;
- release-contract audit;
- Python regresije release alata;
- `go test ./...`;
- `go test -race ./...`;
- `go vet ./...`;
- Windows x64+x86 build svih 12 EXE izlaza;
- PE subsystem/ASLR/NX/resource/telemetry provjeru;
- Linux runtime smoke;
- macOS runtime smoke.

Javni release ne nastaje dok svi production gateovi nisu zeleni.

## Izgradnja iz izvornog koda

Za produkcijski build treba Go **1.26.5+** i isključena Go telemetry postavka.

```text
go telemetry off
```

### Windows — cijeli release kandidat

```powershell
.\BUILD-WINDOWS.ps1
```

### Pojedinačni klijenti

```powershell
go build -o ByFTP.exe .\cmd\byftp
go build -o ByFTP-FTP-Client.exe .\cmd\ftpclient
go build -o ByFTP-SFTP-Client.exe .\cmd\sftpclient
go build -o ByFTP-SSH-Client.exe .\cmd\sshclient
go build -o ByFTP-S3-Client.exe .\cmd\s3client
```

Produkcijski GitHub build dodatno koristi `-trimpath`, `-buildvcs=false`, verzijske `ldflags`, Windows subsystem oznake, PE resurse i sigurnosnu verifikaciju.

## Dokumentacija

- [Instalacija](docs/INSTALACIJA.md)
- [Klijenti i protokoli](docs/KLIJENTI.md)
- [Arhitektura](docs/ARHITEKTURA.md)
- [Sigurnost](docs/SIGURNOST.md)
- [Privatnost](docs/PRIVATNOST.md)
- [Testiranje](docs/TESTIRANJE.md)
- [Provjera izdanja](docs/PROVJERA-IZDANJA.md)
- [Objava na GitHubu](docs/IZDAVANJE-NA-GITHUBU.md)
- [Potpisivanje](docs/POTPISIVANJE.md)
- [Podrška](docs/PODRSKA.md)
- [Plan razvoja](docs/PLAN-RAZVOJA.md)
- [Doprinos](docs/DOPRINOS.md)

## Licenca

ByFTP nije WinSCP fork. WinSCP GPLv3 kod nije uključen u projekt. ByFTP se distribuira prema uvjetima u [LICENSE](LICENSE).

---

**Brendigo · ByFTP 1.0.0 · privatnost i predvidljivo ponašanje prije skrivenih automatizama.**
