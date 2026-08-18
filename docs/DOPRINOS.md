# Doprinos projektu ByFTP

ByFTP je vlasnički/source-available softver tvrtke Brendigo. Objava izvornog koda ne daje opće pravo izmjene, redistribucije, rebrandinga ili izrade izvedenica.

## Potrebno je odobrenje

Issuei, prijave grešaka i prijedlozi funkcija su dobrodošli. Izmjene izvornog koda i pull requestovi dopušteni su samo kada ih Brendigo izričito zatraži ili odobri.

## Pravila za ovlaštene doprinose

- Windows Win32 GUI mora ostati izvorni desktop; ne vraćati browser/localhost UI
- Linux/macOS terminalni frontend mora koristiti isti `api.Engine`, remote i transfer core; ne stvarati paralelni FTP engine
- ne dodavati telemetriju, analitiku, oglašavanje, remote crash reporting ili skrivene vanjske API pozive
- ne dodavati vanjski Go modul bez zasebne arhitekturne/sigurnosne provjere
- ne stavljati plaintext lozinke/passphrase u command line, trajne logove ili privremene datoteke
- na Windowsu očuvati DPAPI i trusted-parent AskPass granicu
- na Linuxu/macOS-u aktivne tajne držati samo u procesnom runtime spremištu ili budućem jednako jakom brokeru
- SFTP `commandArgs()` ne smije koristiti `sftp -b`; mora ostati kompatibilan s `BatchMode=no` AskPass tokom
- AskPass ne smije automatski odgovoriti na nepoznat MFA/OTP/security-key prompt
- očuvati SFTP host-key provjeru, endpoint binding i IPv6 normalizaciju
- očuvati download staging, filesystem-root, reparse/symlink i state safe-open granice
- očuvati connection generation, connection identity i cross-server retry blokadu
- preferirati tipizirane Go interfacee umjesto generičkog JSON dispatcha
- korisničke poruke i dokumentaciju pisati na hrvatskom
- novu detaljnu dokumentaciju dodati u `docs/README.md` i glavni README
- aktualni broj izdanja ne hardkodirati izvan kanonskog `VERSION`/auditiranih prikaza
- Windows, Linux i macOS build moraju čitati isti `VERSION`
- ne zaobilaziti `publish_release.ps1`, `verify_bundle.py` ili platform CI gateove
- javni release ne smije ponovno dodati custom Source ZIP, standalone Uninstaller ili `verification.txt`
- ne uklanjati ByFTP/Brendigo identitet bez pisanog odobrenja
- kod treće strane zahtijeva potvrdu licencne kompatibilnosti

## Prije ovlaštenog pull requesta

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

Platform-specific provjere:

```powershell
.\BUILD-WINDOWS.ps1
```

```bash
bash scripts/BUILD-LINUX.sh
bash scripts/BUILD-MACOS.sh   # macOS
```

Merge nije spreman dok quality, Windows x64+x86, Linux DEB i macOS Universal PKG GitHub Actions jobovi nisu zeleni.
