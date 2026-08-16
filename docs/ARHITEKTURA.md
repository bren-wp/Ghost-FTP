# ByFTP — arhitektura

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
- `internal/security` — validacija unosa/putanja, no-follow filesystem granice i Windows DPAPI zaštita
- `internal/platform` — Win32 dijalozi, Known Folder/System Directory API-ji, Registry, shortcut, replace i single-instance pomoćne funkcije
- `scripts` — reproducibilni build, audit, PE, bundle i release alati

Runtime nema vanjske Go dependencies.

## Granica vjerodajnica

Lozinka i zaporka privatnog ključa ne prolaze kroz generički JSON dispatcher. Spremljeni DPAPI blob ostaje zaštićen kroz profile/manager sloj i otključava se tek neposredno prije sistemskog curl/OpenSSH poziva.

Kod potvrde novog SFTP host ključa ByFTP može privremeno zadržati DPAPI-zaštićene credential blobove najviše do isteka trust prozora. Podaci se brišu nakon potvrde, greške, otkaza ili uspješnog spajanja; jedina grana koja ih namjerno zadržava jest povrat `RequiresTrust` dok korisnik odlučuje o ključu.

## Granica reda prijenosa

Transfer manager drži poslove u memoriji i emitira inkrementalne događaje. Desktop event batch primjenjuje preko mape `job ID -> indeks` kako veliki burst ne bi radio puni scan reda za svaki event.

`Events` vraća duboku snimku događaja. `Event.Job` i `Event.Jobs` ne dijele mutabilni backing state s internom event poviješću.

Završni rezultat protokolarnog adaptera autoritativan je za posao. Kasni cancel/disconnect nakon stvarno dovršenog ili preskočenog transfera ne smije naknadno promijeniti završni status u `cancelled`.

## Granica lokalnih prijenosa

Queued posao ponovno validira `LocalRoot` prije svakog pokušaja. Rekurzivni upload namjerno veže odabrani root uz roditeljsku granicu kako bi i kasna zamjena samog roota symlinkom/junctionom bila otkrivena.

Download se izvodi kroz kriptografski nasumičnu `.byftp-part-*` datoteku. Prije atomske aktivacije staging objekt mora proći `Lstat`, biti regularna datoteka i ne smije biti Windows reparse point. Ciljni replace put dodatno čuva no-follow/no-replace granice i rollback.

`RemoveTreeNoFollow` ne prati symlink/junction/reparse točke, ima depth/item limite i samostalno odbija filesystem root, uključujući Windows drive i UNC root.

## Granica lokalnog stanja

State/config čitanje provjerava regularnu datoteku prije otvaranja, stvarno otvoreni objekt i stabilnost identiteta, veličine i modification metapodataka tijekom čitanja. Oštećeni ili nepouzdani current zapis ne blokira startup: koristi se provjerena `.previous` generacija ili sigurne zadane vrijednosti.

ByFTP data/install/SFTP direktoriji stvaraju se kroz no-redirect provjere ispod kanonskih Windows putanja.

## Granica sesije

Remote operacije dobivaju context koji se otkazuje i kada pozivatelj prekine posao i kada se aktivna ByFTP veza disconnecta. Transfer batch rezervacija pamti generation i opaque identitet veze kako stale posao ne bi nakon reconnecta završio na drugom endpointu.

## Release granica

`VERSION` je jedini kanonski broj izdanja. Windows i lokalni build čitaju ga iz iste datoteke. CI provjerava brand resurse, hrvatski sadržaj, dokumentaciju, verziju, sigurnosne invarijante, privatnost, release ugovor, Python release regresije, Go unit/race/vet i puni Windows production build.

`RELEASE-NOTES.txt` nastaje iz odgovarajućeg `CHANGELOG.md` odjeljka. `BUILD-METADATA.txt` sadrži samo verziju, source commit/ref, Go toolchain, platformu i GitHub Actions identifikatore. Source ZIP nastaje iz `git archive HEAD`.

Windows bundle nastaje iz provjerenih binarija i kompletne Markdown dokumentacije. `BUNDLE-SHA256.txt` pokriva svaku payload datoteku, a `verify_bundle.py` ponovno čita konačni ZIP bez raspakiravanja na disk i provjerava putanje, duplikate, ugovoreni sadržaj i svaki SHA-256.

Objava GitHub Releasea je idempotentna i fail-closed: postojeći tag mora razriješiti na točan release commit, postojeći asset mora imati istu veličinu i SHA-256 digest, a rerun smije samo nadopuniti nedostajući potvrđeni asset. Neočekivani ili izmijenjeni postojeći asset zaustavlja objavu.