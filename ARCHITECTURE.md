# ByFTP 2.13.0 arhitektura

ByFTP je jedan native Win32 desktop proces. Nema browser UI, localhost HTTP servera ni mrežnog IPC-a između UI-a i enginea.

## Moduli

- `cmd/byftp` — startup i strogo ograničen SFTP AskPass način rada
- `internal/desktop` — native Win32 dark/Fluent UI; transfer-event state koristi batch ID indeks umjesto punog scaniranja reda za svaki event
- `internal/api` — potpuno tipizirani in-process engine API i tree-transfer planiranje; nema string-kanala ni JSON dispatchera
- `internal/remote` — FTP/FTPS preko Windows curl i SFTP preko Windows OpenSSH, uz session-scoped cancellation lifecycle
- `internal/transfer` — queue, connection-generation rezervacije, endpoint identity binding, auto-retry, batch pause/resume/cancel/retry i event stream
- `internal/config` — atomsko lokalno spremište; Windows profile metadata i credential blobovi nalaze se unutar DPAPI-zaštićene envelope, settings ne sadrži vjerodajnice; state safe-open uspoređuje identitet validiranog i stvarno otvorenog regularnog objekta
- `internal/itemlist` — zajedničko stabilno directory-first sortiranje s jednom case-fold alokacijom po stavci, namijenjeno velikim file-listovima
- `internal/security` — input/path validation i Windows DPAPI
- `internal/localfs` — lokalne file operacije i bounded directory enumeration
- `internal/platform` — Win32 dijalozi, Known Folder/System Directory API-ji, registry, shortcut, atomic replace i single-instance

Runtime nema vanjske Go dependencies. Mrežni adapteri su zaključani na Windows system alate i ne nasljeđuju proxy/SSH helper routing.

## Privacy boundary

Lozinka/passphrase ne prolaze kroz generički JSON dispatcher. Spremljeni DPAPI blob ostaje zaštićen kroz profile/manager sloj i otključava se tek u protocol adapteru neposredno prije sistemskog curl/OpenSSH poziva. SFTP host-key trust koristi jednokratni, vremenski ograničeni DPAPI pending blob kako potvrda novog servera ne bi zahtijevala ponovno držanje plaintext vjerodajnice u UI toku.

## State-file boundary

Config/store čitanje prvo radi `Lstat` i odbija ne-regularni/symlink objekt, zatim otvara datoteku i uspoređuje `os.SameFile` identitet pre-open i post-open objekta. Ako je putanja zamijenjena između provjere i otvaranja, sadržaj se ne parsira. Store zatim može koristiti provjerenu `.previous` generaciju ili sigurne zadane vrijednosti.

## 2.9 session privacy boundary

SFTP session config dobiva nasumični alias. OpenSSH command line ne nosi korisnički host, username ni privatni key path; ti podaci su samo u kratkoživućoj lokalnoj session konfiguraciji. AskPass nema credential datoteku: helper dobiva samo DPAPI-zaštićene blobove kroz child environment i radi isključivo kada ga pokrene očekivani System32 OpenSSH proces.

Transfer batch rezervacija pamti connection generation. Disconnect/novi Connect invalidiraju rezervacije stare sesije, što sprječava stale tree-transfer commit nakon brzog reconnecta.

## 2.11–2.13 performance i queue granica

UI i engine su isti proces i koriste izravne tipizirane pozive bez JSON marshal/unmarshal koraka. Settings store radi kao single-writer/thread-safe cache nakon prvog čitanja. Transfer job zadržava samo interni opaque connection identity u transfer manageru; identitet se ne prikazuje niti sprema na disk, a služi samo za blokiranje Retry-a prema drugom serveru/accountu. Superseded directory refresh context se aktivno otkazuje kako stari filesystem/curl/sftp rad ne bi ostao do timeouta.

Transfer manager emitira inkrementalne evente, a desktop 2.13 primjenjuje cijeli event batch kroz mapu `job ID -> slice index`, čime se kompleksnost velikog update bursta svodi na O(trenutni red + eventi). `Clear Finished` uklanja završene engine job/event backing strukture i UI dedup ID-eve. Lokalni directory sort izračunava case-folded ključ samo jednom po stavci umjesto u svakom sort comparator pozivu.

## Release boundary

`VERSION` je kanonska release verzija. CI provjerava da za nju postoji generabilan CHANGELOG odjeljak. Release workflow iz tog odjeljka izrađuje `RELEASE-NOTES.txt`, a `BUILD-METADATA.txt` zapisuje samo source commit/ref, Go toolchain, platformu i GitHub Actions run identifikatore. Windows bundle i GitHub Release imaju zasebne SHA-256 zapise; Source ZIP nastaje iz `git archive HEAD`, bez lokalnog build outputa.
