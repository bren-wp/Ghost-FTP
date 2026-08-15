# Changelog

## 2.12.1 — Recursive-upload root hardening & complete release packaging

- recursive upload jobs now bind their local safety boundary to the parent of the selected upload root so the root itself is revalidated before every queued attempt
- late replacement of an already-planned upload root with a symlink, junction or other reparse redirect is rejected before transfer execution
- added regression coverage reproducing the late-root redirect scenario
- introduced a canonical `VERSION` file used by production builds and release automation
- GitHub release automation now produces standalone Setup, Portable and Uninstaller EXEs plus a complete Windows x64 ZIP and exact tracked Source ZIP
- complete Windows ZIP includes release notes, privacy/security/license documentation, verification output and bundle-local SHA-256 checksums
- release workflow regenerates global SHA-256 checksums across EXEs, ZIPs and verification assets and uploads a 30-day complete Actions artifact
- version bumps on `main` now trigger the verified release pipeline automatically while tag/manual releases remain supported

## 2.12.0 — Safer retry, skip-existing, connection timeout & tree-transfer hardening

- added Skip Existing transfer policy for safe non-overwriting upload/download workflows
- added configurable 5–60 second connection timeout propagated to Windows curl and OpenSSH
- automatic retry now repeats only classified transient network/timeout/partial-transfer failures
- queued transfers revalidate their allowed local root immediately before every attempt to block late symlink/junction redirection
- ByFTP-owned data/install/SFTP session directories use secure segment-by-segment no-redirect creation
- increasing transfer parallelism applies immediately to the live queue
- settings cache no longer caches a failed first load; legacy zero retry delay normalizes to the production default
- recursive upload caches already-confirmed remote directories to reduce repeated LIST traffic on large trees
- Clear Finished drops old job/event backing storage to reduce retained path/error metadata in process memory
- added regression coverage for timeout propagation, transient-only retry, skip-existing, LocalRoot runtime validation, directory redirect protection and event metadata purge

## 2.11.0 — Typed engine, bulk workflows, retry safety & hot-path optimization

- removed the obsolete in-process JSON command dispatcher and moved the entire native desktop/engine boundary to typed Go methods
- cached settings in memory after first validated load so transfer scheduling no longer rereads settings storage in its hot path
- added configurable 1–8 transfer parallelism and optional automatic retry (0–3 attempts with bounded delay)
- added multi-select batch upload/download for local/remote files and directories while retaining all path, symlink and reparse protections
- added multi-select queue Cancel/Retry and atomic batch validation
- bound every queued transfer to an opaque active-connection identity so a failed/cancelled transfer cannot be retried against a different server/account after reconnect
- added bulk remote permissions (CHMOD) and adaptive bounded timeouts for larger remote mutation selections
- cached FTP MLSD capability per active session so legacy servers fall back to LIST only once instead of on every directory refresh
- actively cancel superseded local/remote directory refreshes and made large local enumeration context-aware between chunks
- removed unnecessary per-second transfer-event goroutine/dispatch churn; the native UI now reads the in-memory event stream directly
- launch Windows curl/OpenSSH child processes hidden and bound process wait cleanup after cancellation
- remove the current working directory from the Windows DLL search path
- clear active protocol-adapter references to protected credential/session metadata on Close
- fixed remote mutation and local action path races by snapshotting the target directory before background execution
- expanded privacy audit and regression tests for typed dispatch removal, batch retry/cancel, auto-retry, MLSD classification and cross-server retry prevention

## 2.10.0 — Worker containment, junction safety & transactional file hardening

- isolated unexpected transfer-worker panics so one malformed transfer cannot terminate the whole desktop process or permanently consume a worker slot
- introduced an internal transfer operation-provider boundary and regression tests proving a failed/panicking job releases its slot and later jobs continue
- wrapped asynchronous desktop work and UI dispatch callbacks with product-safe panic containment instead of exposing development crashes to the user
- expanded native-control startup validation so a partially created Win32 UI fails cleanly instead of running with missing controls
- replaced predictable transfer staging/rollback tokens with cryptographically random names
- added no-replace local move/rename activation to close check-then-rename overwrite races during local rename and atomic download replacement
- reject Windows junction/reparse-point traversal during direct upload, recursive upload, local listing and local download containment checks
- reject redirected ByFTP-owned data/install descendants under LocalAppData to reduce silent redirection of local state into another filesystem location
- strengthened installer upgrade backup checks against reparse-point installation targets
- disabled OpenSSH global known-host sources, SSHFP/DNS host-key verification and automatic host-key updates so SFTP trust remains limited to the exact ByFTP-approved host key
- fixed local file actions to capture their target directory before background execution, preventing fast navigation from redirecting an already-started mkdir/rename/delete operation
- removed duplicate Win32 click/theme handling left from earlier UI revisions
- expanded automated privacy/release checks for random staging names, no-replace local activation, worker panic containment, reparse-point protection and redirected-data-path blocking

## 2.9.0 — SFTP metadata privacy, path consistency & reconnect isolation

- removed SFTP AskPass credential temp files; encrypted DPAPI blobs now travel only in the short-lived child-process environment
- AskPass helper now requires a valid token, the ByFTP executable and a trusted System32 OpenSSH parent process
- moved user-entered SFTP host/username/private-key path out of the OpenSSH command line into a random short-lived session config and random host alias
- `ssh-keyscan` receives the target host via stdin instead of command-line or disk input
- host-key trust now writes only the selected key and pins the SFTP session to the exact host-key algorithm whose SHA-256 fingerprint was displayed
- fixed remote double-click download bypass so every server-controlled local filename uses `SafeLocalChild` and Windows reserved-name/traversal checks
- split remote-name validation from Windows-local-name validation so legal Unix server names remain usable without weakening local download safety
- FTP directory listing now prefers RFC 3659 MLSD with traditional LIST fallback
- local recursive deletion and optional uninstall data cleanup use no-follow traversal and never recurse through symlink/reparse/junction targets
- tree-download preparation rolls back only empty directories created by that attempt when queue commit fails
- transfer batch reservations carry a connection generation and cannot commit into a later reconnected session
- added Win32 input limits for host, port, username, password, passphrase, key path and local/remote path controls plus a defensive text-read allocation cap
- expanded automated privacy audit for AskPass disklessness, SFTP command-line metadata minimization, host-key algorithm binding, no-follow deletion and bounded sensitive UI inputs

## 2.8.0 — Privacy-by-default, no persistent diagnostics & encrypted profiles

- removed persistent runtime activity/error logging and added cleanup of legacy `byftp.log` files
- sensitive Connect and Save Profile operations now use typed in-process Engine methods instead of generic JSON serialization
- generic dispatcher rejects sensitive channels before payload marshalling
- Windows startup sets `SEM_NOGPFAULTERRORBOX` so Windows Error Reporting is not invoked for ByFTP/child process crashes
- saved profile metadata is now wrapped in a Windows DPAPI-protected envelope; legacy plaintext profile containers migrate automatically
- saved credential blobs now pass from the profile store to protocol adapters without early decryption in the connection manager
- one-time SFTP host-key trust keeps unsaved direct credentials only as a short-lived DPAPI blob bound to the exact host/port/user/key/fingerprint, then discards it
- profile UI asks before persisting newly entered password/private-key passphrase values locally
- AskPass decrypts into mutable byte buffers, writes directly to stdout and wipes buffers immediately afterward
- curl credential config is assembled in a mutable byte buffer, delivered via stdin and wiped after each command
- SFTP known-host material is session-temporary and removed on disconnect; fingerprint temp files stay under the private ByFTP data directory
- private-key file picker uses `OFN_DONTADDTORECENT` to avoid adding selected key paths to Windows Recent items
- password-only SFTP explicitly disables implicit/default SSH identity files and inherited security-key providers
- password/passphrase edit controls are cleared after successful connect/profile save
- desktop, installer and uninstaller catch unexpected Go panics and show product-safe messages instead of development stack output
- uninstaller can optionally remove the local ByFTP profile/settings directory while preserving it by default
- removed now-dead transfer error reporter hook left from persistent diagnostics
- privacy audit now verifies WER suppression, no persistent logging, DPAPI profile envelope and no sensitive JSON dispatcher paths
- production minimum corrected to the currently published Go 1.26.5 security patch release

## 2.7.0 — Windows lifecycle, installer integrity & session hardening

- Windows data and install locations resolve through Known Folder APIs instead of trusting `LOCALAPPDATA`/`WINDIR` environment variables
- Windows curl/OpenSSH discovery uses the real System32 directory from `GetSystemDirectoryW`; Windows still has no arbitrary PATH fallback
- fixed single-instance mutex handling to use the error returned by the actual `CreateMutexW` call
- fixed initial per-monitor-DPI layout sizing and added explicit message-loop error handling
- active remote operations now receive a session-scoped cancellation context; Disconnect cancels in-flight list/rename/delete/chmod/tree operations as well as transfers
- SFTP commands use controlled `-oBatchMode=no -b -` ordering so AskPass remains available while failed batch commands return reliable process failure
- active FTP/SFTP session credentials are DPAPI-protected instead of being retained as plaintext session fields; temporary DPAPI byte buffers are explicitly wiped
- local diagnostic log rejects symlink/non-regular targets and performs checked rotation
- installer location is canonicalized through the Windows Known Folder API and existing executable symlinks are rejected during upgrade backup
- installer payload parser now requires exactly `ByFTP.exe`, `Uninstall.exe` and a validated manifest; manifest size/SHA-256 must match both embedded executables
- installer captures/restores relevant Registry values during rollback so binary and Windows integration versions cannot diverge after a failed upgrade
- copied/moved `Uninstall.exe` refuses to delete arbitrary directories; uninstall is allowed only from the canonical ByFTP install path
- uninstaller now checks delete/schedule/shortcut/Registry cleanup results instead of always reporting success
- release verification now validates Portable, Setup and Uninstaller as three distinct hardened PE binaries
- added regression tests for payload duplicates/tampering, symlink upgrade targets, canonical uninstall path handling and disconnect-driven operation cancellation

## 2.6.0 — Production privacy, safer downloads & transactional hardening

- removed remaining dev/diagnostic UI surfaces and technical user-facing wording
- strict no-telemetry/no-external-API release policy and automated privacy audit
- offline standard-library-only production build; minimum Go 1.26.5
- System32-only curl/OpenSSH selection on Windows; proxy, TLS-keylog, external CA/backend, SSH agent and AskPass inheritance blocked
- ByFTP-managed SSH config blocks ProxyJump/ProxyCommand/agent/forwarding/local-command hooks
- FTPS external Schannel revocation fetch disabled while certificate/hostname verification remains enabled
- AskPass invocation restricted to own executable + managed private file
- Windows reserved filename/traversal protection and nested symlink/junction escape defense for downloads
- random/atomic config, known-host, SSH-config and installer staging files
- installer rollback failures are no longer silently ignored
- removed dead engine channels/capability/status surfaces left from older architecture

## 2.5.0 — Fluent UI, deep reliability audit & resource hardening

- glavni desktop gumbi koriste centralizirane Windows Fluent/MDL2 glyphove umjesto Unicode pseudo-ikona; Windows 11 koristi Segoe Fluent Icons, Windows 10 Segoe MDL2 fallback
- uvedeni owner-draw dark gumbi s accent/danger/subtle varijantama, jasnijim fokusom i konzistentnim ByFTP vizualnim jezikom
- stvarno per-monitor DPI skaliranje: prozor, layout, fontovi, Fluent ikone i list-view stupci prate DPI monitora i WM_DPICHANGED
- sistemske file/folder ikone se cacheiraju po tipu/ekstenziji radi manjeg Shell/GDI overheada u velikim mapama
- async lokalna/remote navigacija koristi generation tokene pa zastarjeli odgovor više ne može prepisati noviju putanju/listing
- Connect/Disconnect dobili su eksplicitno busy stanje; tijekom promjene veze remote i transfer akcije se zaključavaju kako ne bi nastala utrka s CancelAll/Disconnect
- health-check nakon remote greške je dedupliciran i cancellable; paralelne greške više ne pokreću više istodobnih probe/disconnect ciklusa
- SFTP više ne izrađuje privremenu batch datoteku za svaku naredbu; naredbe se validirano šalju izravno kroz stdin uz postojeći DPAPI AskPass kanal
- AskPass sada propagira read/DPAPI/stdout greške i vraća neuspješan exit kod umjesto tihog slanja prazne lozinke
- FTP/SFTP stdout/stderr su ograničeni; nenormalno velik odgovor poslužitelja više ne može nekontrolirano puniti memoriju procesa
- transfer queue ima globalni limit i skraćuje/sanitizira ekstremno dugačke server error poruke prije spremanja/prikaza
- tree transfer koristi rezervaciju queue kapaciteta prije stvaranja direktorija; paralelna akcija više ne može naknadno učiniti već preflightani batch prevelikim
- rekurzivni remote delete za FTP i SFTP koristi isti zajednički sigurni helper umjesto duplicirane implementacije
- lokalni rename više ne prepisuje postojeću ciljnu stavku; delete prijavljuje nestalu stavku i symlink se briše kao link, ne kao cilj
- installer sada prijavljuje grešku ako je ByFTP uspješno instaliran, ali ga Windows nakon instalacije ne može pokrenuti
- dodani testovi za SFTP command stream/injection, bounded output, batch reservations, atomic queue, symlink delete, overwrite protection i postojeće connection/transfer recovery scenarije

## 2.4.0 — Reliability & SFTP hardening

- SFTP se nakon povezivanja otvara u korisnikovom home direktoriju (`.`), a ne u SSH rootu `/`
- ispravljena navigacija "mapa gore" za relativne SFTP putanje; iz `public_html` povratak sada vodi u home, ne u root
- spremljeni SFTP profil bez zadane remote putanje automatski koristi home (`.`); FTP/FTPS profil koristi `/`
- dodan stvarni connection probe nakon remote greške kako ONLINE status ne bi ostao aktivan nakon prekida mreže
- eksplicitni Disconnect sada kontrolirano otkazuje queued/running transfere prije zatvaranja sessiona
- rekurzivni prijenos mape sada radi preflight cijelog stabla i atomski dodaje batch u queue; kasna greška više ne ostavlja djelomičan queue
- dodan `CancelAll`, `ActiveCount` i pouzdaniji transfer lifecycle
- transfer UI koristi inkrementalne evente umjesto ponovnog crtanja cijelog queuea svake sekunde; zastarjeli cursor dobiva puni snapshot
- ispravljen `completed`/`done` lifecycle mismatch
- SFTP fingerprint skeniranje sada ima zasebne timeoutove za `ssh-keyscan` i `ssh-keygen`
- poboljšan SFTP AskPass lifecycle: provjera write/close grešaka, čišćenje starih crash ostataka i `IdentitiesOnly` za eksplicitni privatni ključ
- dodani OpenSSH connect/keepalive parametri i ograničen broj password promptova
- dodani guardovi dubine/broja stavki za rekurzivni remote delete i tree transfer
- `profiles:save` je tipiziran preko `ProfileInput`; nepoznata polja payload-a se odbijaju
- uklonjene mrtve dark/light i activity-log postavke: ByFTP je namjerno dark-only
- uklonjeni Speed/ETA stupci koji nisu imali stvarno mjerenje u protocol adapterima
- ispravljena nepotrebna delete logika vezana uz `Parallelism == 0`
- lokalni file list više ne skriva stavku ako Windows privremeno ne može dohvatiti detaljni `FileInfo`
- provjerava se greška `os.Executable()` u aplikaciji i uninstalleru
- očišćeni Windows `unsafe.Pointer` vet warningi u shortcut i notification kodu
- production build skripte ponovno koriste puni `go vet ./...` bez skrivanja `unsafeptr` provjere
- dodani testovi za atomski transfer batch, CancelAll, stale event snapshot, strict payload, SFTP home putanje, relativne remote direktorije i legacy settings migraciju

## 2.3.0 — Branding & desktop polish

- centraliziran ByFTP/Brendigo branding
- uvedena ByFTP aplikacijska ikona i Windows file/folder sistemske ikone
- poboljšan tamni header, status veze, toolbar i About ekran
- installer dobio rollback prethodne instalacije kod neuspjelog upgradea
- desktop kod dodatno razdvojen na commands, protocols, theme, settings, connections, files i transfers module

## 2.2.0 — Dark Desktop & File Manager

- uveden tamni Native Win32 dizajn i dark title bar
- `api.Server` preimenovan u `api.Engine`; localhost/server terminologija uklonjena iz runtime arhitekture
- dodani spremljeni profili i sigurno spremanje vjerodajnica
- dodana SFTP autentikacija privatnim ključem i passphraseom iz GUI-ja
- dodan implicit FTPS
- dodane lokalne i remote file operacije te CHMOD
- dodan rekurzivni upload/download mapa
- dodane queue kontrole pause/resume/cancel/retry/clear finished
- dodan engine panic recovery i dijagnostički log

## 2.1.0 — Native Desktop

- browser/localhost desktop launcher zamijenjen nativnim Win32 prozorom
- ugrađen lokalni/remote file panel i transfer queue

## 2.0.0

- hardened FTP/FTPS/SFTP engine, installer i portable distribucija
