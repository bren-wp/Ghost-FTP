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
- `internal/profilebinding` — zajednički endpoint/account/private-key identity ugovor za profile
- `internal/itemlist` — zajedničko stabilno directory-first sortiranje velikih lokalnih i udaljenih popisa
- `internal/localfs` — lokalne file operacije i bounded enumeracija
- `internal/security` — validacija unosa/putanja, no-follow filesystem granice i Windows DPAPI zaštita
- `internal/platform` — Win32 dijalozi, Known Folder/System Directory API-ji, Registry, shortcut, replace i single-instance pomoćne funkcije
- `scripts` — reproducibilni build, audit, PE, bundle i release alati

Runtime nema vanjske Go dependencies.

## Granica profila i vjerodajnica

Lozinka i zaporka privatnog ključa ne prolaze kroz generički JSON dispatcher. Spremljeni DPAPI blob ostaje zaštićen kroz profile/manager sloj i otključava se tek neposredno prije sistemskog curl/OpenSSH poziva.

`internal/profilebinding` centralizira tri identitetske razine. Endpoint je `protokol + normalizirani host + port`; account dodaje točno korisničko ime; private-key identitet dodatno dodaje case-insensitive Windows putanju privatnog ključa. Remote, config i desktop sloj koriste isti ugovor kako se sigurnosna pravila ne bi razišla.

Spremljena lozinka automatski se nasljeđuje samo kada account identitet ostaje isti. Spremljeni passphrase dodatno zahtijeva isti privatni ključ. Ako korisnik privremeno promijeni host, port, korisnika ili ključ bez spremanja profila, stari DPAPI blob ne prelazi na novi identitet.

Spremanje izmijenjenog profila također je fail-closed. Password blob automatski se uklanja kada se promijeni account identitet, a passphrase blob kada se promijeni endpoint/korisnik/ključ ili kada privatnog ključa više nema. Novi identitet dobiva spremljenu tajnu samo ako je korisnik ponovno izričito upiše i odobri spremanje.

Prazno polje privatnog ključa je autoritativno: odabrani profil ne smije ponovno vratiti staru key putanju samo zato što je spremljena u profilu.

## Granica SFTP host-key pina

Host-key fingerprint pripada endpointu, ne login računu. Spremljeni pin koristi se samo kada se aktualni `protokol + host + port` podudara sa spremljenim profilom.

Obično uređivanje naziva, korisnika, lokalne/udaljene putanje ili privatnog ključa na istom endpointu čuva pin. Promjena hosta, porta ili protokola čisti stari pin i traži novu trust potvrdu. Privremeno promijenjeni endpoint može biti prihvaćen samo za svoju sesiju i ne smije svoj fingerprint upisati u originalni profil.

Kod potvrde novog SFTP host ključa ByFTP može privremeno zadržati DPAPI-zaštićene credential blobove najviše do isteka trust prozora. Podaci se brišu nakon potvrde, greške, otkaza ili uspješnog spajanja; jedina grana koja ih namjerno zadržava jest povrat `RequiresTrust` dok korisnik odlučuje o ključu.

Privatni SFTP ključ mora biti stvarna regularna lokalna datoteka. Symlink i Windows reparse-point objekti odbijaju se prije izrade session konfiguracije kako OpenSSH ne bi potrošio preusmjereni objekt koji ByFTP nije namjerno odabrao.

## Granica reda prijenosa

Transfer manager drži poslove u memoriji i emitira inkrementalne događaje. Desktop event batch primjenjuje preko mape `job ID -> indeks` kako veliki burst ne bi radio puni scan reda za svaki event.

`Events` vraća duboku snimku događaja. `Event.Job` i `Event.Jobs` ne dijele mutabilni backing state s internom event poviješću.

Završni rezultat protokolarnog adaptera autoritativan je za posao. Kasni cancel/disconnect nakon stvarno dovršenog ili preskočenog transfera ne smije naknadno promijeniti završni status u `cancelled`.

## Granica lokalnih prijenosa

Queued posao ponovno validira `LocalRoot` prije svakog pokušaja. Rekurzivni upload namjerno veže odabrani root uz roditeljsku granicu kako bi i kasna zamjena samog roota symlinkom/junctionom bila otkrivena.

Download se izvodi kroz kriptografski nasumičnu `.byftp-part-*` datoteku. Prije atomske aktivacije staging objekt mora proći `Lstat`, biti regularna datoteka i ne smije biti Windows reparse point. Ciljni replace put dodatno čuva no-follow/no-replace granice i rollback.

`RemoveTreeNoFollow` ne prati symlink/junction/reparse točke, ima depth/item limite i samostalno odbija filesystem root, uključujući Windows drive i UNC root.

## Granica udaljenog prikaza

FTP/FTPS preferira strojno čitljivi MLSD. Kada poslužitelj to ne podržava, tekstualni Unix/DOS listing prolazi ograničeni parser: veličine se pretvaraju provjerenim `int64` parserom, a ` -> ` se tretira kao symlink separator samo kod zapisa koji je stvarno označen kao symlink.

Lokalni i udaljeni prikaz koriste zajednički `internal/itemlist.Sort`. Case-fold ključ računa se jedanput po stavci, pa limit od 50.000 unosa ne stvara lowercase kopiju u svakoj usporedbi sortiranja.

## Granica lokalnog stanja

State/config čitanje provjerava regularnu datoteku prije otvaranja, stvarno otvoreni objekt i stabilnost identiteta, veličine i modification metapodataka tijekom čitanja. Oštećeni ili nepouzdani current zapis ne blokira startup: koristi se provjerena `.previous` generacija ili sigurne zadane vrijednosti.

ByFTP data/install/SFTP direktoriji stvaraju se kroz no-redirect provjere ispod kanonskih Windows putanja.

## Granica sesije

Remote operacije dobivaju context koji se otkazuje i kada pozivatelj prekine posao i kada se aktivna ByFTP veza disconnecta. Svaka uspješna `Operation` registracija drži active-operation referencu sve do idempotentnog `release()` poziva.

Disconnect pod write lockom najprije uklanja aktivnu sesiju iz managera pa novi pozivi više ne mogu dobiti adapter. Zatim otkazuje session context. Stvarna cleanup rutina čeka da postojeći pozivi vrate svoje reference i tek tada poziva `Session.Close()`, čime curl/OpenSSH adapter i SFTP privremene config/known_hosts datoteke ne mogu biti uklonjeni dok ih paralelni `List`, rename, chmod ili transfer još koristi.

Pozivatelj ne čeka taj cleanup neograničeno. `Disconnect(ctx)` koristi isti deadline koji postavlja UI ili shutdown. Ako deadline istekne, manager vraća kontrolu pozivatelju, ali ne ruši sigurnosnu granicu prisilnim `Close()` pozivom. Close-state ostaje živ dok zadnja operacija ne preda `release()`, nakon čega se stari adapter automatski zatvara.

Dok close-state postoji, novi `Connect` fail-closed vraća `ErrSessionClosing`. Ponovljeni `Disconnect` veže se uz isti close-state, pa isti adapter ne može biti zatvoren dvaput. Tek nakon završnog cleanup-a dopušten je novi session lifecycle.

Transfer batch rezervacija dodatno pamti generation i opaque identitet veze kako stale posao ne bi nakon reconnecta završio na drugom endpointu.

## Release granica

`VERSION` je jedini kanonski broj izdanja. Windows i lokalni build čitaju ga iz iste datoteke. CI provjerava brand resurse, hrvatski sadržaj, dokumentaciju, verziju, sigurnosne invarijante, privatnost, release ugovor, Python release regresije, Go unit/race/vet i puni Windows production build.

`RELEASE-NOTES.txt` nastaje iz odgovarajućeg `CHANGELOG.md` odjeljka. `BUILD-METADATA.txt` sadrži samo verziju, source commit/ref, Go toolchain, platformu i GitHub Actions identifikatore. Source ZIP nastaje iz `git archive HEAD`.

Windows bundle nastaje iz provjerenih binarija i kompletne Markdown dokumentacije. `BUNDLE-SHA256.txt` pokriva svaku payload datoteku, a `verify_bundle.py` ponovno čita konačni ZIP bez raspakiravanja na disk i provjerava putanje, duplikate, ugovoreni sadržaj i svaki SHA-256.

Objava GitHub Releasea je idempotentna i fail-closed: postojeći tag mora razriješiti na točan release commit, postojeći asset mora imati istu veličinu i SHA-256 digest, a rerun smije samo nadopuniti nedostajući potvrđeni asset. Neočekivani ili izmijenjeni postojeći asset zaustavlja objavu.
