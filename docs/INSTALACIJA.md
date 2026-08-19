# Instalacija i prvi spoj

**Cilj ovog vodiča je jednostavan: instalirati ByFTP, spojiti se na hosting i doći do svojih web datoteka bez nepotrebnog tehničkog lutanja.**

## 1. Odaberite izdanje za svoj sustav

### Windows

Za većinu korisnika preporučuje se:

- `ByFTP-<verzija>-Setup-x64.exe` — standardna instalacija;
- `ByFTP-<verzija>-Portable-x64.exe` — bez instalacije, praktično za prijenosno korištenje.

x86 paketi postoje za podržane 32-bitne Windows sustave.

### Linux

Odaberite DEB paket prema arhitekturi:

- amd64;
- arm64;
- i386.

### macOS

Koristite `ByFTP-<verzija>-macOS-Universal.pkg`, koji sadrži Intel i Apple Silicon varijantu.

## 2. Provjerite paket prije instalacije

Svako službeno izdanje dolazi sa `SHA256.txt` datotekom.

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

Usporedite rezultat s odgovarajućim checksumom iz izdanja.

## 3. Pripremite FTP podatke

Kod shared hostinga podatke obično nalazite u hosting panelu pod nazivima poput **FTP Accounts**, **FTP pristup**, **File transfer** ili **Connection details**.

Najčešće trebate:

- **Host** — primjerice `ftp.domena.hr`, server hostname ili IP adresa;
- **Korisničko ime** — može biti `korisnik`, ali i `korisnik@domena.hr`;
- **Lozinku**;
- **Protokol**;
- **Port**.

Tipične postavke:

| Vrsta veze | Protokol u ByFTP-u | Uobičajeni port |
|---|---|---|
| klasični FTP | FTP | 21 |
| FTP s TLS-om nakon spajanja | FTPS (eksplicitni) | 21 |
| FTPS od početka veze | FTPS (implicitni) | 990 |
| SSH File Transfer | SFTP | 22 |

Uvijek koristite vrijednosti koje je dao vaš hosting provider ako se razlikuju od uobičajenih.

## 4. Prvi shared-hosting spoj na Windowsu

1. Pokrenite ByFTP.
2. Odaberite protokol.
3. Upišite **Host**.
4. Upišite **Port**.
5. Upišite korisničko ime i lozinku.
6. Kliknite **Poveži**.

ByFTP prikazuje stanje **POVEZANO** tek nakon stvarne autentikacije i početnog udaljenog listinga.

Ako hosting račun nakon prijave otvara korisnički home direktorij, ByFTP taj direktorij tretira kao logički početak FTP prostora. To je namjerno i odgovara tipičnom shared-hosting modelu.

### Gdje je web stranica?

Na mnogim hosting računima web datoteke nalaze se u direktoriju kao što je:

- `public_html`;
- `www`;
- `httpdocs`;
- `htdocs`;
- direktorij konkretne domene.

ByFTP ne pretpostavlja da svi hostinzi koriste isti naziv. Otvorite direktorij koji je vaš provider označio kao web root.

## 5. FTP ili FTPS?

Ako provider nudi FTPS, preporučuje se koristiti TLS varijantu umjesto plain FTP-a.

Za **explicit FTPS** odaberite `FTPS (eksplicitni)` i najčešće port 21.

Za **implicit FTPS** odaberite `FTPS (implicitni)` samo ako hosting izričito kaže da ga podržava; tipični port je 990.

ByFTP ne gasi TLS provjeru certifikata radi lakšeg spajanja. Ako certifikat ne odgovara hostu ili TLS validacija ne prođe, ispravite host/protokol umjesto zaobilaženja zaštite.

## 6. SFTP prvi spoj

Kod SFTP-a prvi kontakt može prikazati SHA-256 fingerprint server ključa. Provjerite fingerprint prema podacima providera prije potvrde.

Windows podržava SFTP lozinku i privatni ključ. Linux/macOS trenutačno podržavaju siguran privatni ključ bez passphrasea; password/passphrase metode ostaju fail-closed dok se ne implementira siguran Unix credential broker.

## 7. Zašto shared hosting ponekad izgleda drukčije

Shared hosting može imati:

- korisnički home/chroot;
- ograničene FTP naredbe;
- MLSD isključen ili nepotpun;
- NAT između FTP servera i interneta;
- zabranu CHMOD-a ili renamea;
- ograničen broj paralelnih konekcija.

ByFTP 1.0.5 je posebno prilagođen tim scenarijima:

- raw FTP naredbe koriste login-home relativne putanje;
- quote operacije ne otvaraju nepotreban data-channel transfer nakon naredbe;
- ako MLSD ne radi, ByFTP prelazi na klasični LIST i pamti fallback do kraja sesije;
- pasivni FTP rad ne vjeruje slijepo PASV IP adresi koju server može pogrešno vratiti kroz NAT.

## 8. Spremanje profila

Na Windowsu možete spremiti profil kako biste se sljedeći put spojili brže.

Spremljene tajne koriste Windows DPAPI i vežu se uz isti endpoint/račun. Promjena hosta, porta, protokola ili korisničkog identiteta ne prenosi staru spremljenu tajnu na novi identitet.

## 9. Nadogradnja

Preporučuje se koristiti službena GitHub izdanja i prije instalacije usporediti checksum.

Produkcijska verzija dolazi iz jedne `VERSION` datoteke, a release workflow provjerava da paket, dokumentacija i GitHub izdanje ne odlutaju na različite brojeve verzije.

## 10. Uklanjanje

### Windows

Koristite standardni ByFTP uninstaller za instalirano izdanje. Portable izdanje uklanja se brisanjem portable datoteke/paketa kada više nije potrebno.

### Linux

DEB paket uklanja se standardnim paketnim alatima distribucije.

### macOS

Uklanjanje ovisi o instalacijskom rasporedu paketa i administratorskim pravilima sustava.

## Ako se ne možete spojiti

Otvorite [Podrška i rješavanje problema](PODRSKA.md). Najčešći uzroci su pogrešan host, pogrešan protokol/port, FTP račun bez prava pristupa, firewall ili TLS postavka hostinga.

---

**ByFTP je spreman kada vaš hosting odgovori stvarnim loginom i listingom — ne samo kada se otvori mrežni proces.**
