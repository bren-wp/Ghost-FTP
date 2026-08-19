# ByFTP — sigurnost

## Cilj

ByFTP je klijent kojem korisnik izričito daje endpoint i vjerodajnicu. Sigurnosni model prioritet daje pravilnom vezivanju tajni uz endpoint, strogoj SFTP host-key provjeri, no-follow datotečnim operacijama, izolaciji transfera i fail-closed produkcijskom release procesu.

## Pouzdano povezivanje

Veza nije „uspješna” kada je samo pokrenut curl/OpenSSH proces. `remote.Manager.Connect` mora dovršiti autentikaciju i početni udaljeni `List` probe. Tek tada vraća `Connected=true`.

### OpenSSH BatchMode granica

ByFTP ne koristi SFTP batch način koji bi prisilno uključio `BatchMode=yes` i time mogao onemogućiti password/passphrase AskPass.

Sigurnosna invarijanta zahtijeva:

- `-oBatchMode=no` na Windows AskPass putu
- naredbe preko stdin-a
- regresiju koja pada čim se vrati konfliktni batch način

### Process-level connect smoke

Ne-Windows testni sloj stvarno pokreće lokalne child procese kroz isti `exec.CommandContext` put kao produkcijski adapter.

FTP/FTPS smoke potvrđuje:

- runtime-secret token
- curl config preko stdin-a
- MLSD odgovor i parser
- wipe-on-close

SFTP smoke potvrđuje:

- sigurne process argumente
- SFTP naredbu preko stdin-a
- produkcijski listing parser

Testni procesi ne kontaktiraju vanjsku mrežu i ne koriste stvarne vjerodajnice. Linux i macOS CI izvršavaju te regresije na svojim runnerima prije pakiranja.

### Timeout i otkazivanje

`curl`, `sftp`, `ssh-keyscan` i `ssh-keygen` vraćaju stvarni `context.Canceled` ili `context.DeadlineExceeded` kada ih context prekine. UI ne mora zaključivati uzrok samo iz teksta vanjskog alata.

### IPv6

Bracketirani IPv6 unos prihvaća se kao korisnički host, ali samo kada postoji točno jedan ispravno uparen par `[]`. Nepotpune ili višestruke zagrade i bracketirani IPv4 odbijaju se prije mrežnog pokušaja. Ispravne IPv6 zagrade uklanjaju se prije OpenSSH `HostName` i `ssh-keyscan` ulaza.

## Vjerodajnice

### Windows

- profilne lozinke/passphrase koriste Windows DPAPI
- aktivni Windows adapteri koriste DPAPI runtime blob
- plaintext se ne stavlja u command line
- AskPass helper provjerava vlastiti executable, jednokratni token i očekivani System32 OpenSSH parent
- AskPass daje password samo `password` promptu, passphrase samo `passphrase` promptu
- MFA, OTP, security-key i nepoznati promptovi ne dobivaju spremljenu tajnu
- UI ne briše upravo unesenu tajnu prije rezultata spajanja; sadržaj se briše nakon potvrđenog `Connected` stanja

### Linux/macOS

FTP/FTPS aktivna lozinka ne sprema se u profil ili datoteku. `ProtectRuntimeString` stvara kriptografski nasumični token i čuva vrijednost samo u procesu. Adapter drži token, `run()` dobiva kratkotrajnu kopiju i briše je, a `Close()` uklanja i briše procesnu vrijednost.

SFTP na Linuxu/macOS-u trenutačno je namjerno ograničen na eksplicitni privatni ključ bez passphrasea. Password/passphrase SFTP odbija se prije mrežnog pokušaja dok Unix credential broker nije sigurnosno dovršen.

### Privatni SFTP ključ

Korisnički odabrana private-key putanja ne predaje se OpenSSH-u odmah nakon običnog path checka. ByFTP:

- ograničava ključ na običnu lokalnu datoteku bez symlink/junction/reparse preusmjeravanja
- odbija praznu ili veću od 1 MiB datoteku
- otvara ključ i uspoređuje filesystem identitet `Lstat` objekta sa stvarno otvorenim handleom
- čita bounded sadržaj i ponovno provjerava identitet, veličinu i modification time nakon čitanja
- zapisuje sadržaj u nasumično imenovan privatni `0600` session snapshot i briše memorijsku kopiju
- OpenSSH konfiguracija koristi session snapshot, ne izvornu korisničku putanju
- `SFTP.Close()` briše snapshot zajedno sa session configom i kratkotrajnim `known_hosts`

Time zamjena izvorne key putanje nakon ByFTP validacije ne mijenja ključ aktivne sesije.

## Profilni identitet

Windows profilne tajne koriste `internal/profilebinding`:

- endpoint = protokol + normalizirani host + port
- account = endpoint + korisničko ime
- private-key identitet = account + privatni ključ

Lozinka se ne prenosi na drugi account, passphrase na drugi ključ, a SFTP host-key pin na drugi endpoint. Prazna putanja privatnog ključa je autoritativna i ne smije vratiti staru spremljenu vrijednost.

## SFTP host-key trust

SFTP spajanje prvo skenira host key, računa SHA-256 fingerprint i veže sesiju uz konkretni odabrani algoritam. `known_hosts` i OpenSSH session config kratkotrajni su i privatni.

Spremljeni pin koristi se samo za isti endpoint. Privremena promjena hosta ili porta ne može naslijediti ili prepisati pin drugog profila.

Windows crash-cleanup SFTP session artefakata izvršava se tek nakon `EnsureNoRedirectDirectory`. Linux/macOS startup ne čisti zajedničku SFTP session mapu jer više terminalskih procesa smije raditi paralelno; normalni `Close()` i deferred cleanup uklanjaju artefakte vlastite sesije.

## Vanjski mrežni alati

- Windows koristi sistemski System32 curl/OpenSSH
- Linux/macOS koriste nativni `curl`, `sftp`, `ssh-keyscan` i `ssh-keygen`
- curl ne nasljeđuje proxy/TLS override varijable i dobiva izričiti no-proxy
- OpenSSH blokira ProxyCommand, ProxyJump, agent, PKCS#11 provider, KnownHostsCommand, local command i forwarding
- bez eksplicitnog SFTP ključa koristi se `IdentitiesOnly yes` i `IdentityFile none`
- host, korisnik i private-key putanja nisu na OpenSSH command lineu

### FTPS certifikat i opoziv

FTPS uvijek zadržava provjeru TLS certifikata koju pruža aktivni curl TLS backend i postavlja minimalni TLS zahtjev kroz `tlsv1.2`; eksplicitni FTPS dodatno zahtijeva TLS pomoću `ssl-reqd`.

ByFTP ne koristi `ssl-no-revoke`. Na Windowsu prvo lokalno i bounded provjerava `curl --version`. Ako curl podržava opciju uvedenu u 7.70.0, koristi `ssl-revoke-best-effort`: provjera opoziva ostaje uključena kada su potrebni podaci dostupni, ali nedostupan ili offline distribution point sam po sebi ne ruši vezu.

Ako je Windows curl stariji, version probe ne uspije ili izlaz nije prepoznatljiv, ByFTP ne šalje nepoznatu curl opciju. U tom slučaju zadržava se zadano Schannel ponašanje provjere opoziva umjesto prelaska na `ssl-no-revoke`. Time stari Windows/curl ne gubi FTPS samo zbog nepodržanog argumenta, a sigurnosna provjera se ne isključuje.

Linux/macOS ne dobivaju Schannel-specifičnu opciju. Ponašanje provjere certifikata i opoziva tamo ostaje u odgovornosti TLS backend-a sistemskog curl-a.

Regresijski Go testovi i `audit_privacy.py` zajedno moraju odbiti ponovno uvođenje `ssl-no-revoke`, potvrditi capability-gated Windows best-effort politiku te fallback bez nepoznate opcije.

## Datoteke i putanje

- udaljene i lokalne putanje prolaze traversal/control-character validaciju
- upload ne prati symlink/reparse objekt
- download staging mora biti regularna datoteka i ne smije biti reparse/symlink
- Windows no-replace koristi exclusive `MoveFileExW` bez replace zastavice
- Linux no-replace koristi kernel `renameat2(RENAME_NOREPLACE)` umjesto `Lstat` + običnog `rename` obrasca
- macOS regularne staging datoteke koriste ekskluzivno `link` + `unlink` premještanje; postojeće odredište se ne prepisuje
- `RemoveTreeNoFollow` blokira filesystem root, ne slijedi symlink/junction/reparse objekt i direktorij čita preko prethodno verificiranog otvorenog handlea
- identitet direktorija ponovno se provjerava prije child rekurzije i završnog uklanjanja
- rekurzivne operacije imaju depth/item limite
- queued transfer ponovno validira lokalni root prije svakog pokušaja

### Lokalni upload source snapshot

FTP/FTPS i SFTP child procesi više ne dobivaju originalnu lokalnu korisničku putanju nakon zasebnog path checka. Za stvarni upload ByFTP:

- `Lstat`-om zahtijeva običnu datoteku bez symlink/junction/reparse preusmjeravanja
- otvara izvor i pomoću `os.SameFile` potvrđuje da handle pripada istom filesystem objektu
- iz otvorenog handlea izrađuje byte-for-byte kopiju u privatnom nasumičnom `byftp-upload-*` direktoriju
- tijekom kopiranja računa SHA-256, zatim isti otvoreni izvor ponovno čita od početka i zahtijeva isti broj bajtova i isti SHA-256
- `curl`/OpenSSH-u predaje samo privatnu snapshot putanju, ne originalni lokalni pathname
- nakon mrežnog čitanja ponovno provjerava filesystem identitet, veličinu, mtime i puni SHA-256 snapshota
- `Verify()` uvijek zatvara handle i uklanja snapshot prije remote revalidation/commit faze; cleanup failure blokira commit i adapter čisti remote temp objekt

Ovaj model zatvara path-replacement/symlink TOCTOU između ByFTP validacije i child-process opena te detektira normalne in-place/torn-copy promjene sadržaja. Ne tvrdi apsolutnu otpornost protiv proizvoljnog privilegiranog procesa koji može istodobno mijenjati kernel/filesystem stanje mimo korisničkih dozvola. Sigurnosni tradeoff je dodatni privremeni disk prostor približno veličini upload datoteke i dodatna lokalna čitanja radi sadržajne stabilnosti; nedostatak prostora završava fail-closed prije finalnog remote commit-a.

### Remote upload commit

Nakon što se lokalni snapshot verificira i ukloni, temp upload još nije automatski finalna datoteka. ByFTP ponovno lista remote direktorij neposredno prije `commitRemoteTemp`:

- novonastali direktorij ili symlink pod finalnim imenom blokira commit i čisti remote temp
- `SkipExisting` ponovno se primjenjuje na svježe remote stanje
- overwrite/backup/rollback koristi svježi remote snapshot, ne listing napravljen prije dugog uploada
- završni listing failure blokira commit i čisti temp objekt

Nove handle/identity/content provjere i platformske no-replace primitive zatvaraju najopasnije ranije check→follow/overwrite prozore. Dio stdlib operacija i dalje na kraju mora imenovati putanju, pa kod ne tvrdi apsolutnu otpornost na svaki namjerni same-user filesystem race. Potpuna takva garancija tražila bi platform-specific handle-relative operacije za cijeli traversal/destructive lifecycle.

## Session lifecycle

Svaka `Operation` registrira aktivnu referencu. Disconnect prvo blokira nove operacije, zatim otkazuje session context i čeka postojeće reference. Adapter se ne zatvara ispod aktivnog `List`, rename, chmod ili transfer poziva.

Ako caller deadline istekne, cleanup nastavlja odvojeno. Reconnect je blokiran dok prethodna sesija nije stvarno očišćena. Ponovljeni disconnect koristi isti close-state.

## Transfer izolacija

Transfer posao pamti connection generation i opaque connection identity. Retry na drugi server/account nije dopušten. Event API vraća duboke kopije. Kasni cancel nakon uspješnog ili preskočenog transfera ne mijenja autoritativni završni status.

Terminal koristi autoritativni snapshot transfer reda za završni status, pa ne ovisi o tome je li završni event još prisutan u ograničenoj event povijesti.

## State/config

State safe-open provjerava regularnost i stabilnost stvarno otvorene datoteke. Nepouzdani current zapis ne mora srušiti startup: koristi se provjerena prethodna generacija ili zadane vrijednosti.

## Go toolchain build privatnost

Go 1.23+ ima zasebnu telemetry postavku. `GOTELEMETRY` je vrijednost koju `go env` prijavljuje; obična OS env varijabla nije pouzdan način za gašenje prikupljanja.

ByFTP produkcijski CI i release workflow zato izvršavaju:

```text
go telemetry off
```

i potvrđuju da `go telemetry` vraća `off` prije testova/builda.

Produkcijske build skripte dodatno odbijaju rad ako telemetry mode nije `off`. Lokalne skripte ne mijenjaju globalnu Go postavku potajno, nego korisniku jasno kažu koju naredbu treba izvršiti.

Ovo se odnosi na Go **build toolchain**, ne na ByFTP runtime. ByFTP aplikacija sama nema ugrađenu telemetriju.

## Release sigurnost

Aktualna produkcijska bazna linija dodatno učvršćuje release granicu:

- `VERSION` je jedini kanonski broj
- automatski release okidač je samo promjena `VERSION` na `main`
- tag koji publisher izradi ne pokreće novi release workflow
- normalni PR CI blokira produkcijsku promjenu pod već postojećim `v$VERSION` tagom ako isti skup promjena ne mijenja `VERSION`
- svi release runovi dijele jednu `byftp-release` concurrency grupu
- zaseban release quality job ponovno pokreće audite, Python regresije, unit, race i vet
- Windows job gradi i verificira x64 i x86
- Linux job testira i gradi amd64, arm64 i i386 DEB
- macOS job testira i gradi Universal Intel+Apple Silicon PKG
- publish čeka sva četiri joba
- release staging mora imati točno 10 očekivanih platformskih paketa
- centralni publisher veže tag uz točan commit i uspoređuje postojeći asset po veličini i SHA-256 digestu
- rerun smije nadopuniti samo nedostajući potvrđeni asset
- dodatni ili neočekivani javni asset zaustavlja izdanje
- GitHub Windows Package mora koristiti isti kanonski `VERSION` kao aplikacija i GitHub Release

## Potpisivanje

Windows paketi nisu Authenticode/Verified Publisher dok ne postoji stvarni Brendigo code-signing certifikat. macOS PKG nije Developer ID potpisan/notariziran bez stvarnog Apple certifikata. Certifikati i privatni ključevi ne smiju biti pohranjeni u repozitoriju.

## Automatizirani gate

`scripts/audit_security.py` zaključava SFTP procesne granice, connect smoke, AskPass prompt, context propagation, IPv6 normalizaciju, runtime tajne, profile binding, session lifecycle i filesystem zaštite.

`scripts/test_filesystem_hardening.py` dodatno zaključava Linux kernel no-replace, macOS exclusive-link aktivaciju, stable-directory delete, private-key snapshot i SFTP cleanup redoslijed.

`scripts/test_remote_commit_revalidation.py` zaključava da FTP/FTPS i SFTP nakon temp uploada osvježavaju remote stanje prije `commitRemoteTemp`.

`scripts/test_upload_source_snapshot.py` zaključava open-handle byte-copy, dvostruku source SHA-256 provjeru, post-upload snapshot SHA-256, cleanup u `Verify()` i adapter ordering `prepare → child snapshot path → verify/cleanup → remote revalidation → commit`.

`scripts/audit_privacy.py` zaključava runtime mrežnu politiku, FTPS revocation pravilo i stvarno gašenje Go build telemetrije.

`scripts/audit_release.py` zaključava single-trigger/serialized release model, production quality gate, platformsku matricu, staging allowlist i završni javni asset ugovor.

`scripts/audit_version.py` dodatno blokira drift trenutačne verzije u README/CHANGELOG i verzionirane aktualne tvrdnje u produkcijskoj dokumentaciji.
