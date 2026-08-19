<p align="center">
  <img src="docs/slike/byftp-zaglavlje.png" alt="ByFTP — siguran prijenos datoteka" width="900">
</p>

<p align="center">
  <strong>Privatan FTP / FTPS / SFTP klijent tvrtke Brendigo.</strong><br>
  Bez telemetrije aplikacije, oglasa, obaveznog cloud računa i skrivenog lokalnog web servisa.
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

ByFTP koristi jedan tipizirani Go engine za FTP, FTPS i SFTP, zajednički transfer queue, provjeru putanja, kontrolirani lifecycle veze i fail-closed sigurnosne granice. Windows izdanje ima puni izvorni Win32 dvopanelni GUI. Linux i macOS koriste funkcionalni terminalni frontend nad istim engineom.

**Trenutačno izdanje: 1.0.4**

## Produkcijska podrška

| Platforma | Službeni paket | Arhitekture | Sučelje |
|---|---|---|---|
| Windows 10/11 | Setup EXE, Portable EXE, ZIP | x64, x86 | puni Win32 GUI |
| Linux | DEB | amd64, arm64, i386 | terminalni klijent |
| macOS | Universal PKG | Intel x86_64 + Apple Silicon arm64 | terminalni klijent |

### Autentikacija

| Način | Windows | Linux / macOS |
|---|---|---|
| FTP lozinka | da | da |
| FTPS lozinka | da | da |
| SFTP privatni ključ bez passphrasea | da | da |
| SFTP lozinka | da | ne, fail-closed |
| SFTP privatni ključ s passphraseom | da | ne, fail-closed |
| SFTP host-key fingerprint provjera | da | da |

Linux/macOS SFTP namjerno ne šalje lozinku ili passphrase kroz argument naredbenog retka ili običnu environment varijablu samo radi prividnog pariteta. Dok siguran Unix credential broker nije implementiran i testiran, ti načini ostaju blokirani.

## Preuzimanje

Službeni distribucijski kanal su GitHub izdanja repozitorija. Svako produkcijsko izdanje koristi verziju iz datoteke [`VERSION`](VERSION).

### Windows

- `ByFTP-<verzija>-Setup-x64.exe` — preporučena x64 instalacija
- `ByFTP-<verzija>-Portable-x64.exe` — x64 bez instalacije
- `ByFTP-<verzija>-Windows-x64.zip` — x64 paket s dokumentacijom i checksumovima
- `ByFTP-<verzija>-Setup-x86.exe` — x86 instalacija
- `ByFTP-<verzija>-Portable-x86.exe` — x86 bez instalacije
- `ByFTP-<verzija>-Windows-x86.zip` — x86 paket s dokumentacijom i checksumovima

### Linux

- `ByFTP-<verzija>-Linux-amd64.deb`
- `ByFTP-<verzija>-Linux-arm64.deb`
- `ByFTP-<verzija>-Linux-i386.deb`

### macOS

- `ByFTP-<verzija>-macOS-Universal.pkg`

Uz platformske pakete izdanje sadrži `SHA256.txt`, `RELEASE-NOTES.txt` i `BUILD-METADATA.txt`.

## Instalacija, nadogradnja i uklanjanje

Detaljni postupci nalaze se u [INSTALACIJA.md](docs/INSTALACIJA.md).

Windows Setup koristi standardni korisnički instalacijski lifecycle i čuva ByFTP korisničke podatke u predviđenoj podatkovnoj mapi. Portable izdanje pokreće se bez klasične instalacije.

Linux DEB instalira terminalni klijent prema arhitekturi sustava. macOS Universal PKG sadrži Intel i Apple Silicon varijantu te pokreće terminalni ByFTP frontend.

## Povezivanje i stabilnost

Veza se smatra uspješnom tek nakon stvarne autentikacije i početnog udaljenog `List` probea. Samo uspješno pokretanje `curl` ili OpenSSH procesa nije dovoljno za stanje **POVEZANO**.

Session lifecycle je referentno brojan. Prekid veze najprije zaustavlja prihvat novih transfera i otkazuje aktivne operacije, zatim čeka sigurno završavanje adaptera. Reconnect je blokiran dok se prethodna sesija još zatvara, čime se sprječava zatvaranje adaptera ispod aktivnog transfera.

### SFTP host-key zaštita

ByFTP:

- dohvaća i prikazuje SHA-256 fingerprint prije prvog povjerenja;
- veže spremljeni fingerprint uz točan endpoint;
- blokira vezu ako se očekivani host ključ promijeni;
- koristi privatni kratkotrajni `known_hosts`;
- ograničava OpenSSH proxy, agent, forwarding i slične implicitne helper putove;
- validira bracketirani i sirovi IPv6 oblik bez prihvaćanja neispravno uparenih zagrada;
- privatni ključ prije OpenSSH korištenja kopira iz stabilno verificiranog file handlea u privatni `0600` session snapshot, pa kasnija zamjena izvorne putanje ne mijenja ključ aktivne sesije.

## Upravljanje datotekama

Windows GUI podržava dvopanelni lokalni/udaljeni prikaz, višestruki odabir, upload, download, stvaranje mapa, preimenovanje, brisanje, osvježavanje, CHMOD gdje je podržan te transfer queue s pause/resume, cancel i retry funkcijama.

Linux/macOS terminal koristi isti engine i podržava `ls`, `cd`, `mkdir`, `rename`, `delete`, `chmod`, `get`, `put`, `pwd`, host-key potvrdu i transfer queue.

## Sigurnost transfera

ByFTP koristi obrambeni, transakcijski pristup:

- kriptografski nasumične staging nazive;
- provjeru regularne datoteke prije aktivacije downloada;
- stvarnu no-replace aktivaciju: Windows koristi exclusive `MoveFileExW`, Linux kernel `renameat2(RENAME_NOREPLACE)`, a macOS regularne staging datoteke aktivira ekskluzivnim hard-link korakom bez check-then-overwrite prozora;
- stabilno otvaranje direktorija prije rekurzivnog lokalnog brisanja i ponovnu provjeru identiteta direktorija prije nastavka/finalnog uklanjanja;
- zaštitu od symlink, junction i reparse traversal izlaza;
- ponovnu validaciju lokalnog root-a prije queued transfera;
- FTP/FTPS/SFTP upload koristi privatni lokalni byte-for-byte snapshot izrađen iz verificiranog otvorenog file handlea, a vanjski child proces ne dobiva originalnu korisničku putanju;
- lokalni upload snapshot SHA-256 provjerava se tijekom izrade, uspoređuje s drugim punim čitanjem izvora te ponovno provjerava nakon mrežnog čitanja i prije remote commit-a;
- snapshot se uklanja prije remote revalidation/commit faze; cleanup failure blokira aktivaciju remote temp objekta;
- ponovnu provjeru remote odredišta nakon završetka temp uploada i neposredno prije rename/backup commit faze;
- `SkipExisting` ponovno se primjenjuje na svježe remote stanje, a novonastali direktorij/symlink blokira commit i čisti temp upload;
- batch rezervacija i retry vežu `ConnectionIdentity()` uz istu monotonu transfer generation prije i poslije identity poziva; reconnect tijekom te granice odbija mutaciju umjesto da stari identity završi na novoj sesiji;
- vezanje retry posla uz identitet iste veze;
- ograničenu rekurziju i broj stavki;
- validaciju udaljenih putanja i command-stream separatora;
- bounded stdout/stderr za vanjske mrežne alate;
- context timeout/cancel propagaciju kroz adaptere i transfer queue.

Sigurniji lokalni upload snapshot namjerno zahtijeva privremeni lokalni prostor približno veličini datoteke koja se šalje i dodatna lokalna čitanja radi sadržajne stabilnosti. Ako nema dovoljno privremenog prostora ili snapshot nije moguće sigurno izraditi/ukloniti, upload se zaustavlja fail-closed prije finalnog remote commit-a.

## Vjerodajnice i privatnost

Na Windowsu spremljene profilne tajne koriste DPAPI. Lozinka se ponovno koristi samo za isti protokol, host, port i korisničko ime; SFTP passphrase dodatno mora pripadati istom privatnom ključu. Promjena identiteta profila čisti tajne koje više ne pripadaju novoj konfiguraciji.

Aktivne Linux/macOS FTP/FTPS lozinke drže se samo u memoriji procesa kroz kratkotrajni runtime-secret token i brišu se pri zatvaranju sesije. Terminalni frontend ne nudi trajno spremanje tih tajni.

ByFTP runtime nema analitiku korištenja, oglase, vanjski crash-reporting servis, obavezni cloud račun, browser upravljanje ni trajni runtime activity/error log.

Produkcijski CI i build skripte eksplicitno izvršavaju `go telemetry off` i provjeravaju rezultat prije produkcijskog builda.

## Provjera SHA-256

Windows PowerShell:

```powershell
Get-FileHash .\ByFTP-<verzija>-Setup-x64.exe -Algorithm SHA256
```

Linux:

```bash
sha256sum ByFTP-<verzija>-Linux-amd64.deb
```

macOS:

```bash
shasum -a 256 ByFTP-<verzija>-macOS-Universal.pkg
```

Rezultat usporedite s odgovarajućim retkom u službenom `SHA256.txt`.

## Potpisivanje i ograničenja

Windows binariji nemaju status Verified Publisher dok nije dostupan stvarni Brendigo Authenticode certifikat. macOS paket nije Developer ID potpisan/notariziran bez stvarnog Apple identiteta i potrebnih secrets. Release workflow ne fabricira publisher status.

## Što donosi 1.0.4

1.0.4 zatvara generation/connection-identity race u transfer queueu:

- `ReserveBatch` capturea aktualnu generation prije `ConnectionIdentity()` i ponovno je provjerava prije rezerviranja kapaciteta;
- `RetryBatch` radi isti dvostruki guard prije promjene failed/cancelled posla u `queued`;
- disconnect/reconnect tijekom identity lookup-a više ne može spojiti stari connection ID s novom generation;
- identity lookup se i dalje izvršava izvan `transfer.Manager.mu`, pa se sigurnosna provjera ne plaća novim lock-order/deadlock rizikom;
- deterministički testovi mijenjaju generation iz samog identity callbacka i potvrđuju da queue/job stanje ostaje nepromijenjeno.

## Što donosi 1.0.3

1.0.3 zatvara lokalni source-path TOCTOU i sadržajnu nestabilnost tijekom uploada:

- originalna lokalna putanja više se ne predaje `curl`/OpenSSH child procesu nakon odvojenog path checka;
- ByFTP kopira verificirani otvoreni izvor u privatni `byftp-upload-*` snapshot i child procesu daje samo snapshot putanju;
- SHA-256 snapshota mora odgovarati drugom punom čitanju istog otvorenog izvora prije početka mrežnog prijenosa;
- nakon mrežnog čitanja snapshot se ponovno hashira, pa i same-size/same-mtime sadržajna izmjena blokira remote commit;
- lokalni snapshot se uklanja prije 1.0.2 fresh remote revalidacije i transakcijskog rename/backup commit-a;
- sigurnosni tradeoff je dodatni lokalni disk prostor i lokalni I/O, što je namjerno odabrano umjesto slabije hard-link semantike.

## Što donosi 1.0.2

1.0.2 dodatno učvršćuje završnu fazu FTP/FTPS/SFTP uploada u okruženjima gdje više klijenata može mijenjati isti remote direktorij:

- nakon prijenosa u `.byftp-part-*` ByFTP ponovno lista odredišni direktorij neposredno prije aktivacije;
- novonastali direktorij ili symlink pod finalnim imenom blokira commit i uzrokuje cleanup temp objekta;
- `SkipExisting` poštuje i datoteku koja se pojavila tijekom samog uploada;
- overwrite/backup/rollback odluke koriste svježi remote snapshot umjesto stanja snimljenog prije dugog prijenosa;
- FTP/FTPS i SFTP dijele isti revalidation helper i regresijske testove.

## Što donosi 1.0.1

1.0.1 je sigurnosno/stabilnosno izdanje koje nadograđuje nepromjenjivi 1.0.0 tag bez prepisivanja već objavljenog sadržaja:

- Linux lokalna aktivacija/rollback više nema `Lstat` → `rename` overwrite race nego koristi kernel `RENAME_NOREPLACE`;
- macOS regularne staging datoteke koriste ekskluzivno hard-link + unlink premještanje, pa konkurentno odredište ne može biti tiho prepisano;
- rekurzivno lokalno brisanje direktorije čita kroz verificirani otvoreni handle i ponovno provjerava filesystem identitet prije svake destruktivne faze;
- privatni SFTP ključ dobiva bounded, stabilno provjeren `0600` session snapshot prije nego ga OpenSSH koristi;
- Linux/macOS novi manager više ne briše startup artefakte druge aktivne SFTP terminalske instance;
- Windows crash-cleanup SFTP artefakata pomaknut je iza provjere sigurnog/no-redirect session direktorija;
- CI regresije zaključavaju filesystem hardening invarijante, a postojeći release-version guard zahtijeva novi semantički broj za produkcijske promjene nakon taga.

## Izgradnja iz izvornog koda

Za produkcijski build potreban je podržani Go toolchain koji zadovoljava build skripte, Python 3 za audite te platformski alati za odgovarajući paket. Produkcijski buildovi koriste lokalni toolchain, `GOPROXY=off`, `GOSUMDB=off` i ne preuzimaju vanjske Go module.

Prije produkcijskog builda:

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

## Automatizirane provjere

Glavni quality skup uključuje:

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

GitHub Actions dodatno gradi i verificira Windows x64/x86, Linux amd64/arm64/i386 te macOS Universal pakete. Javno izdanje nastaje tek nakon prolaska produkcijskih gateova.

## Release integritet i GitHub Packages

`scripts/publish_release.ps1` provjerava vezu taga i commita, dopušteni skup asseta, veličinu i SHA-256 digest te odbija slijepi overwrite različitog sadržaja.

Promjena `VERSION` na `main` pokreće produkcijski release workflow. Nakon uspješnih quality/platformskih gateova isti workflow izrađuje `ByFTP.Windows` NuGet paket i objavljuje ga u GitHub Packages s istom semantičkom verzijom. Ponovno izvođenje iste verzije koristi sigurno ponašanje bez dupliciranja paketa.

## Struktura repozitorija

```text
README.md              glavni produkcijski pregled
LICENSE                vlasnička ByFTP licenca
CHANGELOG.md            javna povijest stabilnih izdanja
VERSION                 jedini izvor produkcijske verzije
BUILD-WINDOWS.*         Windows build ulazi
cmd/                    aplikacija i instalacijski lifecycle
internal/               engine, remote, transfer, config i sigurnosni moduli
build/                  službeni resursi
scripts/                build, audit, bundle i release alati
docs/                   detaljna dokumentacija
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

ByFTP je vlasnički/source-available softver. Objavljeni izvorni kod može se pregledavati radi transparentnosti, sigurnosne provjere i evaluacije, ali prava izmjene, redistribucije, rebrandinga, prodaje ili sublicenciranja uređena su datotekom [LICENSE](LICENSE).

## Podrška i sigurnosne prijave

Za uobičajene probleme pogledajte [PODRSKA.md](docs/PODRSKA.md). Za osjetljive sigurnosne probleme slijedite [SIGURNOST.md](docs/SIGURNOST.md). U javni issue ne stavljajte lozinke, privatne ključeve, stvarne produkcijske hostove ili druge povjerljive podatke.
