# ByFTP — sigurnost

## Vjerodajnice

- spremljene lozinke i zaporke privatnog ključa koriste Windows DPAPI
- plaintext lozinka/passphrase ne ulazi u command-line argumente
- AskPass ne sprema credential datoteku; DPAPI blob predaje se kratkoživućem child procesu kroz sanitizirani environment
- AskPass prihvaća samo vlastiti ByFTP executable, jednokratni token i očekivani System32 OpenSSH parent proces
- spremljeni credential blob ne dešifrira se rano u connection manageru
- spremljena profilna lozinka koristi se automatski samo za isti protokol, host, port i korisničko ime
- spremljeni passphrase dodatno zahtijeva isti privatni ključ
- promjena endpointa, korisnika ili ključa pri spremanju profila automatski uklanja stare blobove koji više ne pripadaju novom identitetu
- uklanjanje privatnog ključa automatski uklanja spremljeni passphrase; profil ne zadržava mrtvu tajnu
- Windows UI omogućuje eksplicitno zadržavanje ili uklanjanje postojećih spremljenih vjerodajnica bez prikazivanja stvarnih vrijednosti
- SFTP trust credential je DPAPI-zaštićen, vezan uz točan host/port/user/key/fingerprint i vremenski ograničen
- privremeni trust blob briše se nakon svake završene trust sekvence, uključujući grešku ili otkaz
- privatni SFTP ključ mora biti regularna lokalna datoteka; symlink i Windows reparse-point objekt se odbijaju

## Profilni identitet i host-key pin

`internal/profilebinding` je zajednička sigurnosna granica za remote, config i desktop sloj. Endpoint identitet je `protokol + normalizirani host + port`; account identitet dodaje točno korisničko ime; private-key identitet dodatno veže lokalnu Windows putanju ključa.

Privremena izmjena hosta/porta/korisnika u odabranom profilu ne smije naslijediti spremljenu lozinku drugog account identiteta. Promjena ili brisanje privatnog ključa ne smije naslijediti stari passphrase. Prazna key putanja u aktualnom UI-u je autoritativna i ne smije biti zamijenjena starom profilnom vrijednošću.

SFTP host-key fingerprint pripada samo endpointu. Spremljeni pin koristi se samo za isti protokol, host i port. Privremeno izmijenjeni endpoint može dobiti potvrdu za svoju sesiju, ali ne može prepisati pin originalnog profila. Obično uređivanje istog endpointa čuva postojeći pin; promjena hosta, porta ili protokola automatski ga resetira i zahtijeva novu trust potvrdu.

## Mrežni procesi

- Windows build koristi stvarni System32 `curl.exe` i OpenSSH otkriven preko sistemskih API-ja
- nema proizvoljnog PATH/WINDIR fallbacka za mrežne alate
- curl ne učitava `.curlrc`, ne nasljeđuje proxy/TLS override varijable i dobiva izričiti no-proxy
- SFTP blokira ProxyCommand, ProxyJump, agent, PKCS#11/security-key provider, KnownHostsCommand, PermitLocalCommand i forwarding
- bez izričito odabranog ključa postavljaju se `IdentitiesOnly=yes` i `IdentityFile=none`
- host ključ provjerava se kroz session-temporary ByFTP known_hosts i sesija se veže uz potvrđeni algoritam
- host/user/private-key metapodaci nisu na OpenSSH command lineu

## Životni ciklus udaljene sesije

Svaka uspješna remote `Operation` registracija drži aktivnu referencu na sesiju do poziva idempotentnog `release()`. Disconnect pod ekskluzivnim lockom najprije uklanja sesiju iz managera, čime blokira nove operacije, zatim otkazuje session context. Zaseban cleanup put čeka postojeće reference i tek kada svi aktivni pozivi završe smije pozvati `Session.Close()`.

Ova granica sprječava da se curl/OpenSSH adapter zatvori, osjetljivi session state očisti ili SFTP config/known_hosts datoteke izbrišu dok ih paralelni list/rename/chmod/transfer poziv još koristi.

`Disconnect(ctx)` istodobno je bounded: UI i shutdown deadline mogu vratiti kontrolu pozivatelju prije završetka cleanup-a. Timeout namjerno ne poziva prisilni `Close()` nad adapterom koji je još u uporabi. Manager zadržava jedan close-state, čeka zadnji `release()` i potom zatvara adapter. Dok close-state traje, novi `Connect` vraća `ErrSessionClosing`, a ponovljeni `Disconnect` čeka isti state umjesto dvostrukog zatvaranja.

`ErrDisconnectTimeout` je odvojen od mrežnog `context.DeadlineExceeded`, pa korisničko sučelje ne prikazuje netočnu poruku da poslužitelj nije odgovorio kada se zapravo lokalno dovršava sigurno zatvaranje stare sesije.

## Datoteke i putanje

- lokalni i udaljeni nazivi imaju traversal/control-character validaciju
- server-controlled lokalni naziv prolazi Windows rezervirane nazive i sigurnu child-path provjeru
- download ne smije izaći kroz nested symlink/junction
- upload ne prati lokalne symlinkove
- rekurzivno brisanje ne prolazi kroz symlink/reparse/junction točke
- `RemoveTreeNoFollow` ima depth/item limite i samostalno blokira filesystem root, uključujući Windows drive i UNC root
- atomski upload/download koristi privremenu datoteku i rollback prije zamjene originala
- download `.byftp-part-*` staging objekt prije aktivacije prolazi `Lstat`, regular-file i Windows reparse-point provjeru
- ciljni lokalni replace odbija nepouzdani reparse objekt i ne prepisuje postojeću stavku kroz check-then-rename utrku
- temp datoteke imaju nepredvidive nazive i stvaraju se ekskluzivno
- rekurzivni upload root ponovno se validira prije svakog queued pokušaja

## Udaljeni listing

Tekstualni fallback listing tretira ` -> ` kao symlink separator samo ako permissions/type polje stvarno označava simboličku poveznicu. Time ime obične datoteke ostaje netaknuto. Veličina se pretvara provjerenim `strconv.ParseInt` pozivom; prevelik, negativan ili neispravan broj ne može wrapati `int64`.

FTP MLSD ostaje preferirani strojno čitljivi format. Udaljeni i lokalni prikaz dijele isti bounded/stable sorter, testiran na 50.000 stavki.

## State/config

State safe-open odbija ne-regularni objekt i provjerava identitet/stabilnost stvarno otvorene datoteke. Ako current zapis nije siguran ili valjan, store koristi provjerenu prethodnu generaciju ili zadane vrijednosti.

## Transfer izolacija

`Events` vraća duboke kopije event struktura. Pozivatelj ne može mutirati pokazivač/slice spremljen u event povijesti i tako utjecati na kasnije event odgovore.

Transfer posao pamti identitet veze i ne može se ručno retryati na drugi server/account. Queued posao ponovno validira lokalni root pri svakom pokušaju. Završni status koristi stvarni rezultat adaptera pa kasni cancel nakon uspješnog/preskočenog transfera ne može lažno promijeniti rezultat u `cancelled`.

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

- CI izvršava unit, race, vet, privacy, security, hrvatski-content, docs, version, release i asset audit
- Windows build provjerava PE32+ GUI strukturu, resurse i sigurnosne mitigacije
- Setup, Portable i Uninstaller moraju biti različiti binariji
- SHA-256 se objavljuje uz release
- Windows ZIP ima zaseban rekurzivni `BUNDLE-SHA256.txt`
- konačni ZIP ponovno se čita i hashira nakon kompresije; ne raspakirava se na filesystem tijekom provjere
- build metadata veže izdanje uz commit/ref/toolchain bez korisničkih podataka
- Source ZIP dolazi iz točnog `git archive HEAD`
- postojeći release tag mora pokazivati na očekivani commit
- rerun releasea smije dopuniti samo nedostajuće assete
- postojeći asset s drugom veličinom ili SHA-256 digestom zaustavlja izdanje umjesto automatskog prepisivanja

## Automatizirani sigurnosni gate

`scripts/audit_security.py` čuva ključne source invarijante i prisutnost odgovarajućih regresijskih testova. Izričito čuva profile endpoint/account/private-key binding, pin scope, autoritativno brisanje privatnog ključa, credential cleanup pri promjeni identiteta, private-key reparse blokadu, active-operation/session-close lifecycle, bounded disconnect timeout, deferred cleanup i engine propagaciju lifecycle konteksta. `scripts/audit_release.py` zasebno čuva release/tag/asset/bundle ugovor.

## Ograničenje

Automatizirana provjera ne zamjenjuje Authenticode potpis i runtime smoke-test na stvarnom Windows 10/11 sustavu sa stvarnim FTP/FTPS/SFTP poslužiteljima. Produkcijski Brendigo certifikat ne smije biti pohranjen u repozitoriju.
