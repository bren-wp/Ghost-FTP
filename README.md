<p align="center">
  <img src="docs/slike/byftp-zaglavlje.png" alt="ByFTP — FTP, FTPS i SFTP klijent" width="900">
</p>

<p align="center">
  <strong>FTP bez komplikacija. Više kontrole nad vašim hostingom.</strong><br>
  ByFTP je Brendigo klijent za FTP, FTPS i SFTP napravljen za brzo upravljanje web datotekama, shared hosting računima i svakodnevnim prijenosima — bez oglasa, obaveznog cloud računa i aplikacijske telemetrije.
</p>

<p align="center">
  <a href="https://github.com/bren-wp/by-ftp/releases"><strong>⬇ Preuzmi ByFTP</strong></a> ·
  <a href="docs/INSTALACIJA.md"><strong>Brzi početak</strong></a> ·
  <a href="docs/PODRSKA.md"><strong>Pomoć pri spajanju</strong></a> ·
  <a href="docs/SIGURNOST.md"><strong>Sigurnost</strong></a>
</p>

<p align="center">
  <a href="../../actions/workflows/ci.yml"><img alt="Provjere" src="../../actions/workflows/ci.yml/badge.svg"></a>
</p>

# ByFTP

ByFTP je klijent za ljude koji žele **jednostavno otvoriti hosting, pronaći `public_html`, prenijeti web stranicu i nastaviti raditi** — bez nepotrebnih koraka i bez skrivanja važnih sigurnosnih odluka.

Windows izdanje nudi izvorni dvopanelni Win32 GUI za lokalne i udaljene datoteke. Linux i macOS koriste terminalno sučelje nad istim Go engineom.

**Trenutačno izdanje: 1.0.5**

## Zašto koristiti ByFTP

- **Shared hosting na prvom mjestu** — FTP/FTPS rad je usklađen s login/home direktorijem kakav je uobičajen na hosting računima.
- **FTP, explicit FTPS, implicit FTPS i SFTP** — jedan alat za više vrsta poslužitelja.
- **Dvopanelni Windows rad** — lokalne datoteke lijevo, hosting desno.
- **Upload i download s kontroliranim stagingom** — prijenos se ne tretira kao završen dok završna faza nije provjerena.
- **Backup i rollback logika** — sigurniji overwrite kada je uključen backup.
- **Transfer queue** — više prijenosa, pause/resume, cancel i retry.
- **SFTP host-key provjera** — prvi kontakt traži potvrdu fingerprinta, a promjena ključa blokira vezu.
- **Bez aplikacijske telemetrije i oglasa** — ByFTP runtime nema analitiku korištenja ni obavezni Brendigo cloud račun.

## Shared hosting — spojite se u nekoliko koraka

Za tipični shared hosting najčešće su potrebna samo četiri podatka iz hosting panela:

1. **Poslužitelj / Host** — npr. `ftp.vasadomena.hr` ili hostname koji je dao hosting provider.
2. **Korisničko ime** — može biti klasično korisničko ime ili oblik poput `korisnik@domena.hr`.
3. **Lozinka** — FTP lozinka za taj račun.
4. **Port i protokol** — najčešće FTP ili explicit FTPS na portu `21`; implicit FTPS najčešće koristi `990` kada ga hosting izričito nudi.

Nakon klika na **Poveži**, ByFTP ne prikazuje stanje **POVEZANO** samo zato što je pokrenut mrežni alat. Veza mora proći autentikaciju i početni udaljeni listing.

Za FTP/FTPS ByFTP zadržava login/home semantiku servera. To je važno na shared hostingu: logička putanja `/public_html` predstavlja `public_html` unutar direktorija u koji vas je server smjestio nakon prijave, a ne proizvoljni fizički root poslužitelja.

### Kompatibilnost sa starijim FTP serverima

ByFTP prvo koristi strukturirani `MLSD` listing kada ga server pravilno podržava. Ako shared-hosting FTP servis ne podržava MLSD ili vraća neupotrebljiv format, ByFTP prelazi na klasični `LIST` i taj fallback pamti za ostatak sesije.

Pasivni FTP rad koristi curlov EPSV/PASV mehanizam, uz zaštitu od pogrešne privatne PASV adrese koju pojedini NAT/shared-hosting sustavi vraćaju klijentu.

## Produkcijska podrška

| Platforma | Paket | Arhitekture | Sučelje |
|---|---|---|---|
| Windows 10/11 | Setup EXE, Portable EXE, ZIP | x64, x86 | puni dvopanelni GUI |
| Linux | DEB | amd64, arm64, i386 | terminal |
| macOS | Universal PKG | Intel + Apple Silicon | terminal |

### Autentikacija

| Način | Windows | Linux / macOS |
|---|---|---|
| FTP lozinka | da | da |
| FTPS lozinka | da | da |
| SFTP privatni ključ bez passphrasea | da | da |
| SFTP lozinka | da | ne, fail-closed |
| SFTP privatni ključ s passphraseom | da | ne, fail-closed |
| SFTP fingerprint provjera | da | da |

Linux/macOS SFTP lozinka i passphrase nisu umjetno provedeni kroz nesigurne command-line ili obične environment varijable. Dok siguran Unix credential broker nije implementiran i testiran, ti načini ostaju blokirani.

## Preuzimanje

Službena izdanja dostupna su na GitHub Releases stranici projekta.

### Windows

- `ByFTP-<verzija>-Setup-x64.exe` — preporučeno za većinu Windows računala
- `ByFTP-<verzija>-Portable-x64.exe` — pokretanje bez instalacije
- `ByFTP-<verzija>-Windows-x64.zip` — ZIP distribucija
- dostupne su i x86 varijante za podržane 32-bitne sustave

### Linux

- `ByFTP-<verzija>-Linux-amd64.deb`
- `ByFTP-<verzija>-Linux-arm64.deb`
- `ByFTP-<verzija>-Linux-i386.deb`

### macOS

- `ByFTP-<verzija>-macOS-Universal.pkg`

Svako izdanje uključuje checksum i release metapodatke kako biste mogli provjeriti što ste preuzeli.

## Instalacija, nadogradnja i uklanjanje

Za instalaciju i prvi spoj otvorite [Vodič za instalaciju i brzi početak](docs/INSTALACIJA.md).

Windows korisnicima preporučujemo Setup x64 paket. Portable izdanje je praktično kada ne želite klasičnu instalaciju. Linux koristi DEB paket prema arhitekturi, a macOS Universal PKG pokriva Intel i Apple Silicon.

## Upravljanje datotekama

Windows GUI podržava:

- pregled lokalnih i remote datoteka u dva panela;
- upload i download;
- višestruki odabir;
- stvaranje direktorija;
- preimenovanje;
- rekurzivno brisanje uz zaštitne limite;
- CHMOD gdje ga server podržava;
- transfer queue s pause/resume, cancel i retry funkcijama;
- profile za brže ponovno spajanje.

Linux/macOS terminal koristi isti engine za `ls`, `cd`, `mkdir`, `rename`, `delete`, `chmod`, `get`, `put`, `pwd` i transfer queue.

## Sigurnost transfera

ByFTP je projektiran tako da **greška ne izgleda kao uspjeh** i da završna aktivacija datoteke bude stroža od običnog “copy pa se nadamo”.

Ključne zaštite uključuju:

- kriptografski nasumične privremene/staging nazive;
- provjeru lokalnih symlink/junction/reparse preusmjeravanja;
- no-replace lokalnu aktivaciju gdje platforma to podržava;
- privatni byte-for-byte snapshot lokalnog upload izvora;
- SHA-256 provjeru upload snapshota prije i nakon mrežnog čitanja;
- ponovno provjeravanje remote odredišta neposredno prije finalnog rename/backup commita;
- fresh `SkipExisting` odluku nakon dugog uploada;
- vezanje retry/batch poslova uz istu connection identity i transfer generation;
- bounded stdout/stderr mrežnih child procesa;
- timeout i cancel propagaciju;
- FTPS certificate revocation zaštitu bez globalnog `ssl-no-revoke` gašenja;
- SFTP host-key pinning i privatni `known_hosts` lifecycle.

Sigurniji upload snapshot koristi dodatni privremeni lokalni prostor približno veličini datoteke koja se šalje. To je namjeran tradeoff: ByFTP daje prednost stabilnom sadržaju ispred slabije, ali jeftinije path-only provjere.

## Privatnost koja je dio proizvoda

ByFTP runtime nema oglase, analytics SDK, vanjski crash-reporting servis, obavezni cloud račun ni fiksni Brendigo API kojem bi slao vaše aktivnosti.

Na Windowsu spremljene profilne tajne koriste DPAPI. Aktivne vjerodajnice se ne stavljaju u command-line argumente mrežnih alata. Produkcijski build dodatno zahtijeva isključenu Go telemetriju.

Detalji: [PRIVATNOST.md](docs/PRIVATNOST.md).

## Što donosi 1.0.5

1.0.5 je izdanje usmjereno na **stvarni shared-hosting FTP/FTPS rad**:

- FTP control naredbe (`MKD`, `RNFR/RNTO`, `DELE`, `RMD`, `SITE CHMOD`) koriste isti login/home namespace kao URL listing i upload/download;
- uklonjena je mogućnost da početni `/` u raw FTP naredbi ode prema fizičkom server rootu na non-chrooted hostingu;
- quote-only operacije koriste control-channel-only `no-body`, pa uspješna mutacija više ne ovisi o nepotrebnom naknadnom directory data transferu;
- MLSD fallback se nakon uspješnog `LIST`-a pamti za ostatak sesije, što smanjuje ponavljanje neuspjelih naredbi na starijim/shared FTP serverima;
- regresijski testovi uključuju shared-hosting username oblika `account@domain`, login-home `public_html` putanju i MLSD→LIST fallback ponašanje;
- README i kompletna dokumentacija preuređeni su u jasniji, korisnički i benefit-first format.

## Provjera SHA-256

Prije instalacije možete provjeriti preuzeti paket prema službenom `SHA256.txt`.

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

## Dokumentacija

- [Dokumentacijski centar](docs/README.md)
- [Instalacija i prvi spoj](docs/INSTALACIJA.md)
- [Podrška i rješavanje problema](docs/PODRSKA.md)
- [Sigurnost](docs/SIGURNOST.md)
- [Privatnost](docs/PRIVATNOST.md)
- [Arhitektura](docs/ARHITEKTURA.md)
- [Testiranje i kvaliteta](docs/TESTIRANJE.md)
- [Plan razvoja](docs/PLAN-RAZVOJA.md)

## Važna ograničenja

ByFTP ne može jamčiti kompatibilnost sa svakom nestandardnom FTP implementacijom ili pravilima svakog hosting providera. Hosting može zasebno ograničiti write, rename, CHMOD, broj konekcija, TLS verzije ili pristup pojedinim direktorijima.

Windows binariji neće imati Verified Publisher status bez stvarnog Authenticode certifikata, a macOS paket neće biti Developer ID notariziran bez stvarnog Apple signing identiteta. Projekt te statuse ne simulira.

---

<p align="center"><strong>ByFTP — otvorite hosting, pronađite datoteke i nastavite raditi.</strong></p>
