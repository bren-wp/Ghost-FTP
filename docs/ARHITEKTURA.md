# ByFTP 2.14.0 — arhitektura

ByFTP je jedan izvorni Win32 desktop proces. Nema browser UI, localhost HTTP server ni mrežni IPC između korisničkog sučelja i enginea.

## Moduli

- `cmd/byftp` — startup i strogo ograničen SFTP AskPass način rada
- `cmd/installer` — per-user Windows instalacija s integritetom payloada i rollbackom
- `cmd/uninstaller` — kontrolirano uklanjanje iz kanonske instalacijske putanje
- `internal/desktop` — izvorni Win32 dark/Fluent UI i transfer-event prikaz
- `internal/api` — tipizirani in-process engine i planiranje tree transfera
- `internal/remote` — FTP/FTPS preko Windows curl i SFTP preko Windows OpenSSH
- `internal/transfer` — red prijenosa, rezervacije, retry, cancellation, event stream i worker lifecycle
- `internal/config` — atomsko lokalno stanje, DPAPI profili i settings cache
- `internal/itemlist` — stabilno directory-first sortiranje velikih popisa
- `internal/localfs` — lokalne file operacije i bounded enumeracija
- `internal/security` — validacija unosa/putanja i Windows DPAPI zaštita
- `internal/platform` — Win32 dijalozi, Known Folder/System Directory API-ji, Registry, shortcut, replace i single-instance pomoćne funkcije

Runtime nema vanjske Go dependencies.

## Granica vjerodajnica

Lozinka i zaporka privatnog ključa ne prolaze kroz generički JSON dispatcher. Spremljeni DPAPI blob ostaje zaštićen kroz profile/manager sloj i otključava se tek neposredno prije sistemskog curl/OpenSSH poziva.

Kod potvrde novog SFTP host ključa ByFTP može privremeno zadržati DPAPI-zaštićene credential blobove najviše do isteka trust prozora. Od 2.14 ti se podaci automatski brišu nakon potvrde, greške, otkaza ili uspješnog spajanja; jedina grana koja ih namjerno zadržava jest povrat `RequiresTrust` dok korisnik odlučuje o ključu.

## Granica reda prijenosa

Transfer manager drži poslove u memoriji i emitira inkrementalne događaje. Desktop event batch primjenjuje preko mape `job ID -> indeks` kako veliki burst ne bi radio puni scan reda za svaki event.

Od 2.14 `Events` vraća duboku snimku događaja. `Event.Job` i `Event.Jobs` više ne dijele mutabilni backing state s internom event poviješću, pa UI ili budući API potrošač ne može nenamjerno promijeniti buduće odgovore mutacijom već vraćene vrijednosti.

## Granica lokalnog stanja

State/config čitanje provjerava regularnu datoteku prije otvaranja, stvarno otvoreni objekt i stabilnost veličine/modification metapodataka tijekom čitanja. Oštećeni ili nepouzdani current zapis ne blokira startup: koristi se provjerena `.previous` generacija ili sigurne zadane vrijednosti.

ByFTP data/install/SFTP direktoriji stvaraju se kroz no-redirect provjere ispod kanonskih Windows putanja.

## Granica sesije

Remote operacije dobivaju context koji se otkazuje i kada pozivatelj prekine posao i kada se aktivna ByFTP veza disconnecta. Transfer batch rezervacija pamti generation i opaque identitet veze kako stale posao ne bi nakon reconnecta završio na drugom endpointu.

## Release granica

`VERSION` je jedini kanonski broj izdanja. Windows i lokalni build čitaju ga iz iste datoteke. CI provjerava:

1. slikovne resurse,
2. hrvatski korisnički sadržaj,
3. privacy/network politiku,
4. unit/race/vet provjere,
5. release bilješke za točan `VERSION`,
6. puni Windows production build.

`RELEASE-NOTES.txt` nastaje iz odgovarajućeg `CHANGELOG.md` odjeljka. `BUILD-METADATA.txt` sadrži samo verziju, source commit/ref, Go toolchain, platformu i GitHub Actions identifikatore. Source ZIP nastaje iz `git archive HEAD` i ne sadrži lokalni build output.
