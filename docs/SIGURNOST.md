# ByFTP 2.14.0 — sigurnost

## Vjerodajnice

- spremljene lozinke i zaporke privatnog ključa koriste Windows DPAPI
- plaintext lozinka/passphrase ne ulazi u command-line argumente
- AskPass ne sprema credential datoteku; DPAPI blob predaje se kratkoživućem child procesu kroz sanitizirani environment
- AskPass prihvaća samo vlastiti ByFTP executable, jednokratni token i očekivani System32 OpenSSH parent proces
- spremljeni credential blob ne dešifrira se rano u connection manageru
- SFTP trust credential je DPAPI-zaštićen, vezan uz točan host/port/user/key/fingerprint i vremenski ograničen
- od 2.14 privremeni trust blob briše se nakon svake završene trust sekvence, uključujući grešku ili otkaz

## Mrežni procesi

- Windows build koristi stvarni System32 `curl.exe` i OpenSSH otkriven preko sistemskih API-ja
- nema proizvoljnog PATH/WINDIR fallbacka za mrežne alate
- curl ne učitava `.curlrc`, ne nasljeđuje proxy/TLS override varijable i dobiva izričiti no-proxy
- SFTP blokira ProxyCommand, ProxyJump, agent, PKCS#11/security-key provider, KnownHostsCommand, PermitLocalCommand i forwarding
- bez izričito odabranog ključa postavljaju se `IdentitiesOnly=yes` i `IdentityFile=none`
- host ključ provjerava se kroz session-temporary ByFTP known_hosts i sesija se veže uz potvrđeni algoritam
- host/user/private-key metapodaci nisu na OpenSSH command lineu

## Datoteke i putanje

- lokalni i udaljeni nazivi imaju traversal/control-character validaciju
- server-controlled lokalni naziv prolazi Windows rezervirane nazive i sigurnu child-path provjeru
- download ne smije izaći kroz nested symlink/junction
- upload ne prati lokalne symlinkove
- rekurzivno brisanje ne prolazi kroz symlink/reparse/junction točke
- atomski upload/download koristi privremenu datoteku i rollback prije zamjene originala
- direktorij/symlink ne prepisuje se običnom datotekom
- temp datoteke imaju nepredvidive nazive i stvaraju se ekskluzivno
- rekurzivni upload root ponovno se validira prije svakog queued pokušaja

## State/config

State safe-open odbija ne-regularni objekt i provjerava identitet/stabilnost stvarno otvorene datoteke. Ako current zapis nije siguran ili valjan, store koristi provjerenu prethodnu generaciju ili zadane vrijednosti.

## Transfer event izolacija

Do 2.13 poziv `Events` vraćao je plitke kopije event struktura. Iako se time nije mijenjao glavni `jobs` slice, pozivatelj je mogao mutirati pokazivač/slice spremljen u event povijesti i tako utjecati na kasnije event odgovore. U 2.14 svaki `Event.Job` i `Event.Jobs` izlaz duboko se kopira prije izlaganja pozivatelju i prije spremanja eventa.

## Installer i uninstaller

- payload ima manifest s veličinama i SHA-256 vrijednostima
- instalacija radi samo u kanonskoj per-user ByFTP putanji
- nadogradnja ima rollback datoteka i Registry vrijednosti
- postojeći executable symlink/reparse scenariji odbijaju se
- uninstaller se mora pokretati iz očekivane instalirane lokacije i ne briše proizvoljne putanje

## Procesne zaštite

- Windows Error Reporting onemogućen je za ByFTP proces
- current-directory DLL search uklonjen je prije GUI starta
- production GUI ne prikazuje razvojni Go stack trace korisniku
- nema trajnog runtime activity/error loga

## Release zaštite

- CI izvršava unit, race, vet, privacy, hrvatski-content i asset audit
- Windows build provjerava PE32+ GUI strukturu, resurse i sigurnosne mitigacije
- Setup, Portable i Uninstaller moraju biti različiti binariji
- SHA-256 se objavljuje uz release
- Windows ZIP ima zaseban rekurzivni `BUNDLE-SHA256.txt`
- build metadata veže izdanje uz commit/ref/toolchain bez korisničkih podataka

## Ograničenje

Automatizirana provjera ne zamjenjuje Authenticode potpis i runtime smoke-test na stvarnom Windows 10/11 sustavu sa stvarnim FTP/FTPS/SFTP poslužiteljima. Produkcijski Brendigo certifikat ne smije biti pohranjen u repozitoriju.
