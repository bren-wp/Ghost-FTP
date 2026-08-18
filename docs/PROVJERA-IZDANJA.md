# ByFTP — kontrolna lista izdanja

## Verzija i sadržaj

- [ ] `VERSION` sadrži točnu semantičku verziju
- [ ] `CHANGELOG.md` ima odgovarajući odjeljak
- [ ] README i svi detaljni dokumenti opisuju istu support/auth/release matricu
- [ ] `release_notes.py` generira hrvatske bilješke bez hardkodirane aktualne verzije
- [ ] GitHub bug predložak i workflow nemaju ručno sinkronizirani semver default

## Quality gate

- [ ] asset audit prolazi
- [ ] hrvatski audit prolazi
- [ ] version audit prolazi
- [ ] docs audit prolazi
- [ ] security audit prolazi
- [ ] privacy audit prolazi
- [ ] release audit prolazi
- [ ] Python release regresije prolaze
- [ ] `go test ./...` prolazi
- [ ] `go test -race ./...` prolazi
- [ ] `go vet ./...` prolazi

## Connect regresije

- [ ] SFTP command args sadrže `BatchMode=no` i **ne sadrže `-b`**
- [ ] AskPass password/passphrase testovi prolaze
- [ ] MFA/OTP/nepoznati AskPass prompt ne dobiva tajnu
- [ ] bracketirani IPv6 OpenSSH test prolazi
- [ ] curl/SFTP timeout/cancel context propagacija je zaključana auditom
- [ ] Windows unos vjerodajnice ostaje dostupan za retry/trust continuation do potvrđenog `Connected`
- [ ] connect se smatra uspješnim tek nakon početnog remote `List` probea

## Windows

- [ ] `BUILD-WINDOWS.ps1` uspješno gradi x64 i x86
- [ ] x64 Portable/Setup/internal Uninstaller su različiti PE32+ GUI binariji
- [ ] x86 Portable/Setup/internal Uninstaller su različiti PE32 GUI binariji
- [ ] x64 ima HIGH_ENTROPY_VA, DYNAMIC_BASE, NX_COMPAT i TERMINAL_SERVER_AWARE
- [ ] x86 ima DYNAMIC_BASE, NX_COMPAT i TERMINAL_SERVER_AWARE
- [ ] oba manifesta imaju ispravni `processorArchitecture`
- [ ] oba Windows ZIP-a sadrže Setup, Portable, release metadata i kompletnu Markdown dokumentaciju
- [ ] Windows ZIP **ne sadrži** standalone Uninstaller ni verification report
- [ ] `verify_bundle.py --arch x64` prolazi nad konačnim x64 ZIP-om
- [ ] `verify_bundle.py --arch x86` prolazi nad konačnim x86 ZIP-om

## Linux

- [ ] `BUILD-LINUX.sh` uspješno gradi amd64 DEB
- [ ] uspješno gradi arm64 DEB
- [ ] uspješno gradi i386 DEB
- [ ] `dpkg-deb` potvrđuje package=`byftp`, točnu verziju i arhitekturu
- [ ] paket instalira `/usr/bin/byftp` i terminalni desktop launcher
- [ ] runtime FTP/FTPS tajna ostaje procesna i briše se pri closeu

## macOS

- [ ] `BUILD-MACOS.sh` prolazi na macOS runneru
- [ ] amd64 i arm64 binariji uspješno se spajaju u Universal binarij
- [ ] `.icns` resurs prolazi `iconutil`
- [ ] `/Applications/ByFTP.app` i `/usr/local/bin/byftp` ulaze u PKG
- [ ] `pkgbuild` proizvodi valjani Universal PKG
- [ ] `pkgutil --expand` potvrđuje package strukturu

## Javni GitHub Release

Točan javni asset ugovor ima **13 asseta**:

1. `ByFTP-<v>-Portable-x64.exe`
2. `ByFTP-<v>-Setup-x64.exe`
3. `ByFTP-<v>-Windows-x64.zip`
4. `ByFTP-<v>-Portable-x86.exe`
5. `ByFTP-<v>-Setup-x86.exe`
6. `ByFTP-<v>-Windows-x86.zip`
7. `ByFTP-<v>-Linux-amd64.deb`
8. `ByFTP-<v>-Linux-arm64.deb`
9. `ByFTP-<v>-Linux-i386.deb`
10. `ByFTP-<v>-macOS-Universal.pkg`
11. `SHA256.txt`
12. `RELEASE-NOTES.txt`
13. `BUILD-METADATA.txt`

- [ ] nema custom `verification.txt`
- [ ] nema custom `ByFTP-<v>-Source.zip`
- [ ] nema standalone `Uninstall-*.exe`
- [ ] v2.15.0 cleanup korak uklonio je ta tri stara custom asseta i završno potvrdio odsutnost
- [ ] GitHub automatski Source code ZIP/TAR linkovi tretiraju se kao platform-generated tag poveznice, ne kao ByFTP asseti
- [ ] `SHA256.txt` pokriva svih 10 platformskih paketa + release notes/build metadata
- [ ] tag se razrješava na točan release commit
- [ ] rerun postojeće assete uspoređuje po veličini + GitHub SHA-256 digestu
- [ ] mismatch zaustavlja izdanje, a nedostajući potvrđeni asset može se dopuniti
- [ ] kompletni Actions artefakt je spremljen
- [ ] `ByFTP.Windows` GitHub Package sadrži x64+x86 Windows javne pakete bez standalone uninstallera/verification reporta

## Potpisi i smoke test

- [ ] Windows Authenticode provjeren je samo ako postoji stvarni Brendigo certifikat; bez certifikata dokumentirano je `unsigned`
- [ ] macOS Developer ID/notarizacija provjerena je samo ako postoji stvarni Apple certifikat; bez certifikata ne tvrditi da je paket potpisan
- [ ] Windows x64 smoke-test sa stvarnim FTP/FTPS/SFTP serverom
- [ ] Windows x86 smoke-test gdje je dostupan odgovarajući sustav
- [ ] Linux FTP/FTPS i SFTP-key smoke-test
- [ ] macOS FTP/FTPS i SFTP-key smoke-test
