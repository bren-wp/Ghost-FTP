# ByFTP WEB 1.9.1

ByFTP WEB je samostalni, višekorisnički FTP/FTPS/SFTP file manager za PHP shared hosting. Smješten je u zasebnoj mapi unutar glavnog ByFTP repozitorija i prati isti kanonski release broj kao Windows, Linux, macOS, Android i iOS klijenti.

Od izdanja 1.9.0 WEB ima vlastiti verificirani javni artefakt; za aktualno izdanje paket je `ByFTP-1.9.1-WEB-shared-hosting.zip`. Paket se generira isključivo iz Git-tracked produkcijskih datoteka u `ByFTP WEB/` i ne smije sadržavati runtime korisnike, konfiguraciju, backupove, lockove, cache ili druge podatke nastale korištenjem instalacije.

## Što je uključeno

- FTP i FTPS preko PHP `ext-ftp`
- SFTP preko PHP `ext-ssh2`, uključujući password i public-key autentikaciju
- obavezni pinned SHA-256 SFTP host fingerprint prije stvaranja SFTP klijenta
- sigurni spremljeni profili po korisniku; username/password/ključni materijal šifriraju se Sodium secretboxom ili AES-256-GCM/OpenSSL fallbackom
- višekorisnički računi, admin upravljanje korisnicima i izolirani workspace po korisniku
- pregled direktorija, upload datoteka i mapa, download, create/rename/delete, copy/move/duplicate, CHMOD, editor, search, favorites i batch operacije
- ZIP download/kreiranje/raspakiranje kada je `ext-zip` dostupan
- PWA instalacija bez cacheiranja autentificiranih stranica, API-ja i download/preview odgovora
- responzivni desktop/mobitel UI bez CDN-a i bez obveznog Node/Docker/cron runtimea

## Sigurnosni i stability model u 1.9.1

Remote putanje i nazivi su fail-closed. Aplikacija ne pretvara `\` u `/`, ne prihvaća traversal, `./`, dvostruke separatore ni protocol-control znakove. Host, port, timeout, account identity i SFTP fingerprint provjeravaju se prije normalizacije; vrijednosti kao `22junk`, rubni razmaci ili CR/LF ne mogu postati valjani unos tihim čišćenjem.

SSRF zaštita je uključena prema zadanim postavkama: privatne, loopback i rezervirane IP adrese nisu dopuštene dok administrator eksplicitno ne uključi private-host pristup. DNS se razrješava na konkretne validirane targete, a transport se spaja na provjerenu IP adresu kako se host ne bi ponovno razrješavao nakon sigurnosne provjere.

SFTP ne može koristiti password ili privatni ključ bez pinned SHA-256 identiteta servera. `ClientFactory` odbija SFTP profil bez fingerprinta prije mrežne veze, a `SftpClient` uspoređuje stvarni server fingerprint s pinom.

### Bounded JSON runtime state

U 1.9.1 svaki JSON runtime state file ima limit od 8 MiB pri čitanju i pisanju. `JsonStore` čita najviše limit + 1 bajt prije `json_decode`, pa oštećena ili nenormalno velika state datoteka više ne može natjerati aplikaciju da prvo učita proizvoljno velik sadržaj u PHP memoriju.

### FTP LIST fallback i nazivi datoteka

Kada server nema MLSD i koristi se Unix `LIST`, tekst ` -> ` uklanja se samo za stvarni symlink redak. Obična datoteka naziva poput `report -> archive.txt` ostaje točno tog imena umjesto da se pogrešno prikaže kao `report`.

### ZIP extraction bez kasnog parcijalnog writea

Raspakiravanje ZIP-a provjerava kompletnu arhivsku topologiju i postojeće remote konflikte prije remote mutacije. Sve file stavke se zatim lokalno materializiraju u zaštićeni temp prostor i provjerava se stvarni kumulativni broj dekomprimiranih bajtova. Tek nakon što je cijela arhiva čitljiva i unutar limita počinju remote `mkdir` i atomic upload operacije.

Time kasnija oštećena/nečitljiva ZIP stavka ili netočan metadata size ne može uzrokovati da se ranije stavke prvo upišu na server. Limit od 512 MiB primjenjuje se i na deklarirane i na stvarno dekomprimirane bajtove. Svi staged temp fajlovi uklanjaju se u `finally` cleanupu.

### Privilegirane dijagnostike

`diagnostics.php` dostupan je samo administratoru. Stranica prikazuje PHP/OpenSSL/runtime i hosting capability informacije koje običnom prijavljenom korisniku nisu potrebne.

### Fail-closed storage i recovery

Security/privacy state ne vraća se automatski iz starije `.bak` generacije kada primarna datoteka nedostaje ili je oštećena. To vrijedi za:

- `storage/app.json` — encryption ključ i runtime sigurnosna politika
- `storage/users.json` — autentikacija, role, active/deleting i session generation
- rate-limit brojače
- korisničke `profiles.json` — šifrirane FTP/SFTP vjerodajnice i SSH key materijal
- korisničke `preferences.json` — favorites, last/recent remote paths i client state
- legacy profile/preferences migraciju

`.bak` se zadržava za eksplicitni administratorski/operator recovery, ali runtime ne smije tiho vratiti stariju lozinku, obrisani profil, obrisanu remote path povijest ili slabiju aplikacijsku politiku.

### Authentication concurrency

Login i registracija koriste atomarni `RateLimiter::consume()` prije password/hash rada. Login prvo troši IP budget; ako je IP blokiran, account-specific budget se uopće ne dira.

Promjena lozinke koristi compare-and-swap nad točnom hash generacijom koja je verificirana. Dva paralelna zahtjeva sa starom lozinkom ne mogu oba commitati. Završetak autentikacije također pod zaključanim registry updateom potvrđuje da je verificirani hash još aktualan, pa login sa starom lozinkom ne može objaviti novu sesiju nakon paralelnog password reset/change događaja.

### Credential binding

Prazno password/passphrase/private-key polje može naslijediti postojeću tajnu samo kada su endpoint, account i relevantni key identity isti. Promjena hosta, porta, usernamea ili privatnog ključa prekida nasljeđivanje starih vjerodajnica.

### Account lifecycle

Brisanje korisnika je dvofazno: račun se prvo označi `deleting` i deaktivira, zatim se workspace sigurno uklanja, a registry zapis nestaje tek nakon potvrđenog filesystem cleanup-a. Root workspace symlink se samo unlinka i nikad se rekurzivno ne prati. Cleanup failure ostavlja retryable deaktivirani zapis umjesto orphaned privatnih podataka.

Aplikacija koristi `SameSite=Strict`, HttpOnly session cookie, CSRF tokene, `Sec-Fetch-Site` zaštitu za cross-site POST, CSP, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, no-store za osjetljive odgovore i `X-Robots-Tag`/`robots.txt` zabranu indeksiranja.

## Minimalni zahtjevi

- PHP 8.1+
- `ext-ftp` za FTP/FTPS
- Sodium ili OpenSSL za šifrirano spremanje vjerodajnica
- zapisiv `storage/` direktorij
- `ext-ssh2` za SFTP
- `ext-zip` za ZIP funkcije
- HTTPS za produkciju

## Instalacija na shared hosting

1. Preuzmi `ByFTP-1.9.1-WEB-shared-hosting.zip` iz odgovarajućeg GitHub Releasea ili koristi verificirani source iz `ByFTP WEB/`.
2. Raspakiraj sadržaj na odabranu web putanju.
3. Provjeri da PHP može zapisivati u `storage/`, `storage/tmp/`, `storage/logs/` i `storage/users/`.
4. Otvori `/setup` i izradi administratorski račun.
5. Nakon administratorske prijave otvori **Diagnostics** i provjeri FTP/SFTP/ZIP/crypto mogućnosti hostinga.
6. Za svaki SFTP profil unesi provjereni SHA-256 host fingerprint prije testiranja/spremanja veze.
7. U produkciji koristi HTTPS. `.htaccess` već zabranjuje direktan pristup `app/`, `storage/`, README-u, Composer metapodacima i JSON/backup/lock datotekama.

Ako Apache `mod_rewrite` nije dostupan, PHP datoteke i dalje mogu raditi direktno (`login.php`, `index.php`, `api.php`), ali clean URL-ovi neće biti aktivni.

## Recovery

Ako je primarni `app.json`, `users.json`, profile/preference registry ili rate-limit state oštećen, ByFTP namjerno neće automatski odabrati stariji backup. Prvo provjeri integritet i sadržaj sigurnosne kopije, zatim eksplicitno vrati provjerenu primarnu datoteku. Novi setup je blokiran dok postoje podaci koji mogu zahtijevati recovery kako novi encryption ključ ne bi učinio stare vjerodajnice nedostupnima.

## Verzija, cache i release paket

`VERSION` u ovoj mapi mora biti identičan root `VERSION`. `app/bootstrap.php` učitava taj broj pri pokretanju, Composer metadata ga ponavlja radi package konzistentnosti, a repository audit blokira mismatch. Service Worker koristi `byftp-static-v1.9.1` i pri aktivaciji uklanja prethodne `byftp-static-*` cache generacije.

`scripts/package_web.py` iz root repozitorija generira `ByFTP-1.9.1-WEB-shared-hosting.zip` iz `git ls-files` popisa. Packager odbija symlink i unsafe archive putanju, provjerava arhivirani `VERSION`, Composer verziju i PWA cache namespace te nakon izrade ponovno provjerava sadržaj ZIP-a.

## Legacy podaci

Migracija stare single-admin web instalacije ostaje podržana. Ako postoje legacy profile/preferences JSON podaci, migrator čita isključivo valjanu primarnu generaciju. Corrupt-primary ili backup-only stanje zahtijeva ručni recovery i ne smije automatski migrirati stale credential/privacy state.

## Testiranje

Iz root repozitorija preporučena produkcijska provjera je:

```bash
python scripts/audit_web.py
python scripts/audit_security.py
python scripts/audit_privacy.py
python scripts/audit_release.py
python scripts/package_web.py --output-dir dist
```

Iz same mape `ByFTP WEB/` dostupne su i niže razine provjere:

```bash
find . -name '*.php' -print0 | xargs -0 -n1 php -l
node --check assets/js/api.js
node --check assets/js/utils.js
node --check assets/js/settings.js
node --check assets/js/pwa.js
node --check assets/js/app.js
node --check service-worker.js
php tests/unit.php
php tests/json-store-bounds.php
php tests/ftp-listing.php
php tests/user-registry.php
php tests/profile-recovery.php
php tests/config-security.php
php tests/rate-limiter.php
```

Glavni ByFTP CI pokreće WEB runtime testove, repository-wide audit, sigurnosne/privacy provjere, deterministic WEB packaging regression i puni native platform matrix prije releasea.

## Ograničenja web modela

Browser/PHP shared-hosting aplikacija ne može imati isti neograničeni lokalni filesystem pristup niti trajni background socket kao native FileZilla/WinSCP tip klijenta. Folder upload koristi browser file/directory API, a veliki transferi ovise o PHP i hosting limitima. ByFTP WEB namjerno nema daemon, cron ni obvezni vanjski backend.
