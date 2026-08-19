<p align="center">
  <img src="docs/slike/byftp-zaglavlje.png" alt="ByFTP — FTP, FTPS i SFTP klijent" width="900">
</p>

<p align="center">
  <strong>FTP bez komplikacija. Više kontrole nad vašim hostingom.</strong><br>
  ByFTP je Brendigo klijent za FTP, FTPS i SFTP namijenjen brzom i sigurnom upravljanju web datotekama, shared-hosting računima i svakodnevnim prijenosima — bez oglasa, obaveznog cloud računa i aplikacijske telemetrije.
</p>

<p align="center">
  <a href="https://github.com/bren-wp/by-ftp/releases"><strong>⬇ Preuzmi ByFTP</strong></a> ·
  <a href="https://github.com/users/bren-wp/packages?repo_name=by-ftp"><strong>Packages</strong></a> ·
  <a href="docs/SHARED-HOSTING.md"><strong>Shared hosting</strong></a> ·
  <a href="docs/INSTALACIJA.md"><strong>Brzi početak</strong></a> ·
  <a href="docs/PODRSKA.md"><strong>Pomoć</strong></a> ·
  <a href="docs/SIGURNOST.md"><strong>Sigurnost</strong></a>
</p>

<p align="center">
  <a href="../../actions/workflows/ci.yml"><img alt="Provjere" src="../../actions/workflows/ci.yml/badge.svg"></a>
</p>

# ByFTP

ByFTP je klijent za korisnike koji žele otvoriti hosting, pronaći `public_html`, prenijeti datoteke i nastaviti raditi bez nepotrebnih koraka. Windows izdanje koristi izvorni dvopanelni Win32 GUI, dok Linux i macOS koriste terminalno sučelje nad istim Go engineom.

**Trenutačno izdanje: 1.0.7**

## Što ByFTP radi

- **FTP, eksplicitni FTPS, implicitni FTPS i SFTP** u jednom klijentu.
- **Shared hosting bez putanjskih iznenađenja** — FTP/FTPS rad zadržava login/home semantiku servera.
- **Dvopanelni Windows rad** — lokalne datoteke lijevo, udaljene desno.
- **Transfer queue** — više prijenosa, pause/resume, cancel i retry.
- **Sigurniji overwrite** — privremeni upload, ponovna provjera odredišta, backup i rollback.
- **SFTP host-key pinning** — prvi kontakt traži potvrdu SHA-256 fingerprinta, a promjena pina blokira vezu.
- **Bez aplikacijske telemetrije i oglasa** — ByFTP nema analytics SDK ni obavezni Brendigo cloud račun.

## Shared hosting

Za tipični shared-hosting račun obično su potrebni host, korisničko ime, lozinka te protokol/port. Korisničko ime može biti i u obliku `korisnik@domena.hr`.

Za FTP/FTPS logička putanja `/public_html` predstavlja `public_html` unutar direktorija u koji je server smjestio korisnika nakon prijave. ByFTP raw FTP naredbe koriste isti login/home namespace kao listing i upload/download, pa početni `/` ne pretvara korisničku putanju u proizvoljni fizički root servera.

ByFTP prvo pokušava strojno čitljiv `MLSD`. Ako ga stariji ili nestandardni shared-hosting FTP servis ne podržava ili vrati neupotrebljiv format, klijent prelazi na `LIST` i taj fallback pamti do kraja sesije.

Detaljni vodič: [ByFTP za shared hosting](docs/SHARED-HOSTING.md).

## Produkcijska podrška

| Platforma | Distribucija | Arhitekture | Sučelje |
|---|---|---|---|
| Windows 10/11 | Setup EXE, Portable EXE, ZIP | x64, x86 | puni dvopanelni GUI |
| Linux | DEB | amd64, arm64, i386 | terminal |
| macOS | Universal PKG | Intel + Apple Silicon | terminal |

### Autentikacija

| Način | Windows | Linux / macOS |
|---|---|---|
| FTP/FTPS lozinka | da | da |
| SFTP privatni ključ bez passphrasea | da | da |
| SFTP lozinka | da | ne, fail-closed |
| SFTP privatni ključ s passphraseom | da | ne, fail-closed |
| SFTP fingerprint provjera | da | da |

Linux/macOS SFTP lozinka i passphrase ne provode se kroz nesigurne command-line argumente ili obične environment varijable. Dok siguran Unix credential broker nije implementiran i testiran, ti načini ostaju namjerno blokirani.

## Preuzimanje

Službene distribucije dostupne su kroz GitHub Releases. GitHub Packages koristi isti kanonski broj verzije iz `VERSION`, tako da runtime, release i Windows paket ostaju u istom release ciklusu.

### Windows

- `ByFTP-<verzija>-Setup-x64.exe` — preporučeno za većinu Windows računala
- `ByFTP-<verzija>-Portable-x64.exe` — pokretanje bez instalacije
- `ByFTP-<verzija>-Windows-x64.zip` — provjereni distribucijski ZIP
- dostupne su i x86 varijante za podržane 32-bitne sustave

### Linux

- `ByFTP-<verzija>-Linux-amd64.deb`
- `ByFTP-<verzija>-Linux-arm64.deb`
- `ByFTP-<verzija>-Linux-i386.deb`

### macOS

- `ByFTP-<verzija>-macOS-Universal.pkg`

Svako izdanje uključuje SHA-256 i release metapodatke za provjeru distribucije.

## Instalacija, nadogradnja i uklanjanje

Za instalaciju, prvi spoj, nadogradnju i uklanjanje otvorite [Vodič za instalaciju i brzi početak](docs/INSTALACIJA.md).

Windows korisnicima preporučujemo Setup x64 paket. Portable izdanje je prikladno kada ne želite klasičnu instalaciju. Linux koristi DEB paket prema arhitekturi, a macOS Universal PKG pokriva Intel i Apple Silicon.

## Upravljanje datotekama

Windows GUI podržava pregled lokalnih i udaljenih datoteka, višestruki odabir, upload/download, stvaranje mapa, preimenovanje, rekurzivno brisanje uz zaštitne limite, CHMOD gdje ga server podržava, profile i transfer queue.

Linux/macOS terminal koristi isti engine za `ls`, `cd`, `mkdir`, `rename`, `delete`, `chmod`, `get`, `put`, `pwd` i transfere.

Pojedinačni upload/download u 1.0.7 zahtijeva konkretnu remote datoteku. Root, `.` i putanja koja završava direktorijskim separatorom odbijaju se prije dodavanja posla u red, dok prijenos cijele mape ostaje zasebna tree-transfer operacija.

## Sigurnost transfera

ByFTP je projektiran tako da greška ne izgleda kao uspjeh. Ključne zaštite uključuju:

- kriptografski nasumične privremene i staging nazive;
- provjeru lokalnih symlink/junction/reparse preusmjeravanja;
- no-replace lokalnu aktivaciju gdje je platformski dostupna;
- privatni byte-for-byte snapshot lokalnog upload izvora;
- SHA-256 provjeru upload snapshota prije i nakon mrežnog čitanja;
- novu provjeru da vidljivi remote `.byftp-part-*` staging objekt nije direktorij ili symlink;
- ponovnu provjeru finalnog remote odredišta neposredno prije rename/backup commita;
- fresh `SkipExisting` odluku nakon dugog uploada;
- vezanje retry/batch poslova uz istu connection identity i transfer generation;
- bounded stdout/stderr mrežnih child procesa;
- timeout i cancel propagaciju;
- FTPS certificate-revocation zaštitu bez globalnog `ssl-no-revoke` gašenja;
- SFTP SHA-256 fingerprint pinning i privatni `known_hosts` lifecycle.

Sigurniji upload snapshot zahtijeva dodatni privremeni lokalni prostor približno veličini datoteke koja se šalje. To je namjeran tradeoff u korist stabilnog sadržaja.

## SFTP RSA sigurnost u 1.0.7

RSA host ključ i algoritam kojim server potpisuje SSH handshake nisu ista stvar. ByFTP i dalje pin-a isti RSA javni ključ kroz `known_hosts` i SHA-256 fingerprint, ali više ne pretvara skenirani tip ključa `ssh-rsa` u prisilni session `HostKeyAlgorithms ssh-rsa`.

Time moderni OpenSSH može za isti pinani RSA ključ pregovarati moderni RSA/SHA-2 potpis. Ed25519 i ECDSA ostaju eksplicitno vezani uz skenirani tip ključa.

## Unix runtime cleanup u 1.0.7

Na Linuxu i macOS-u ByFTP sada traži native executable `curl`. Windows-specifični naziv `curl.exe` više nema prednost u Unix PATH-u, a Windows i dalje koristi fail-closed sistemsku `curl.exe` putanju.

## Privatnost

ByFTP runtime nema oglase, analytics SDK, vanjski crash-reporting servis, obavezni cloud račun ni fiksni Brendigo API kojem bi slao korisničke aktivnosti.

Na Windowsu spremljene profilne tajne koriste DPAPI. Aktivne vjerodajnice ne stavljaju se u command-line argumente mrežnih alata. Produkcijski build zahtijeva isključenu Go telemetriju.

Detalji: [Privatnost](docs/PRIVATNOST.md).

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

## Dokumentacija

**Početak i svakodnevni rad:**

- [Dokumentacijski centar](docs/README.md)
- [ByFTP za shared hosting](docs/SHARED-HOSTING.md)
- [Instalacija i prvi spoj](docs/INSTALACIJA.md)
- [Podrška i rješavanje problema](docs/PODRSKA.md)

**Sigurnost, privatnost i kvaliteta:**

- [Sigurnost](docs/SIGURNOST.md)
- [Privatnost](docs/PRIVATNOST.md)
- [Testiranje i kvaliteta](docs/TESTIRANJE.md)
- [Provjera izdanja](docs/PROVJERA-IZDANJA.md)
- [Potpisivanje distribucija](docs/POTPISIVANJE.md)
- [Obavijesti trećih strana](docs/OBAVIJESTI-TRECIH-STRANA.md)

**Razvoj i održavanje projekta:**

- [Arhitektura](docs/ARHITEKTURA.md)
- [Plan razvoja](docs/PLAN-RAZVOJA.md)
- [Doprinos projektu](docs/DOPRINOS.md)
- [Izdavanje na GitHubu](docs/IZDAVANJE-NA-GITHUBU.md)

## Važna ograničenja

ByFTP ne može jamčiti kompatibilnost sa svakom nestandardnom FTP/SFTP implementacijom ili pravilima svakog hosting providera. Hosting može zasebno ograničiti write, rename, CHMOD, broj konekcija, TLS/SSH algoritme ili pristup pojedinim direktorijima.

Windows binariji neće imati Verified Publisher status bez stvarnog Authenticode certifikata, a macOS paket neće biti Developer ID notariziran bez stvarnog Apple signing identiteta. Projekt te statuse ne simulira.

---

<p align="center"><strong>ByFTP — otvorite hosting, pronađite datoteke i nastavite raditi.</strong></p>
