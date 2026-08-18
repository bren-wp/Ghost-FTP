# ByFTP — podrška

Za ponovljivu programsku grešku koristite GitHub predložak **Prijava greške** i navedite samo sanitizirane tehničke podatke.

## Podržane platforme

- Windows 10/11 x64 — puni Win32 GUI
- Windows 10/11 x86 — puni Win32 GUI
- Linux amd64/arm64/i386 — terminalni DEB paket
- macOS Intel/Apple Silicon — Universal PKG, Finder launcher + terminalni klijent

## Podržana autentikacija

| Način | Windows | Linux/macOS |
|---|---|---|
| FTP lozinka | da | da |
| FTPS lozinka | da | da |
| SFTP privatni ključ bez passphrasea | da | da |
| SFTP lozinka | da | trenutačno ne |
| SFTP ključ s passphraseom | da | trenutačno ne |
| SFTP host-key potvrda | da | da |

Linux/macOS nepodržani SFTP način odbija prije mrežnog pokušaja. ByFTP ne prikazuje lažno stanje „povezano”.

## Ako se veza ne uspostavi

Provjerite redom:

1. protokol — FTP, eksplicitni FTPS, implicitni FTPS ili SFTP
2. adresu poslužitelja
3. port
4. korisničko ime
5. lozinku ili privatni ključ
6. je li odgovarajući servis stvarno pokrenut na serveru
7. firewall/NAT pravila ako koristite FTP/FTPS
8. za Windows SFTP — je li instaliran Windows OpenSSH Client
9. za SFTP — potvrđujete li očekivani host-key fingerprint
10. koristi li Linux/macOS podržani SFTP način s privatnim ključem bez passphrasea

ByFTP čuva timeout/cancel kao tipizirani uzrok i razlikuje autentikacijsku grešku, DNS/host problem, odbijeni port, timeout, host-key scan i sigurnosni pin problem.

Windows nakon neuspjelog pokušaja ne briše upravo unesenu vjerodajnicu; možete ispraviti podatke i pokušati ponovno. Tajna se briše iz edit kontrole nakon potvrđenog uspješnog spajanja.

## ByFTP kaže „Povezano”, ali ne vidim datoteke

Stanje **POVEZANO** nastaje tek nakon uspješne autentikacije i početnog `List` probea. Ako se kasnije prikaz udaljene mape ne osvježava:

- provjerite dozvole korisničkog računa na udaljenoj putanji
- provjerite je li putanja još dostupna
- pokušajte ručno osvježavanje
- zabilježite točan hrvatski tekst greške
- ne šaljite stvarne povjerljive nazive datoteka u javni issue

## FTP/FTPS problemi

Kod FTP/FTPS provjerite:

- odgovara li eksplicitni/implicitni FTPS način konfiguraciji servera
- je li port točan
- dopušta li firewall pasivne FTP veze
- je li TLS certifikat poslužitelja valjan u sistemskom trust storeu

ByFTP ne nasljeđuje proizvoljne proxy/TLS override varijable za curl adapter.

## SFTP problemi

Kod SFTP-a provjerite:

- port i korisnika
- postoji li podržani privatni ključ
- odgovara li host-key fingerprint očekivanoj vrijednosti
- je li se host ključ poslužitelja promijenio nakon administratorske promjene
- postoji li sistemski OpenSSH alat

Promijenjeni spremljeni host-key pin ByFTP tretira kao sigurnosni problem i ne bi ga trebalo automatski prihvatiti bez neovisne provjere.

## Instalacijski problemi

### Windows

Za klasičnu instalaciju koristite Setup paket koji odgovara arhitekturi. Za uklanjanje koristite standardni Windows **Postavke → Aplikacije → Instalirane aplikacije → ByFTP** lifecycle.

Ako Setup ne pokreće aplikaciju nakon nadogradnje, navedite:

- ByFTP verziju
- x64 ili x86
- Windows verziju
- je li postojeća ByFTP instanca bila pokrenuta tijekom nadogradnje
- točan tekst instalacijske poruke

### Linux

Provjerite:

```bash
dpkg --print-architecture
```

i koristite odgovarajući DEB. Paket zahtijeva sistemski `curl`, `openssh-client` i `ca-certificates`.

### macOS

Universal PKG podržava Intel i Apple Silicon. Trenutačni paket nije Developer ID potpisan/notariziran bez stvarnog Brendigo Apple identiteta, pa korisnik treba posebno paziti da paket dolazi iz službenog izdanja i da SHA-256 odgovara.

## Provjera preuzetog paketa

Prije prijave da je paket oštećen, usporedite SHA-256 s `SHA256.txt` iz istog izdanja.

Windows:

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

## Podaci za bug report

Korisno je navesti:

- ByFTP verziju
- platformu i arhitekturu
- odabrani protokol
- vrstu paketa: Setup, Portable, DEB ili PKG
- je li problem nastao prije ili poslije host-key potvrde
- točan hrvatski tekst poruke greške
- je li problem reproducibilan s testnim serverom
- korake za reprodukciju bez osjetljivih podataka

Ne objavljujte:

- lozinke ili zaporke privatnog ključa
- SSH privatne ključeve
- povjerljive produkcijske hostove i korisnička imena
- podatke klijenata
- povjerljive putanje ili nazive datoteka
- sadržaj DPAPI/runtime credential vrijednosti

## Build problem pri vlastitoj izgradnji

Produkcijski build zahtijeva Go 1.26.5+ i stvarno ugašenu Go toolchain telemetriju. Prije builda pokrenite:

```bash
go telemetry off
```

Ako skripta odbije build jer `go telemetry` nije `off`, to je namjerni fail-closed privacy gate.

## Sigurnosna prijava

Za sigurnosno osjetljiv problem slijedite [SIGURNOST.md](SIGURNOST.md) umjesto javnog issuea. U javni issue ne stavljajte exploit detalje koji otkrivaju aktivne produkcijske tajne ili stvarne podatke klijenata.
