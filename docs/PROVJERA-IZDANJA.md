# ByFTP — kontrolna lista izdanja

Ova lista je fail-closed kriterij za javno ByFTP izdanje. Release nije spreman ako je bilo koja primjenjiva stavka neprovjerena.

## Verzija i sadržaj

- [ ] `VERSION` sadrži točnu semantičku verziju
- [ ] `CHANGELOG.md` ima odgovarajući odjeljak
- [ ] README prikazuje isti broj kao `VERSION`
- [ ] README i detaljni dokumenti opisuju istu platform/auth matricu
- [ ] `release_notes.py` generira hrvatske bilješke iz aktualnog CHANGELOG odjeljka
- [ ] GitHub bug predložak i workflow nemaju ručno sinkronizirani semver default

## Go build privatnost

- [ ] `go telemetry off` je izvršen u CI/release okruženju
- [ ] `go telemetry` vraća `off` prije prvog Go testa/builda
- [ ] produkcijske build skripte odbijaju drugi telemetry način
- [ ] build skripte se ne oslanjaju na običnu `GOTELEMETRY` OS env varijablu
- [ ] `GOTOOLCHAIN=local`, `GOPROXY=off` i `GOSUMDB=off` ostaju aktivni

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
- [ ] release workflow ima vlastiti production quality/race job

## Connect regresije

- [ ] SFTP command args sadrže `BatchMode=no` i ne vraćaju batch način koji gasi AskPass
- [ ] AskPass password/passphrase testovi prolaze na Windows kodu
- [ ] MFA/OTP/nepoznati AskPass prompt ne dobiva tajnu
- [ ] bracketirani IPv6 OpenSSH test prolazi
- [ ] curl/SFTP timeout/cancel context propagacija je zaključana auditom
- [ ] Windows unos vjerodajnice ostaje dostupan za retry/trust continuation do potvrđenog `Connected`
- [ ] connect se smatra uspješnim tek nakon početnog remote `List` probea
- [ ] process-level FTP/FTPS smoke prolazi
- [ ] process-level SFTP smoke prolazi na Unix runneru

## Windows

- [ ] `BUILD-WINDOWS.ps1` zahtijeva Go 1.26.5+
- [ ] `BUILD-WINDOWS.ps1` zahtijeva `go telemetry=off`
- [ ] x64 produkcijski build prolazi
- [ ] x86 produkcijski build prolazi
- [ ] x64 javni binariji imaju očekivani PE32+ format
- [ ] x86 javni binariji imaju očekivani PE32 format
- [ ] x64 ima HIGH_ENTROPY_VA, DYNAMIC_BASE, NX_COMPAT i TERMINAL_SERVER_AWARE
- [ ] x86 ima DYNAMIC_BASE, NX_COMPAT i TERMINAL_SERVER_AWARE
- [ ] oba manifesta imaju ispravni `processorArchitecture`
- [ ] oba Windows ZIP-a sadrže Setup, Portable, release metadata i kompletnu Markdown dokumentaciju
- [ ] `verify_bundle.py --arch x64` prolazi nad konačnim x64 ZIP-om
- [ ] `verify_bundle.py --arch x86` prolazi nad konačnim x86 ZIP-om
- [ ] lokalni `dist/` root prikazuje samo javne izlaze; tehnički build dokazi su izdvojeni u `dist/internal/`

## Linux

- [ ] `BUILD-LINUX.sh` zahtijeva Go 1.26.5+
- [ ] `BUILD-LINUX.sh` zahtijeva `go telemetry=off`
- [ ] amd64 DEB se gradi
- [ ] arm64 DEB se gradi
- [ ] i386 DEB se gradi
- [ ] `dpkg-deb` potvrđuje package=`byftp`, točnu verziju i arhitekturu
- [ ] paket instalira `/usr/bin/byftp` i terminalni desktop launcher
- [ ] runtime FTP/FTPS tajna ostaje procesna i briše se pri closeu
- [ ] Linux job izvršava Go testove i `go vet` prije builda paketa

## macOS

- [ ] `BUILD-MACOS.sh` zahtijeva Go 1.26.5+
- [ ] `BUILD-MACOS.sh` zahtijeva `go telemetry=off`
- [ ] amd64 i arm64 binariji uspješno se spajaju u Universal binarij
- [ ] `.icns` resurs prolazi `iconutil`
- [ ] `/Applications/ByFTP.app` i `/usr/local/bin/byftp` ulaze u PKG
- [ ] `pkgbuild` proizvodi valjani Universal PKG
- [ ] `pkgutil --expand` potvrđuje package strukturu
- [ ] macOS job izvršava Go testove i `go vet` na stvarnom macOS runneru prije builda

## Release workflow arhitektura

- [ ] automatski okidač je promjena `VERSION` na `main`
- [ ] tag koji publisher izradi ne pokreće drugi release workflow
- [ ] manualni `workflow_dispatch` ostaje dostupan za siguran rerun
- [ ] svi release runovi koriste jednu `byftp-release` concurrency grupu
- [ ] `publish` ovisi o quality + Windows + Linux + macOS jobu
- [ ] publish staging prije metapodataka sadrži točno 10 platformskih paketa
- [ ] dodatna ili nedostajuća staging datoteka zaustavlja release

## Javni GitHub Release

Točan custom asset ugovor ima 13 stavki:

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

- [ ] nema dodatnih custom asseta izvan allowlista
- [ ] `SHA256.txt` pokriva svih 10 platformskih paketa + release notes/build metadata
- [ ] `BUILD-METADATA.txt` bilježi release quality gate
- [ ] tag se razrješava na točan release commit
- [ ] rerun postojeće assete uspoređuje po veličini + GitHub SHA-256 digestu
- [ ] mismatch zaustavlja izdanje, a nedostajući potvrđeni asset može se dopuniti
- [ ] kompletni Actions release artefakt je spremljen
- [ ] `ByFTP.Windows` GitHub Package sadrži samo aktualne Windows javne pakete, dokumentaciju i release metapodatke

## Potpisi

- [ ] Windows Authenticode status se tvrdi samo ako je paket stvarno potpisan pravim Brendigo certifikatom
- [ ] macOS Developer ID/notarizacija se tvrdi samo ako je paket stvarno potpisan/notariziran pravim Apple identitetom
- [ ] bez certifikata dokumentacija jasno kaže da paket nije Verified Publisher/Developer ID

## Produkcijski smoke sa stvarnim poslužiteljem

Automatizirani process smoke testovi koriste lokalne fake procese i ne zamjenjuju završni pre-release test protiv kontroliranog stvarnog poslužitelja.

- [ ] Windows x64 FTP smoke
- [ ] Windows x64 eksplicitni FTPS smoke
- [ ] Windows x64 SFTP password smoke
- [ ] Windows x64 SFTP key smoke
- [ ] Windows x86 osnovni smoke gdje je dostupan odgovarajući sustav
- [ ] Linux FTP/FTPS smoke
- [ ] Linux SFTP key smoke
- [ ] macOS FTP/FTPS smoke
- [ ] macOS SFTP key smoke

Nikada ne stavljati produkcijske vjerodajnice ili privatne ključeve u CI fixturee, logove ili javne issue/PR poruke.
