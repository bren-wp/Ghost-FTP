# ByFTP — testiranje

## Cilj

ByFTP testni sustav mora dokazati više od same kompilacije. Obavezno pokriva tipizirani engine, mrežne adaptere, child-process plumbing, transfer/session lifecycle, datotečne granice, privatnost, dokumentaciju i završno pakiranje za svaku podržanu platformu.

## Go toolchain telemetrija prije testova

Produkcijski CI i release workflow prije Go testova izvršavaju:

```bash
go telemetry off
```

i zahtijevaju da:

```bash
go telemetry
```

vrati `off`.

Produkcijske build skripte dodatno odbijaju rad u `local` ili `on` načinu. Obična OS env varijabla nije zamjena za stvarnu Go telemetry postavku.

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

Platformni buildovi:

```powershell
.\BUILD-WINDOWS.ps1
```

```bash
bash scripts/BUILD-LINUX.sh
bash scripts/BUILD-MACOS.sh   # na macOS-u
```

## Pull-request CI gateovi

GitHub CI prije mergea ima četiri neovisna joba.

### Quality / Linux

Provjerava:

- brand resurse
- hrvatski sadržaj
- verzijsku konzistentnost
- dokumentaciju
- sigurnosne invarijante
- privatnost
- release arhitekturu
- Python regresije release alata
- Go unit testove
- Go race detector
- `go vet`
- generiranje release bilješki

### Windows x64+x86

Pokreće puni `BUILD-WINDOWS.ps1`, koji dodatno:

- zahtijeva Go 1.26.5+
- zahtijeva stvarno ugašenu Go telemetriju
- gradi x64 i x86
- provjerava Portable i Setup PE datoteke
- provjerava resurse, manifest i sigurnosne mitigacije
- čuva tehnički verification dokaz samo u CI/internal sloju

### Linux

Na stvarnom Linux runneru prije pakiranja izvršava:

```text
go test ./...
go vet ./...
```

Zatim gradi amd64, arm64 i i386 DEB te `dpkg-deb` provjerava package ID, verziju i arhitekturu.

### macOS

Na stvarnom macOS runneru prije pakiranja izvršava Go testove i `go vet`, zatim gradi Intel + Apple Silicon Universal PKG i provjerava njegovu strukturu s `pkgutil`.

Merge nije spreman dok sva četiri joba nisu zelena.

## Process-level connect smoke testovi

Unit test koji samo uspoređuje niz argumenata nije dovoljan za kritični connect put. Zbog toga `internal/remote/process_connect_smoke_other_test.go` stvarno pokreće lokalne child procese kroz produkcijski `exec.CommandContext` put.

### FTP/FTPS smoke

`TestCurlFTPProcessSmokeUsesRuntimeSecretAndParsesListing` potvrđuje:

- aktivna tajna nije plaintext polje adaptera
- runtime secret token se može otključati samo tijekom aktivne sesije
- curl config ide kroz standardni ulaz
- credential nije command-line argument
- MLSD odgovor prolazi stvarni parser
- `Close()` briše aktivnu runtime tajnu

### SFTP smoke

`TestSFTPProcessSmokeUsesStdinWithoutBatchMode` potvrđuje:

- child proces ne dobiva batch način koji bi ugasio AskPass
- `BatchMode=no` sigurnosni ugovor ostaje očuvan
- `ls -la` naredba stvarno dolazi kroz stdin
- povratni listing prolazi produkcijski parser

Smoke testovi ne kontaktiraju vanjsku mrežu i ne koriste stvarne vjerodajnice. Lažni lokalni procesi služe samo kao deterministični adapter boundary.

## SFTP i AskPass regresije

Testovi dodatno zahtijevaju:

- SHA-256 host-key trust tok
- endpoint-scoped host-key pin
- bracketirani IPv6 host normalizaciju
- nativne OpenSSH nazive alata izvan Windowsa
- password prompt dobiva samo password
- passphrase prompt dobiva samo passphrase
- MFA/OTP/security-key/nepoznati prompt ne dobiva spremljenu tajnu
- timeout/cancel vraća stvarni `context` uzrok

## Profili i vjerodajnice

Regresije pokrivaju:

- endpoint = protokol + normalizirani host + port
- account = endpoint + korisničko ime
- private-key identity = account + putanja ključa
- zabranu automatskog slanja stare lozinke drugom accountu
- zabranu prijenosa passphrasea na drugi privatni ključ
- očuvanje pina na istom endpointu
- reset pina nakon promjene endpointa
- autoritativno brisanje privatnog ključa
- čišćenje credential blobova koji više ne pripadaju profilu

## Remote/session lifecycle

Regresije zahtijevaju:

- svaka aktivna `Operation` drži adapter živim do `release()`
- `release()` je idempotentan
- disconnect prvo blokira nove operacije
- session context se otkazuje prije konačnog zatvaranja
- adapter se ne zatvara ispod aktivne operacije
- caller timeout vraća kontrolu bez rušenja aktivnog cleanup procesa
- deferred cleanup blokira reconnect do stvarnog završetka
- drugi disconnect koristi isti close-state

## Transfer i filesystem

Pokriveni su:

- connection generation i cross-server retry izolacija
- late-cancel nakon uspjeha ili `ErrSkipped`
- lokalni i udaljeni path validation
- symlink/junction/reparse traversal
- staging regular-file provjera
- no-replace backup/rollback
- filesystem-root zabrana rekurzivnog brisanja
- depth/item limiti
- veliki direktoriji i transfer redovi
- FTP MLSD/fallback parser
- DOS-style listing
- sigurno parsiranje veličina

## Release regresije

`test_release_tools.py` gradi kontrolirane Windows ZIP fixturee i provjerava da bundle verifier odbija:

- path traversal
- absolute/drive putanje
- duplicirane putanje
- hash mismatch
- nepopisanu datoteku
- tehničku datoteku koja ne pripada javnom bundleu

`verify_bundle.py` čita ZIP bez ekstrakcije nepouzdanih putanja na disk i zahtijeva potpun `BUNDLE-SHA256.txt`.

`audit_release.py` zaključava:

- jedan autoritativni release okidač
- jednu release concurrency grupu
- zaseban production quality/race job
- točno 10 platformskih staging paketa
- točno 13 završnih custom asseta
- Windows x64/x86, Linux amd64/arm64/i386 i macOS Universal podršku
- centralni fail-closed publisher

## Release workflow nije kopija PR CI-ja

Produkcijski release ponovno izvršava vlastiti quality job. To znači da javno izdanje ne ovisi o pretpostavci da je prethodni PR check bio zelen ili da se `main` može mijenjati samo kroz PR.

`publish` čeka:

1. production quality/race gate
2. Windows build
3. Linux build
4. macOS build

Tek tada smije generirati završne metapodatke i objaviti release.

## Build skripte i minimalni Go

Windows, Linux, macOS i lokalni produkcijski build zahtijevaju Go **1.26.5+**. Buildovi koriste lokalni toolchain i ne smiju automatski preuzimati Go module ili zamjenski toolchain.

## Verzija i dokumentacija

`audit_version.py` zahtijeva da svi buildovi čitaju isti `VERSION` i ugrađuju ga u runtime.

`audit_docs.py` provjerava:

- lokalne Markdown/HTML poveznice
- izlazak linka iz repozitorija
- indeksiranje svakog `docs/*.md` dokumenta
- zastarjele verzionirane naslove detaljnih dokumenata

`audit_release.py` dodatno sprječava da se glavni README, instalacijske upute ili generirane release bilješke vrate na zastarjeli javni paketni model.

## Pre-release test sa stvarnim serverom

Lokalni process smoke je deterministična regresija, ali ne zamjenjuje kontrolirani završni test protiv stvarnog FTP/FTPS/SFTP poslužitelja.

Prije šire javne distribucije preporučeni su:

- Windows x64 FTP/FTPS/SFTP test
- Windows x86 osnovni test gdje je dostupan sustav
- Linux FTP/FTPS i SFTP-key test
- macOS FTP/FTPS i SFTP-key test

Stvarne produkcijske vjerodajnice ne smiju završiti u repozitoriju, Actions secrets logovima, fixtureima ili issue/PR sadržaju.
