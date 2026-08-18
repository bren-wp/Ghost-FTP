# ByFTP — arhitektura

ByFTP je jedan Go kodni sustav s **zajedničkim tipiziranim engineom** i platform-specific korisničkim sučeljima. Nema browser UI, localhost HTTP server, cloud kontrolni kanal ni mrežni IPC između sučelja i enginea.

## Slojevi

- `cmd/byftp` — startup, verzija i strogo ograničen Windows AskPass način rada
- `internal/api` — tipizirani in-process Engine; jedina javna granica prema UI sloju
- `internal/remote` — FTP/FTPS preko curl i SFTP preko OpenSSH
- `internal/transfer` — red prijenosa, retry/cancel, event stream, generation i worker lifecycle
- `internal/localfs` — lokalne file operacije i bounded enumeracija
- `internal/config` — atomsko lokalno stanje; Windows profilne tajne koriste DPAPI
- `internal/profilebinding` — endpoint/account/private-key identitet profila
- `internal/security` — validacija, filesystem granice, DPAPI i procesno runtime spremište tajni
- `internal/platform` — OS-specifične putanje i Win32 integracija
- `internal/desktop` — Windows Win32 GUI ili Linux/macOS terminalni frontend
- `cmd/installer` / `cmd/uninstaller` — Windows Setup lifecycle
- `scripts` — Windows/Linux/macOS build, PE, bundle, audit i release alati

Runtime nema vanjske Go module.

## Platform-specific frontend

### Windows

Windows koristi puni Win32 GUI s dvopanelnim local/remote prikazom, profilima, transfer queueom, toolbarom i dijalozima. Vanjski `curl.exe`, `sftp.exe`, `ssh-keyscan.exe` i `ssh-keygen.exe` uzimaju se iz stvarnog Windows System32/OpenSSH direktorija.

### Linux i macOS

Ne-Windows `internal/desktop/other.go` pokreće terminalni ByFTP klijent nad istim `api.Engine` objektom. `ls`, `cd`, remote operacije i `get`/`put` pozivaju iste metode i transfer queue kao Windows UI.

Linux/macOS frontend podržava FTP/FTPS lozinku i SFTP privatni ključ bez passphrasea. Nepodržani SFTP password/passphrase način odbija se prije mrežnog pokušaja. To je namjerno fail-closed ograničenje dok se ne uvede Unix AskPass broker.

## Povezivanje

`desktop -> api.Engine.Connect -> remote.Manager.Connect -> adapter -> probe`

Veza se smatra uspostavljenom tek nakon što adapter uspješno napravi početni remote `List` nad `/` ili `.`. UI zato ne prikazuje „Povezano” samo zato što je mrežni proces pokrenut.

### SFTP

1. validacija endpointa i vjerodajnica
2. `ssh-keyscan` i SHA-256 fingerprint
3. endpoint-scoped host-key pin/trust odluka
4. privatni `known_hosts` + privatni session config
5. stvarni `sftp` authentication
6. početni `List` probe
7. tek tada `ConnectResult{Connected:true}`

ByFTP **ne koristi `sftp -b`**. Aktualni OpenSSH `-b` uključuje `BatchMode=yes`, što je nespojivo s password/passphrase AskPass tokom. Naredbe se i dalje šalju preko stdin-a, ali uz eksplicitni `BatchMode=no`.

IPv6 URL-style uglate zagrade uklanjaju se prije OpenSSH `HostName` i `ssh-keyscan` unosa.

## Process-level connect dokaz

Od 2.16.1 adapter arhitektura ima dodatni testni sloj između čistih unit testova i stvarnog mrežnog poslužitelja. `internal/remote/process_connect_smoke_other_test.go` stvara lokalne kratkotrajne fake `curl` i `sftp` executable datoteke te ih pokreće kroz isti production `exec.CommandContext` put.

Time se zajedno provjeravaju:

- executable discovery preko `PATH`-a
- stdin/config transport prema child procesu
- aktivni runtime-secret token i njegovo čišćenje
- zabrana OpenSSH `-b`
- SFTP command stream
- MLSD/Unix listing parser
- adapter `Close()` cleanup

Testovi nemaju vanjski mrežni promet niti stvarne vjerodajnice. Izvršavaju se na Linux i macOS CI runnerima prije pakiranja.

## Vjerodajnice

### Windows

Spremljene profilne tajne i aktivni adapter blobovi koriste Windows DPAPI. AskPass helper dobiva DPAPI blob samo u kontroliranom child-process okruženju i provjerava vlastiti executable, token i trusted OpenSSH parent.

AskPass je fail-closed: tajna se vraća samo ako prompt jasno sadrži `password` ili `passphrase`. MFA/OTP/security-key prompt ne dobiva spremljenu tajnu.

### Linux/macOS

FTP/FTPS lozinka aktivne sesije sprema se u procesnu mapu iza kriptografski nasumičnog tokena. Adapter drži token, ne plaintext. `run()` dobiva kratkotrajnu kopiju, briše je nakon izrade curl konfiguracije, a `Close()` uklanja i briše procesnu vrijednost.

ByFTP ne sprema terminalne profile ni terminalne vjerodajnice na disk.

## Profilni identitet

`internal/profilebinding` definira:

- endpoint = `protokol + normalizirani host + port`
- account = endpoint + korisničko ime
- private-key identitet = account + putanja privatnog ključa

Spremljena lozinka ne prelazi na drugi account, passphrase ne prelazi na drugi ključ, a SFTP fingerprint ne prelazi na drugi endpoint.

## Session lifecycle

`remote.Manager.Operation` registrira aktivnu operaciju i daje joj context koji se otkazuje kada pozivatelj odustane ili kada veza ide u disconnect. `release()` je idempotentan.

Disconnect:

1. uklanja aktivnu session referencu pa novi pozivi više ne ulaze
2. otkazuje session context
3. čeka postojeće operacije
4. zatvara adapter tek nakon zadnjeg `release()`
5. ako caller deadline istekne, cleanup se nastavlja odvojeno bez zatvaranja adaptera ispod aktivne operacije
6. reconnect ostaje blokiran dok stara sesija nije stvarno zatvorena

## Transfer lifecycle

Transfer queue koristi connection generation i opaque connection identity. Stari posao ne može nakon reconnecta neprimjetno prijeći na drugi server/account.

Download koristi kriptografski nasumični staging sibling, `Lstat`/regular-file/reparse provjeru te no-replace backup/rollback. Rekurzivni upload ponovno validira lokalni root prije izvršenja.

## Release arhitektura

`VERSION` je jedini kanonski broj izdanja.

CI prije mergea ima četiri gatea:

1. quality: auditi + Python regresije + Go unit/race/vet
2. Windows: x64 + x86 production build
3. Linux: Go test/vet na Linux runneru + amd64 + arm64 + i386 DEB build
4. macOS: Go test/vet na macOS runneru + Universal Intel+Apple Silicon PKG build

Release workflow ponovno gradi sve platforme, preuzima njihove Actions artefakte u završni publish job, generira `SHA256.txt`, `RELEASE-NOTES.txt` i `BUILD-METADATA.txt` te koristi centralni `publish_release.ps1`.

Javni release ne uključuje standalone Uninstaller, interni verification report ni custom Source ZIP. Windows ZIP sadrži Setup, Portable i dokumentaciju, a `verify_bundle.py` provjerava manifest i SHA-256 nakon kompresije.

GitHub automatski Source code ZIP/TAR nije ByFTP build artefakt i postoji za svaki tag neovisno o workflowu.
