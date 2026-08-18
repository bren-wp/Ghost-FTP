# ByFTP — testiranje

## Obavezne provjere

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

Platform-specific buildovi:

```powershell
.\BUILD-WINDOWS.ps1
```

```bash
bash scripts/BUILD-LINUX.sh
bash scripts/BUILD-MACOS.sh   # na macOS-u
```

## CI gateovi

GitHub CI prije mergea ima četiri neovisna joba:

1. **Quality / Linux** — asset, hrvatski, version, docs, security, privacy i release auditi; Python release regresije; `go test`, `go test -race`, `go vet`; generiranje release notes.
2. **Windows x64+x86** — puni `BUILD-WINDOWS.ps1`, PE32/PE32+ resursi, manifest, mitigacije i interne verifikacijske datoteke.
3. **Linux DEB** — prvo izvršava `go test ./...` i `go vet ./...` na Linux runneru, zatim stvarno gradi amd64, arm64 i i386 `.deb` te provjerava package/version/architecture metapodatke s `dpkg-deb`.
4. **macOS Universal** — na stvarnom `macos-14` runneru prvo izvršava `go test ./...` i `go vet ./...`, zatim gradi Intel i Apple Silicon binarije, spaja ih `lipo` alatom, radi `.icns`, `ByFTP.app` i Universal `.pkg` te provjerava PKG strukturu.

Merge nije spreman dok **sva četiri** joba nisu zelena.

## Povezivanje

2.16.1 dodatno zaključava stvarni connect put:

- `TestSFTPCommandArgsKeepAskPassEnabled` zahtijeva `BatchMode=no` i zabranjuje `sftp -b`
- `TestCurlFTPProcessSmokeUsesRuntimeSecretAndParsesListing` stvarno pokreće lažni `curl` child proces preko produkcijskog `exec.CommandContext` puta, provjerava runtime tajnu u config stdin-u, MLSD odgovor, parser i wipe-on-close
- `TestSFTPProcessSmokeUsesStdinWithoutBatchMode` stvarno pokreće lažni `sftp` child proces, odbija svaki `-b`, zahtijeva `ls -la` kroz stdin i provjerava parser udaljenog listinga
- oba process smoke testa izvršavaju se na Linuxu i macOS-u prije izrade instalacijskih paketa
- AskPass regresije potvrđuju password/passphrase odabir i odbijanje MFA/OTP/security-key promptova
- IPv6 regresija potvrđuje uklanjanje `[]` prije OpenSSH `HostName`
- non-Windows OpenSSH regresija potvrđuje `sftp`, `ssh-keyscan` i `ssh-keygen` nativne nazive bez `.exe`
- usererror testovi razlikuju timeout, auth, host-key scan i session-closing stanje
- connect se smatra uspješnim tek nakon remote `List` probea

Process smoke testovi **ne kontaktiraju vanjsku mrežu**. Koriste kratkotrajne lokalne testne executable datoteke i produkcijski adapter/process plumbing, pa reproducibilno provjeravaju prijenos konfiguracije, stdin, cleanup i parser bez stvarnih vjerodajnica ili servera.

## Runtime vjerodajnice

- Windows testovi i sigurnosni audit čuvaju DPAPI + trusted-parent AskPass model
- Linux/macOS runtime-secret spremište koristi nasumični token, procesnu mapu i wipe-on-forget
- process-level FTP smoke dokazuje da se aktivna tajna može koristiti u mrežnom adapteru i više nije dostupna nakon `Close()`
- terminalni frontend namjerno odbija SFTP password/passphrase prije mrežnog pokušaja dok Unix AskPass broker nije sigurnosno dovršen
- FTP/FTPS terminalni unos ne prikazuje lozinku (`stty -echo`)

## Profili i trust

Testovi pokrivaju endpoint/account/private-key identity, zabranu prijenosa spremljene lozinke na drugi host/port/korisnika, passphrasea na drugi ključ, očuvanje/reset host-key pina, autoritativno brisanje privatnog ključa i čišćenje mrtvih blobova.

## Remote/session lifecycle

Regresije zahtijevaju:

- aktivna `Operation` referenca čuva adapter živim do `release()`
- `release()` je idempotentan
- disconnect otkazuje session context prije zatvaranja adaptera
- caller timeout vraća kontrolu bez prisilnog closea ispod aktivne operacije
- deferred cleanup blokira reconnect do stvarnog završetka
- drugi disconnect koristi isti close-state

## Transfer i filesystem

Pokriveni su:

- connection generation i cross-server retry blokada
- late-cancel status nakon uspjeha ili `ErrSkipped`
- symlink/junction/reparse traversal
- download staging regular-file provjera
- no-replace backup/rollback
- filesystem-root zabrana brisanja
- recursive depth/item limiti
- veliki direktoriji i redovi do javnih limita
- FTP MLSD/fallback parser, DOS listing i sigurno parsiranje veličina

## Release regresije

`test_release_tools.py` gradi privremene x64 i x86 Windows ZIP fixturee. Pozitivan fixture mora proći; traversal, hash mismatch, nepopisana datoteka te interni standalone uninstaller/verification file moraju pasti.

`verify_bundle.py` provjerava stvarni komprimirani ZIP bez ekstrakcije na disk i zahtijeva točan `BUNDLE-SHA256.txt` manifest.

`audit_release.py` zaključava 13 javnih asseta: šest Windows, tri Linux, jedan macOS te SHA256/release notes/build metadata. Custom Source ZIP, standalone Uninstaller i `verification.txt` ne smiju se vratiti u javni `$assets` blok.

## Verzija i dokumentacija

`audit_version.py` zahtijeva da Windows, Linux, macOS i lokalni buildovi čitaju isti `VERSION` i ugrađuju ga u runtime. `audit_docs.py` provjerava sve lokalne poveznice i da je svaki `docs/*.md` dokument indeksiran u `docs/README.md` i glavnom README-u.
