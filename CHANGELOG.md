# Povijest promjena

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
