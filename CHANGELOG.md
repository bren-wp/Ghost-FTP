# Povijest promjena

## 1.0.5 — Shared hosting bez putanjskih iznenađenja

**Fokus izdanja:** jednostavnije i pouzdanije spajanje na tipične shared-hosting FTP/FTPS račune te jasnija komunikacija prema korisniku.

- FTP raw control naredbe (`MKD`, `RNFR`, `RNTO`, `DELE`, `RMD`, `SITE CHMOD`) sada koriste login/home relativnu putanju, isti logički namespace koji korisnik vidi kroz listing i URL upload/download
- time logička `/public_html/...` putanja više ne postaje pogrešno server-absolute `/public_html/...` operand na non-chrooted shared-hosting serverima
- quote-only FTP operacije koriste `no-body`, pa uspješan mkdir/rename/delete/chmod više ne ovisi o nepotrebnom naknadnom directory data-channel transferu
- MLSD kompatibilnost je poboljšana: ako MLSD ne radi ili vrati neprepoznatljiv format, a obični LIST uspije, ByFTP pamti LIST fallback do kraja sesije i ne ponavlja neuspjeli MLSD pri svakom refreshu
- dodani su process-smoke testovi za shared-hosting username oblika `account@example.com`, home-relative `public_html` i control-only quote ponašanje
- dodan je MLSD→LIST regresijski test koji zahtijeva slijed jednog MLSD pokušaja i daljnjih LIST poziva
- korisničke FTP greške sada jasnije objašnjavaju `530` login problem, uključujući puni `korisnik@domena` username, `421` limit konekcija i `425/426` data-channel problem
- Windows connect ekran koristi konkretnije host/username cueove i šire polje za shared-hosting korisničko ime
- glavni README i kompletna `docs/` dokumentacija preuređeni su u benefit-first, marketinški jasniji hrvatski stil bez uklanjanja tehničkih ograničenja i sigurnosnih činjenica
- dodan je zaseban `docs/SHARED-HOSTING.md` vodič za prvi FTP/FTPS spoj, `public_html`, MLSD/LIST fallback, pasivni FTP, WordPress workflow i najčešće hosting greške
- verzija je povećana na `1.0.5`; već objavljeni `v1.0.4` ostaje nepromjenjiv

## 1.0.4 — Transfer generation binding

- `ReserveBatch` više ne dohvaća `ConnectionIdentity()` pa tek nakon toga uzima aktualnu transfer generation; generation se sada capturea pod `Manager.mu` prije identity lookup-a i ponovno provjerava nakon povratka
- ako disconnect/reconnect ili drugo lifecycle prebacivanje promijeni generation tijekom `ConnectionIdentity()` poziva, batch rezervacija se fail-closed odbija prije rezerviranja kapaciteta ili dodavanja poslova
- time stari connection ID više ne može biti uparen s novom generation i kasnije pokrenuti upload/download na pogrešnoj sesiji
- `RetryBatch` koristi isti dvostruki generation guard prije bilo kakve promjene statusa failed/cancelled posla u `queued`
- `ConnectionIdentity()` se namjerno poziva bez držanja `transfer.Manager.mu`, pa hardening ne uvodi lock-order/deadlock ovisnost između transfer i remote managera
- dodani su deterministički Go testovi koji inkrementiraju generation iz samog `ConnectionIdentity()` callbacka; rezervacija mora ostati bez kapaciteta/poslova, a retry mora ostaviti posao potpuno nepromijenjenim
- dodan je Python `unittest` koji zaključava ordering `capture generation → unlock → ConnectionIdentity → lock → generation recheck → mutation`
- verzija se povećava na `1.0.4` umjesto mijenjanja već objavljenog i nepromjenjivog `v1.0.3`

## 1.0.3 — Stabilni lokalni upload snapshot

- FTP/FTPS i SFTP upload više ne predaju vanjskom `curl`/OpenSSH procesu izvornu korisničku lokalnu putanju nakon zasebnog `Lstat` checka
- neposredno prije stvarnog uploada ByFTP otvara validirani regularni izvor i provjerava da otvoreni handle pripada istom filesystem objektu kao prethodni `Lstat`
- izvor se kopira byte-for-byte u kriptografski nasumični privatni `byftp-upload-*` direktorij; child proces dobiva samo snapshot putanju
- tijekom izrade kopije računa se SHA-256 digest, a isti već otvoreni izvor se zatim ponovno čita u cijelosti i njegov digest mora biti identičan snapshotu
- ako se izvor tijekom izrade snapshota promijeni po identitetu, veličini, mtimeu, broju pročitanih bajtova ili SHA-256 sadržaju, upload se blokira prije mrežnog prijenosa
- nakon što `curl`/OpenSSH završi čitanje snapshota ByFTP ponovno provjerava njegov filesystem identitet, veličinu, mtime i puni SHA-256 prije remote revalidation/commit faze
- promjena originalne putanje nakon pripreme snapshota više ne može preusmjeriti sadržaj koji child proces šalje
- same-size/same-mtime izmjena snapshota više ne prolazi samo kroz metadata provjeru jer završni SHA-256 mora odgovarati početnom digestu
- privremeni snapshot direktorij briše se kroz postojeći no-follow `RemoveTreeNoFollow`, a ne nekontrolirani `RemoveAll`
- sigurnija kopija namjerno zahtijeva dodatni lokalni privremeni prostor približno veličini datoteke koja se šalje i dodatno lokalno čitanje radi sadržajne stabilnosti
- oba adaptera zadržavaju 1.0.2 post-upload remote revalidaciju, pa se lokalni source snapshot provjerava prije fresh remote provjere i transakcijskog `commitRemoteTemp`
- dodani su Go testovi za zamjenu originalne putanje, snapshot tampering, same-size/same-mtime tampering, cleanup i symlink izvor
- dodan je Python `unittest` koji zaključava da oba adaptera koriste `source.Path()` i `source.Verify()` prije remote commit faze te da helper ostaje copy+SHA-256, a ne hard-link pristup
- verzija se povećava na `1.0.3` umjesto mijenjanja već objavljenog i nepromjenjivog `v1.0.2`

## 1.0.2 — Remote commit revalidacija

- FTP/FTPS i SFTP upload više ne donose završnu overwrite/backup odluku prema remote direktorijskom snapshotu snimljenom prije potencijalno dugog prijenosa
- nakon uspješnog prijenosa u kriptografski nasumični `.byftp-part-*` objekt ByFTP ponovno lista odredišni direktorij neposredno prije commit/rename faze
- ako se cilj tijekom uploada pretvori u direktorij ili simboličku poveznicu, commit se blokira, privremeni upload briše i postojeći objekt se ne dira
- ako se obična ciljna datoteka pojavi tijekom uploada i uključen je `SkipExisting`, ByFTP briše temp objekt i vraća `ErrSkipped` umjesto da prepiše novonastalu datoteku
- ako je overwrite dopušten, `commitRemoteTemp` dobiva svježi remote snapshot pa backup/rollback odluku donosi prema stvarnom stanju neposredno prije aktivacije
- ako završna revalidacija direktorija ne uspije, temp objekt se fail-closed čisti i završni rename se ne pokušava
- FTP/FTPS i SFTP koriste isti `revalidateRemoteCommit` helper kako sigurnosna logika ne bi divergirala između adaptera
- dodani su Go regresijski testovi za novonastalu datoteku, direktorij, symlink, listing grešku, overwrite i još-nepostojeći cilj
- dodan je Python source-ordering regression koji potvrđuje da oba upload adaptera pozivaju revalidaciju prije `commitRemoteTemp`
- verzija se povećava na `1.0.2` umjesto mijenjanja već objavljenog i nepromjenjivog `v1.0.1`

## 1.0.1 — Filesystem i SFTP hardening

- ispravljena je Unix no-replace semantika koja je ranije radila `Lstat(dst)` pa obični `rename`, što je ostavljalo TOCTOU prozor u kojem je konkurentno odredište moglo biti prepisano
- Linux lokalna aktivacija i rollback sada koriste kernel `renameat2(RENAME_NOREPLACE)` umjesto check-then-rename obrasca
- macOS regularne staging datoteke koriste ekskluzivno hard-link + unlink premještanje; ako odredište već postoji, ono se ne prepisuje
- generički ne-Windows fallback također koristi exclusive hard-link semantiku za obične datoteke umjesto neatomskog destination checka
- rekurzivno lokalno brisanje više ne radi `os.ReadDir(target)` nakon zasebnog path checka; direktorij se otvara, verificira s `os.SameFile`, čita preko stabilnog handlea i ponovno provjerava prije destruktivnih koraka
- dodana je regresija koja zamijeni provjereni direktorij symlinkom i potvrđuje da se vanjski sadržaj ne obilazi niti briše
- privatni SFTP ključ sada ima maksimalnu veličinu od 1 MiB i prije OpenSSH korištenja se čita iz stabilno verificiranog regularnog file handlea
- sadržaj privatnog ključa kopira se u kriptografski nasumično imenovan privatni `0600` session snapshot; memorijska kopija se briše nakon stvaranja, a session snapshot pri `Close()`
- kasnija zamjena ili izmjena izvorne private-key putanje ne mijenja ključ koji koristi aktivna SFTP sesija
- Linux/macOS `NewManager` više pri pokretanju ne briše zajedničke SFTP temp artefakte, pa paralelni terminalski proces ne može srušiti konfiguraciju druge aktivne sesije
- Windows startup cleanup privremenih SFTP datoteka premješten je iza `EnsureNoRedirectDirectory`, nakon potvrde da ByFTP session mapa nije symlink/junction/reparse preusmjeravanje
- novi Python filesystem-hardening regression zaključava Linux `RENAME_NOREPLACE`, macOS exclusive link, stable-directory delete, private-key snapshot i siguran SFTP cleanup redoslijed
- postojeći release-version guard osigurava da se 1.0.1 objavi kao novi nepromjenjivi semantički release umjesto prepisivanja već objavljenog `v1.0.0`

## 1.0.0 — Stabilna produkcijska bazna linija

- produkcijska semantička verzija resetirana je na `1.0.0`; `VERSION` je jedini autoritativni izvor verzije za aplikaciju, platformske pakete, GitHub Release i `ByFTP.Windows` GitHub Package
- pooštrena je validacija hosta: bracketirani IPv6 sada mora imati točno jedan ispravno uparen par uglatih zagrada, bracketirani IPv4 se odbija, a host s priključenim portom ne prolazi kao hostname
- dodane su regresije za nepotpune i višestruke IPv6 zagrade kako se sigurnosni propust ne bi vratio budućim refaktorom
- zadržan je fail-closed SFTP host-key model s SHA-256 fingerprintom, endpoint bindingom, privatnim kratkotrajnim `known_hosts` zapisom i blokiranjem promijenjenog pina
- zadržano je credential binding pravilo: spremljena lozinka vrijedi samo za isti protokol, host, port i korisnika, a passphrase dodatno za isti privatni ključ
- Windows profilne tajne ostaju zaštićene DPAPI-jem; Linux/macOS terminal ne nudi trajno spremanje vjerodajnica
- Linux/macOS FTP/FTPS lozinke ostaju samo u memoriji procesa; SFTP password/passphrase na tim platformama ostaje fail-closed dok ne postoji siguran Unix credential broker
- transfer queue zadržava batch validaciju prije mutacije, connection-identity zaštitu za retry, cancel/pause/resume, kontrolirani broj paralelnih radnika i bounded event history
- disconnect/reconnect lifecycle ostaje referentno brojan i ne zatvara remote adapter ispod aktivne operacije; cleanup se nastavlja sigurno i nakon isteka korisničkog deadlinea
- lokalni download putovi ponovno se provjeravaju protiv symlink, junction i reparse traversal izlaza prije stvarnog transfera
- vanjski `curl` i OpenSSH procesi koriste sanitizirani environment, ograničeni stdout/stderr, context timeout/cancel i kontrolirane argumente/standardni ulaz
- state spremište zadržava ograničenje veličine, privatne dozvole, fsync privremene datoteke, atomsku zamjenu i prethodnu generaciju za oporavak
- Windows x64/x86, Linux amd64/arm64/i386 i macOS Universal paketi ostaju zasebni produkcijski build gateovi
- release workflow ostaje serijaliziran i fail-closed: prije javne objave ponovno izvršava audite, Python regresije, `go test`, `go test -race`, `go vet` i platformske build provjere
- javni release staging dopušta samo ugovoreni skup platformskih paketa i zajedničkih metapodataka; postojeći asset pod istim nazivom mora imati istu veličinu i SHA-256 digest
- GitHub Packages objava koristi istu `VERSION` vrijednost i `--skip-duplicate`, pa ne postoji zasebna ručno održavana verzija paketa
- README je ponovno usklađen s realnim mogućnostima platformi, sigurnosnim ograničenjima, build/release modelom i verzijom `1.0.0`

### Razvoj prije 1.0.0

Prije stabilne 1.0.0 bazne linije projekt je prolazio kroz interne razvojne 2.x oznake tijekom intenzivnog hardeninga, cross-platform build rada i sigurnosnih revizija. Detaljna povijest tih razvojnih commitova ostaje dostupna u Git povijesti repozitorija; od `1.0.0` nadalje ovaj CHANGELOG prati javnu stabilnu semantičku liniju proizvoda.
