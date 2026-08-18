# ByFTP — arhitektura

ByFTP je jedan Go kodni sustav sa **zajedničkim tipiziranim engineom**, platform-specific korisničkim sučeljima i odvojenim produkcijskim build/release slojem. Runtime nema browser UI, localhost HTTP server, cloud kontrolni kanal ni mrežni IPC između sučelja i enginea.

## Slojevi

- `cmd/byftp` — startup, verzija i strogo ograničen Windows AskPass način rada
- `internal/api` — tipizirani in-process `Engine`; javna granica prema UI sloju
- `internal/remote` — FTP/FTPS preko sistemskog curl-a i SFTP preko sistemskog OpenSSH-a
- `internal/transfer` — red prijenosa, retry/cancel, event stream, generation i worker lifecycle
- `internal/localfs` — lokalne datotečne operacije i bounded enumeracija
- `internal/config` — atomsko lokalno stanje; Windows profilne tajne koriste DPAPI
- `internal/profilebinding` — endpoint/account/private-key identitet profila
- `internal/security` — validacija, filesystem granice, DPAPI i procesno runtime spremište tajni
- `internal/platform` — OS-specifične putanje i Win32 integracija
- `internal/desktop` — Windows Win32 GUI ili Linux/macOS terminalni frontend
- `cmd/installer` / `cmd/uninstaller` — interni Windows Setup lifecycle
- `scripts` — Windows/Linux/macOS build, PE, bundle, audit i release alati
- `.github/workflows` — PR CI i odvojeni produkcijski release gateovi

Runtime nema vanjske Go module.

## Platform-specific frontend

### Windows

Windows koristi puni Win32 GUI s dvopanelnim lokalnim/udaljenim prikazom, profilima, transfer queueom, toolbarom i dijalozima. Vanjski `curl.exe`, `sftp.exe`, `ssh-keyscan.exe` i `ssh-keygen.exe` uzimaju se iz stvarnog Windows System32/OpenSSH direktorija.

### Linux i macOS

`internal/desktop/other.go` pokreće terminalni ByFTP klijent nad istim `api.Engine` objektom. `ls`, `cd`, udaljene operacije i `get`/`put` pozivaju iste engine/transfer granice kao Windows UI.

Linux/macOS frontend podržava FTP/FTPS lozinku i SFTP privatni ključ bez passphrasea. Nepodržani SFTP password/passphrase način odbija se prije mrežnog pokušaja.

## Tipizirani pozivni put

UI ne šalje generičke JSON naredbe prema runtimeu. Glavni put izgleda ovako:

```text
desktop -> api.Engine -> remote / transfer / localfs -> platform adapter
```

Time se argumenti i greške provjeravaju kroz Go tipove, a ne kroz runtime string dispatcher.

## Povezivanje

```text
desktop -> api.Engine.Connect -> remote.Manager.Connect -> adapter -> početni List probe
```

Veza se smatra uspostavljenom tek nakon što adapter uspješno dovrši autentikaciju i pročita početni udaljeni direktorij. UI ne prikazuje „Povezano” samo zato što je mrežni child proces pokrenut.

### SFTP tijek

1. validacija endpointa i vjerodajnica
2. `ssh-keyscan` i SHA-256 fingerprint
3. endpoint-scoped host-key pin/trust odluka
4. privatni `known_hosts` i session config
5. stvarna SFTP autentikacija
6. početni `List` probe
7. tek tada `ConnectResult{Connected:true}`

ByFTP ne koristi SFTP batch način koji bi prisilno uključio `BatchMode=yes` i mogao blokirati Windows AskPass. Naredbe se šalju preko stdin-a, uz eksplicitni `BatchMode=no` na Windows credential putu.

IPv6 URL-style uglate zagrade uklanjaju se prije OpenSSH `HostName` i `ssh-keyscan` unosa.

## Process-level connect dokaz

`internal/remote/process_connect_smoke_other_test.go` stvara kratkotrajne lokalne fake `curl` i `sftp` izvršne datoteke i pokreće ih kroz isti produkcijski `exec.CommandContext` put.

Time se zajedno provjeravaju:

- executable discovery preko `PATH`-a
- stdin/config transport prema child procesu
- aktivni runtime-secret token i njegovo čišćenje
- SFTP process argumenti i command stream
- MLSD/Unix listing parser
- adapter `Close()` cleanup

Testovi nemaju vanjski mrežni promet niti stvarne vjerodajnice. Izvršavaju se na Linux i macOS CI runnerima prije pakiranja.

## Vjerodajnice

### Windows

Spremljene profilne tajne i aktivni adapter blobovi koriste Windows DPAPI. AskPass helper dobiva zaštićeni blob samo u kontroliranom child-process okruženju i provjerava vlastiti executable, token i trusted OpenSSH parent.

AskPass je fail-closed: tajna se vraća samo ako prompt jasno traži `password` ili `passphrase`. MFA/OTP/security-key prompt ne dobiva spremljenu tajnu.

### Linux/macOS

FTP/FTPS lozinka aktivne sesije sprema se u procesnu mapu iza kriptografski nasumičnog tokena. Adapter drži token, ne plaintext. `run()` dobiva kratkotrajnu kopiju, briše je nakon izrade curl konfiguracije, a `Close()` uklanja procesnu vrijednost.

Terminalni frontend trenutačno ne sprema profilne vjerodajnice na disk.

## Profilni identitet

`internal/profilebinding` definira:

- endpoint = `protokol + normalizirani host + port`
- account = endpoint + korisničko ime
- private-key identitet = account + putanja privatnog ključa

Spremljena lozinka ne prelazi na drugi account, passphrase ne prelazi na drugi ključ, a SFTP fingerprint ne prelazi na drugi endpoint.

## Session lifecycle

`remote.Manager.Operation` registrira aktivnu operaciju i daje joj context koji se otkazuje kada pozivatelj odustane ili veza ulazi u disconnect. `release()` je idempotentan.

Disconnect:

1. uklanja aktivnu session referencu pa novi pozivi više ne ulaze
2. otkazuje session context
3. čeka postojeće operacije
4. zatvara adapter tek nakon zadnjeg `release()`
5. ako caller deadline istekne, cleanup nastavlja odvojeno bez zatvaranja adaptera ispod aktivne operacije
6. reconnect ostaje blokiran dok stara sesija nije stvarno zatvorena

## Transfer lifecycle

Transfer queue koristi connection generation i opaque connection identity. Stari posao ne može nakon reconnecta neprimjetno prijeći na drugi server ili account.

Download koristi kriptografski nasumični staging sibling, `Lstat`/regular-file/reparse provjeru i no-replace backup/rollback. Rekurzivni upload ponovno validira lokalni root prije izvršenja.

## Build arhitektura

Produkcijski buildovi koriste jedan kanonski `VERSION` i minimalni Go **1.26.5+**.

Standardna build politika je:

```text
GOTOOLCHAIN=local
GOPROXY=off
GOSUMDB=off
CGO_ENABLED=0
```

Prije produkcijskog testa/builda `go telemetry` mora biti `off`. GitHub Actions to stanje eksplicitno postavlja naredbom `go telemetry off`, a build skripte ponovno provjeravaju rezultat i odbijaju build u drugom načinu.

### Windows build

`BUILD-WINDOWS.ps1` gradi x64 i x86. Javni `dist/` izlazi su Setup i Portable; tehničke instalacijske komponente i verification dokazi odvojeni su u internu build lokaciju.

### Linux build

`scripts/BUILD-LINUX.sh` cross-builda amd64, arm64 i i386 binarije i pakira ih u DEB s deklariranim sistemskim ovisnostima.

### macOS build

`scripts/BUILD-MACOS.sh` gradi amd64 i arm64, spaja ih `lipo` alatom, izrađuje `.icns`, `ByFTP.app` i Universal PKG.

## PR CI arhitektura

Prije mergea postoje četiri neovisna gatea:

1. quality: auditi + Python regresije + Go unit/race/vet
2. Windows: x64 + x86 produkcijski build
3. Linux: test/vet na Linux runneru + amd64/arm64/i386 DEB
4. macOS: test/vet na macOS runneru + Universal Intel/Apple Silicon PKG

## Produkcijska release arhitektura

2.16.2 odvaja release validaciju od pretpostavke da je PR CI već bio zelen.

Automatski release okidač je samo promjena `VERSION` na `main`; dostupan je i manualni rerun. Publisher-created tag ne pokreće drugi workflow. Svi release runovi dijele jednu `byftp-release` concurrency grupu.

Release ponovno izvršava četiri gatea:

1. **quality** — auditi, Python regresije, unit, race, vet i release-note provjera
2. **Windows** — x64/x86 build + ZIP bundle verifikacija
3. **Linux** — Linux test/vet + tri DEB paketa i metadata provjera
4. **macOS** — macOS test/vet + Universal PKG i strukturalna provjera

`publish` ovisi o sva četiri joba.

Prije generiranja zajedničkih metapodataka `release/` staging mora sadržavati točno 10 očekivanih platformskih paketa. Zatim se dodaju `SHA256.txt`, `RELEASE-NOTES.txt` i `BUILD-METADATA.txt`, čime nastaje završni ugovor od 13 custom asseta.

`scripts/publish_release.ps1` jedini smije stvarati/nadopunjavati GitHub Release. Tag mora pokazivati na release commit, postojeći asset mora odgovarati po veličini i SHA-256 digestu, a neočekivani sadržaj zaustavlja izdanje.

## GitHub Package

`ByFTP.Windows` je dodatni Windows-only distribucijski paket. On nije runtime ovisnost ByFTP-a i ne utječe na Linux/macOS distribuciju.

## Digitalni identitet

Build/release arhitektura ne fabricira potpis:

- Windows Verified Publisher zahtijeva stvarni Brendigo Authenticode certifikat
- macOS Developer ID/notarizacija zahtijeva stvarni Apple identitet i secrets

Bez toga release ostaje tehnički testiran i SHA-verificiran, ali nepotpisan u smislu tih platformskih publisher identiteta.
