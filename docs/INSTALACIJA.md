# Instalacija ByFTP-a

Ovaj dokument opisuje službene pakete i stvarna ograničenja platformi. Kanonska verzija nalazi se u [`../VERSION`](../VERSION), a javni paketi objavljuju se kroz [GitHub Releases](https://github.com/bren-wp/by-ftp/releases).

## Windows

### x64

Za većinu računala koristite:

- `ByFTP-<verzija>-Setup-x64.exe` — preporučena instalacija
- `ByFTP-<verzija>-Portable-x64.exe` — pokretanje bez instalacije
- `ByFTP-<verzija>-Windows-x64.zip` — Setup + Portable + dokumentacija

### x86 / 32-bitni Windows

Za 32-bitni Windows ili kada je potreban 32-bitni proces koristite:

- `ByFTP-<verzija>-Setup-x86.exe`
- `ByFTP-<verzija>-Portable-x86.exe`
- `ByFTP-<verzija>-Windows-x86.zip`

Windows Setup interno ugrađuje odgovarajući program za uklanjanje. Standalone `Uninstall-*.exe` nije javni release asset.

### Windows preduvjeti

- Windows 10 ili Windows 11
- sistemski `curl.exe` za FTP/FTPS
- Windows OpenSSH Client za SFTP

Ako SFTP komponenta nedostaje, ByFTP prikazuje hrvatsku poruku koja upućuje na instalaciju Windows OpenSSH Client značajke.

## Linux

Službeni DEB paketi:

- `ByFTP-<verzija>-Linux-amd64.deb`
- `ByFTP-<verzija>-Linux-arm64.deb`
- `ByFTP-<verzija>-Linux-i386.deb`

Primjer instalacije:

```bash
sudo apt install ./ByFTP-<verzija>-Linux-amd64.deb
```

Paket instalira:

- `/usr/bin/byftp`
- desktop launcher s `Terminal=true`
- ByFTP ikonu

Ovisnosti paketa: `ca-certificates`, `curl`, `openssh-client`.

Linux izdanje koristi terminalno sučelje i isti engine/transfer/security core kao Windows izdanje. Prije izrade svakog službenog DEB skupa GitHub CI na Linux runneru izvršava Go testove, process-level FTP/SFTP connect smoke testove i `go vet`.

## macOS

Službeni paket:

- `ByFTP-<verzija>-macOS-Universal.pkg`

Universal paket sadrži Intel x86_64 i Apple Silicon arm64 binarij. Instalira:

- `/usr/local/bin/byftp`
- `/Applications/ByFTP.app`

`ByFTP.app` je Finder launcher koji otvara stvarni ByFTP terminalni klijent u Terminal aplikaciji. Prije izrade službenog PKG-a GitHub CI na stvarnom macOS runneru izvršava Go testove, process-level connect smoke testove i `go vet`.

Paket nije Apple Developer ID potpisan dok nije dostupan stvarni Brendigo Apple certifikat. Ne zaobilazite macOS sigurnosne provjere na neprovjerenim kopijama paketa; usporedite SHA-256 sa službenim `SHA256.txt`.

## Podržana autentikacija u 2.16.1

| Način | Windows | Linux/macOS |
|---|---|---|
| FTP/FTPS lozinka | da | da |
| SFTP privatni ključ bez passphrasea | da | da |
| SFTP lozinka | da | trenutačno ne |
| SFTP ključ s passphraseom | da | trenutačno ne |
| SFTP host-key potvrda | da | da |

Linux/macOS namjerno odbija nepodržani SFTP auth prije mrežnog pokušaja. To sprječava lažno stanje uspješnog povezivanja. Windows SFTP password/passphrase tok koristi OpenSSH bez `sftp -b`, jer ta opcija uključuje `BatchMode=yes` i može blokirati AskPass autentikaciju.

## Kada ByFTP prikazuje „Povezano”

ByFTP ne označava vezu uspješnom samo zato što je `curl` ili `sftp` proces pokrenut. `remote.Manager.Connect()` prvo mora dovršiti autentikaciju i uspješno pročitati početni udaljeni direktorij (`List` probe). Tek nakon toga engine vraća `Connected=true`, a Windows UI prikazuje stanje **POVEZANO**.

Kod neuspješnog Windows pokušaja unesena lozinka/passphrase ostaje u zaključanom polju za ponovni pokušaj i briše se tek nakon stvarno potvrđene veze. SFTP host-key trust korak može ponovno koristiti upravo unesenu vjerodajnicu bez ponovnog upisivanja.

## Provjera SHA-256

`SHA256.txt` pokriva sve javne ByFTP release pakete i zajedničke metapodatke. Na Windowsu možete koristiti:

```powershell
Get-FileHash .\ByFTP-<verzija>-Setup-x64.exe -Algorithm SHA256
```

Na Linuxu/macOS-u:

```bash
sha256sum ByFTP-<verzija>-Linux-amd64.deb
```

ili na macOS-u:

```bash
shasum -a 256 ByFTP-<verzija>-macOS-Universal.pkg
```

## Što nije javni release asset

ByFTP ne objavljuje kao zasebne custom assete:

- `verification.txt`
- `ByFTP-<verzija>-Source.zip`
- `ByFTP-<verzija>-Uninstall-*.exe`

Interni verifikacijski izvještaji ostaju CI/build dokaz, a Windows uninstaller ostaje dio Setup instalacije. GitHub automatski prikazuje vlastite `Source code (zip)` i `Source code (tar.gz)` poveznice za svaki tag. To je ugrađena GitHub funkcija i nije dodatni ByFTP build artefakt.
