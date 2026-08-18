# Instalacija ByFTP-a

Ovaj dokument opisuje službene pakete i stvarna ograničenja platformi. Kanonska verzija nalazi se u [`../VERSION`](../VERSION), a javni paketi objavljuju se kroz [GitHub Releases](https://github.com/bren-wp/by-ftp/releases).

## Windows

### x64

Za većinu računala koristite:

- `ByFTP-<verzija>-Setup-x64.exe` — preporučena instalacija
- `ByFTP-<verzija>-Portable-x64.exe` — pokretanje bez instalacije
- `ByFTP-<verzija>-Windows-x64.zip` — Setup + Portable + dokumentacija

### x86

Za 32-bitni Windows ili kada je potreban 32-bitni proces koristite:

- `ByFTP-<verzija>-Setup-x86.exe`
- `ByFTP-<verzija>-Portable-x86.exe`
- `ByFTP-<verzija>-Windows-x86.zip`

Windows Setup interno ugrađuje odgovarajući program za uklanjanje. Standalone `Uninstall-*.exe` više nije javni release asset.

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

Linux izdanje koristi terminalno sučelje i isti engine/transfer/security core kao Windows izdanje.

## macOS

Službeni paket:

- `ByFTP-<verzija>-macOS-Universal.pkg`

Universal paket sadrži Intel x86_64 i Apple Silicon arm64 binarij. Instalira:

- `/usr/local/bin/byftp`
- `/Applications/ByFTP.app`

`ByFTP.app` je Finder launcher koji otvara stvarni ByFTP terminalni klijent u Terminal aplikaciji.

Paket nije Apple Developer ID potpisan dok nije dostupan stvarni Brendigo Apple certifikat. Ne zaobilazite macOS sigurnosne provjere na neprovjerenim kopijama paketa; usporedite SHA-256 sa službenim `SHA256.txt`.

## Podržana autentikacija u 2.16.0

| Način | Windows | Linux/macOS |
|---|---|---|
| FTP/FTPS lozinka | da | da |
| SFTP privatni ključ bez passphrasea | da | da |
| SFTP lozinka | da | ne u 2.16.0 |
| SFTP ključ s passphraseom | da | ne u 2.16.0 |
| SFTP host-key potvrda | da | da |

Linux/macOS namjerno odbija nepodržani SFTP auth prije mrežnog pokušaja. To sprječava lažno stanje uspješnog povezivanja.

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

ByFTP više ne objavljuje kao zasebne custom assete:

- `verification.txt`
- `ByFTP-<verzija>-Source.zip`
- `ByFTP-<verzija>-Uninstall-*.exe`

GitHub automatski prikazuje vlastite `Source code (zip)` i `Source code (tar.gz)` poveznice za svaki tag. To nije moguće ukloniti iz pojedinačnog GitHub Releasea i nije dodatni ByFTP build artefakt.
