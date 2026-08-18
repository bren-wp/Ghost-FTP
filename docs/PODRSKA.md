# ByFTP — podrška

Za ponovljivu programsku grešku koristite GitHub predložak **Prijava greške** i navedite samo sanitizirane tehničke podatke.

## Podržane platforme

- Windows 10/11 x64 — puni Win32 GUI
- Windows 10/11 x86 — puni Win32 GUI
- Linux amd64/arm64/i386 — terminalni DEB paket
- macOS Intel/Apple Silicon — Universal PKG, Finder launcher + terminalni klijent

## Autentikacija u 2.16.0

| Način | Windows | Linux/macOS |
|---|---|---|
| FTP/FTPS lozinka | da | da |
| SFTP privatni ključ bez passphrasea | da | da |
| SFTP lozinka | da | još ne |
| SFTP ključ s passphraseom | da | još ne |
| SFTP host-key potvrda | da | da |

Linux/macOS nepodržani SFTP način odbija prije mrežnog pokušaja. Ne prijavljuje lažno stanje „povezano”.

## Ako se veza ne uspostavi

Provjerite redom:

1. protokol — FTP, eksplicitni FTPS, implicitni FTPS ili SFTP
2. host/adresu
3. port
4. korisničko ime
5. lozinku ili privatni ključ
6. je li odgovarajući servis stvarno pokrenut na serveru
7. za Windows SFTP — je li instaliran Windows OpenSSH Client
8. za SFTP — potvrđujete li očekivani host-key fingerprint

ByFTP 2.16.0 čuva timeout/cancel kao tipizirani uzrok i prikazuje različite hrvatske poruke za auth failure, DNS/host, odbijeni port, timeout, host-key scan i sigurnosni pin problem.

Windows nakon neuspjelog pokušaja ne briše upravo unesenu vjerodajnicu; možete ispraviti host/port i pokušati ponovno. Tajna se briše iz edit kontrole nakon potvrđenog uspješnog spajanja.

## Podaci za bug report

Korisno je navesti:

- ByFTP verziju
- platformu i arhitekturu
- odabrani protokol
- je li problem nastao prije ili poslije host-key potvrde
- točan hrvatski tekst poruke greške
- je li problem reproducibilan s testnim serverom
- je li korišten Setup, Portable, DEB ili PKG paket

Ne objavljujte:

- lozinke ili zaporke privatnog ključa
- SSH privatne ključeve
- povjerljive produkcijske hostove i korisnička imena
- podatke klijenata
- stvarne povjerljive putanje ili nazive datoteka
- sadržaj DPAPI/runtime credential vrijednosti

Za sigurnosno osjetljiv problem slijedite [SIGURNOST.md](SIGURNOST.md) umjesto javnog issuea.
