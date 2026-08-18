# Povijest promjena

## 2.16.2 — Produkcijski release hardening, stvarno ugašena build telemetrija i detaljna dokumentacija

- ispravljena je build-privacy pogrešna pretpostavka: obična OS varijabla `GOTELEMETRY=off` nije pouzdan način za promjenu Go telemetry načina; CI i release sada stvarno izvršavaju `go telemetry off` i potvrđuju da rezultat glasi `off`
- `BUILD-WINDOWS.ps1`, `BUILD-LINUX.sh`, `BUILD-MACOS.sh` i `BUILD-LOCAL.sh` fail-closed odbijaju produkcijski build ako stvarni Go telemetry način nije `off`, umjesto da prikazuju netočan osjećaj zaštite
- Linux i macOS build skripte sada, jednako kao Windows build, zahtijevaju Go 1.26.5 ili noviji podržani sigurnosni patch
- uklonjen je release self-trigger race: publisher-created `v*` tag više ne pokreće drugi release workflow nad istim izdanjem
- svi release runovi koriste jednu serijaliziranu `byftp-release` concurrency grupu uz `cancel-in-progress: false`, pa dva publishera ne rade paralelno
- release workflow dobio je zaseban produkcijski quality job koji ponovno izvršava asset/hrvatski/version/docs/security/privacy/release audite, Python regresije, `go test`, `go test -race`, `go vet` i probno generiranje release bilješki prije javne objave
- `publish` sada ovisi o sva četiri release gatea: quality, Windows, Linux i macOS; javno izdanje više ne pretpostavlja da je raniji PR CI dovoljan dokaz
- Linux release job ponovno provjerava DEB package/version/architecture metapodatke, a macOS release job ponovno širi završni PKG s `pkgutil` i potvrđuje njegovu strukturu
- završni release staging prije generiranja zajedničkih metapodataka mora sadržavati točno 10 očekivanih platformskih paketa; dodatna, nedostajuća ili pogrešno imenovana datoteka zaustavlja objavu
- `BUILD-METADATA.txt` bilježi da je produkcijski release quality gate prošao, dok `SHA256.txt` i dalje pokriva svih 10 platformskih paketa te release notes/build metadata
- uklonjen je jednokratni v2.15 migracijski cleanup iz svakog budućeg releasea; produkcijska objava više ne ovisi nepotrebno o povijesnom tagu
- lokalni Windows build odvojio je javni `dist/` sadržaj od tehničkih build dokaza: Setup, Portable i checksum ostaju javni izlazi, dok interna komponenta uklanjanja i verification zapis završavaju u `dist/internal/`
- release audit sada blokira povratak tag-triggera, paralelnog publishera, jednokratnog legacy cleanupa, nedostatka production quality/race gatea i povratak zastarjelih javnih paketnih naziva u README/instalaciju/release bilješke
- privacy audit sada zaključava stvarno gašenje Go build telemetrije i odbija povratak na neučinkovitu istoimenu OS environment varijablu
- glavni README potpuno je proširen kao produkcijski ulaz: platforme, auth matrica, odabir paketa, instalacija/nadogradnja/OS uklanjanje, prvi spoj, connect semantika, transfer sigurnost, vjerodajnice, SHA-256, signing ograničenja, build i release integritet
- `INSTALACIJA.md`, `ARHITEKTURA.md`, `SIGURNOST.md`, `PRIVATNOST.md`, `TESTIRANJE.md`, `PROVJERA-IZDANJA.md`, `IZDAVANJE-NA-GITHUBU.md`, `PODRSKA.md`, `POTPISIVANJE.md`, `PLAN-RAZVOJA.md`, `DOPRINOS.md` i docs indeks usklađeni su s aktualnim produkcijskim modelom
- Windows korisnik za uklanjanje koristi standardni `Postavke → Aplikacije → Instalirane aplikacije → ByFTP` lifecycle; javna dokumentacija više ne upućuje korisnika na zasebnu tehničku release datoteku
- javni distribucijski ugovor ostaje 13 custom asseta: šest Windows paketa, tri Linux DEB paketa, jedan macOS Universal PKG te `SHA256.txt`, `RELEASE-NOTES.txt` i `BUILD-METADATA.txt`
- potpisivanje ostaje fail-closed: bez stvarnog Brendigo Authenticode odnosno Apple Developer ID identiteta release ne tvrdi Verified Publisher, notarizaciju ili drugi fabricirani publisher status

## 2.16.1 — Process-level connect provjera i stabilnija platformna matrica

- dodani su process-level FTP/FTPS i SFTP connect smoke testovi koji stvarno pokreću child proces preko istog `exec.CommandContext`, stdin/config i parser puta koji koristi produkcijski adapter
- FTP/FTPS smoke potvrđuje da unesena lozinka prolazi kroz kratkotrajni runtime-secret token, stiže u curl konfiguraciju preko standardnog ulaza, da se MLSD odgovor stvarno parsira i da `Close()` uklanja aktivnu tajnu
- SFTP smoke potvrđuje da proces ne dobiva `-b`, da `BatchMode=no` ostaje aktivan, da `ls -la` naredba stvarno stiže kroz stdin te da se povratni listing parsira kroz produkcijski parser
- Linux CI prije izrade amd64, arm64 i i386 DEB paketa sada izvršava `go test ./...` i `go vet ./...` na stvarnom Linux runneru
- macOS CI prije izrade Universal Intel+Apple Silicon PKG-a sada izvršava `go test ./...` i `go vet ./...` na stvarnom macOS runneru
- `audit_security.py` zahtijeva process-level connect smoke regresije pa budući refaktor ne može zadržati samo površinski unit test i ukloniti dokaz stvarnog child-process puta
- dokumentacija sada eksplicitno objašnjava da Windows stanje `POVEZANO` nastaje tek nakon uspješne autentikacije i stvarnog udaljenog `List` probea, a ne nakon samog pokretanja curl/OpenSSH procesa
- 2.16.1 zadržava javni release ugovor 2.16 linije: Windows x64/x86 Setup, Portable i ZIP; Linux amd64/arm64/i386 DEB; macOS Universal PKG; `SHA256.txt`, `RELEASE-NOTES.txt` i `BUILD-METADATA.txt`
- `verification.txt`, dodatni `ByFTP-<verzija>-Source.zip` i standalone `Uninstall-*.exe` nisu javni ByFTP release asseti; interni verifier i uninstaller ostaju u CI/Setup sloju gdje su potrebni

## 2.16.0 — Pouzdanije povezivanje, Windows x86, Linux i macOS

- ispravljen je stvarni SFTP authentication bug: uklonjen je `sftp -b`, jer aktualni OpenSSH pri obradi `-b` prisilno uključuje `BatchMode=yes` i time može onemogućiti password/passphrase AskPass autentikaciju
- SFTP naredbe i dalje idu kroz standardni ulaz bez vidljive konzole, ali `BatchMode=no` ostaje eksplicitno postavljen i više ga ne poništava kasnija CLI opcija
- postojeći pogrešni regresijski test koji je tvrdio da je kombinacija `BatchMode=no` + `-b` kompatibilna s AskPassom zamijenjen je testom koji zabranjuje povratak `-b`
- Windows unesena lozinka i passphrase ostaju u zaključanim kontrolama do stvarno uspješnog spajanja; mrežna/port/auth greška više ne briše korisnikov unos prije mogućeg ponovnog pokušaja
- dvostupanjska SFTP trust potvrda može ponovno koristiti upravo unesenu vjerodajnicu, dok se osjetljivi UI unos briše tek nakon potvrđenog `Connected` stanja
- AskPass je fail-closed i automatski daje tajnu samo jasno prepoznatom `password` ili `passphrase` promptu; MFA/OTP/security-key i nepoznati promptovi ne dobivaju spremljenu tajnu
- curl, ssh-keyscan, ssh-keygen i sftp timeout/cancel putovi vraćaju stvarni `context` uzrok pa korisničko sučelje može razlikovati timeout, otkazivanje, odbijeni port i autentikacijsku pogrešku
- bracketirani IPv6 hostovi normaliziraju se prije OpenSSH `HostName` i `ssh-keyscan` upotrebe
- `findOpenSSH` na Linuxu/macOS-u koristi nativne executable nazive bez `.exe` nastavka
- uvedeno je procesno memorijsko runtime spremište aktivnih ne-Windows tajni s kriptografski nasumičnim tokenom i wipe-on-close ponašanjem; aktivna FTP/FTPS lozinka ne zapisuje se na disk
- Windows aktivne runtime tajne i dalje koriste DPAPI
- ne-Windows desktop stub zamijenjen je stvarno funkcionalnim terminalnim ByFTP klijentom koji koristi isti `api.Engine`, remote adaptere i transfer queue kao Windows aplikacija
- Linux/macOS terminal podržava remote `ls`, `cd`, `mkdir`, `rename`, `delete`, `chmod`, `get`, `put`, `pwd`, host-key potvrdu i zajedničke transfer sigurnosne granice
- Linux/macOS FTP/FTPS podržava password autentikaciju; SFTP u 2.16.0 namjerno podržava eksplicitni privatni ključ bez passphrasea dok Unix AskPass broker nije sigurnosno dovršen
- Windows produkcijski build sada proizvodi i provjerava x64 i x86 Setup/Portable binarije te PE32/PE32+ resurse, manifest i mitigacije
- dodan je `scripts/BUILD-LINUX.sh` koji proizvodi DEB pakete za amd64, arm64 i i386
- dodan je `scripts/BUILD-MACOS.sh` koji proizvodi Universal Intel+Apple Silicon PKG i Finder `ByFTP.app` terminal launcher
- CI sada ima odvojene quality/race, Windows x64+x86, Linux DEB i macOS Universal PKG gateove
- javni GitHub Release više ne objavljuje custom `verification.txt`, `ByFTP-<verzija>-Source.zip` ni standalone `Uninstall-*.exe`; Windows uninstaller ostaje interni dio Setup paketa
- release workflow pri 2.16.0 objavi uklanja navedena tri custom asseta i iz postojećeg v2.15.0 izdanja te završno potvrđuje njihovu odsutnost
- GitHubovi automatski `Source code (zip)` i `Source code (tar.gz)` linkovi ostaju jer ih platforma generira za svaki tag
- novi 2.16.0 javni release ugovor sadrži 6 Windows paketa, 3 Linux DEB paketa, 1 macOS Universal PKG te `SHA256.txt`, `RELEASE-NOTES.txt` i `BUILD-METADATA.txt`
- `audit_security.py`, `audit_release.py`, bundle verifier i release regresije prošireni su kako bi navedene connect, multi-platform i public-asset granice postale obavezne CI invarijante

## 2.15.0 — Sigurniji profili, credential binding i kontrola host-key pina

- uveden je zajednički `internal/profilebinding` modul za jedan konzistentan endpoint/account/private-key identity ugovor kroz remote, config i Windows UI sloj
- spremljena lozinka profila automatski se nasljeđuje samo za isti protokol, host, port i korisničko ime; privremena promjena endpointa ili računa više ne može poslati staru lozinku drugom odredištu
- spremljeni passphrase koristi se samo za isti endpoint, korisnika i privatni ključ; promjena ili brisanje ključa blokira nasljeđivanje stare zaporke ključa
- prazna putanja privatnog ključa u aktivnom UI profilu postala je autoritativna i više ne vraća potajno staru spremljenu putanju
- profilni SFTP host-key fingerprint koristi se samo na istom protokolu, hostu i portu; privremeni endpoint ne može naslijediti stari pin niti svoj prihvaćeni pin zapisati natrag u originalni profil
- uređivanje istog SFTP endpointa čuva postojeći fingerprint, dok promjena hosta, porta ili protokola resetira pin i zahtijeva novu trust potvrdu
- spremanje profila automatski uklanja password/passphrase blobove koji više ne pripadaju novom endpointu, korisniku ili ključu
- SFTP passphrase više se ne čuva kada profil nema privatni ključ, čime se uklanja mrtva osjetljiva vrijednost iz spremišta
- Windows UI sada eksplicitno nudi zadržavanje ili uklanjanje postojećih spremljenih vjerodajnica i jasno objašnjava automatsko uklanjanje tajni nakon promjene identiteta profila
- oznake polja za spremljenu lozinku/passphrase osvježavaju se odmah nakon spremanja ili brisanja profila bez prikazivanja stvarne tajne
- privremeno promijenjeni endpoint više ne preuzima lokalnu i udaljenu početnu putanju starog profila
- `UpdateFingerprint` dodatno validira SHA-256 format i dopušta pin samo SFTP profilu
- dodane su regresije za host normalizaciju, account/key binding, čišćenje password/passphrase blobova, očuvanje/reset fingerprinta i autoritativno brisanje privatnog ključa
- `audit_security.py` sada blokira regresiju svih profilnih identity, credential i host-key pin granica

## 2.14.5 — Ograničeno sigurno zatvaranje veze i pouzdaniji reconnect

- ispravljen je lifecycle regresijski rizik iz 2.14.4: `remote.Manager.Disconnect` više ne može neograničeno blokirati pozivatelja na `activeOps.Wait()`
- remote disconnect sada prima `context.Context` i poštuje postojeći UI/shutdown deadline kroz cijeli engine stack
- istekom deadlinea adapter se namjerno ne zatvara ispod aktivne operacije; odvojeni cleanup čeka posljednji `Operation.release()` i tek tada poziva `Session.Close()`
- uveden je eksplicitni close-state pa je reconnect blokiran dok se prethodna curl/OpenSSH sesija još sigurno zatvara
- drugi `Disconnect` tijekom odgođenog čišćenja čeka isti close-state umjesto stvaranja drugog zatvaranja ili dvostrukog `Close()` poziva
- `ErrDisconnectTimeout` i `ErrSessionClosing` razlikuju lokalni session lifecycle od stvarnog mrežnog timeouta
- korisničke poruke sada jasno objašnjavaju kada se stara sesija još zatvara umjesto netočne poruke da poslužitelj nije odgovorio
- dodane su regresije za normalni disconnect, timeout/deferred close, reconnect blokadu, drugi disconnect i idempotentni release
- `audit_security.py` sada zahtijeva bounded disconnect, deferred session cleanup, engine context propagaciju i pripadajuće regresije
- README, sigurnosni model i testna dokumentacija usklađeni su s 2.14.5 ponašanjem

## 2.14.4 — Stabilniji udaljeni prikaz, SFTP ključevi i lifecycle veze

- ispravljen je Unix FTP/SFTP fallback parser: niz ` -> ` uklanja se samo iz stvarnog symlink zapisa pa obična datoteka s tim nizom u nazivu više nije pogrešno skraćena
- veličine iz tekstualnih listinga sada se parsiraju preko `strconv.ParseInt`; neispravna ili prevelika vrijednost pada na sigurnu nulu umjesto mogućeg `int64` wraparounda
- DOS/Windows-style listing sada pravilno prenosi veličinu obične datoteke u `model.Item`
- uklonjeno je duplicirano remote sortiranje s ponavljanim `strings.ToLower` pozivima po usporedbi; udaljeni prikaz koristi zajednički optimizirani `internal/itemlist.Sort`
- dodane su regresije parsera za regularni naziv s ` -> `, symlink, overflow veličine, DOS veličinu i puni limit od 50.000 udaljenih stavki
- SFTP privatni ključ sada mora biti obična datoteka bez symlinka i Windows reparse-point preusmjeravanja
- remote session lifecycle dobio je reference count aktivnih operacija: disconnect prvo zatvara prijem novih poziva i otkazuje session context, čeka svaki postojeći `Operation` release, a tek zatim zatvara curl/OpenSSH adapter
- `Operation` release postao je idempotentan kroz `sync.Once`, čime dvostruko čišćenje ne može spustiti active-operation brojač ispod nule
- dodane su determinističke regresije koje potvrđuju da disconnect ne zatvara adapter prije završetka aktivnog poziva i da dvostruki release ostaje siguran
- `audit_security.py` sada blokira regresiju private-key reparse zaštite i active-operation/session-close granice
- README, arhitektura, sigurnosna dokumentacija i testni katalog usklađeni su s novim 2.14.4 ponašanjem

## 2.14.3 — Fail-closed izdavanje i dokumentacija kao provjerena komponenta

- GitHub Release objava izdvojena je u `scripts/publish_release.ps1` i postala sigurno ponovljiva nakon djelomičnog pada
- postojeći release tag mora se razriješiti na točan release commit; mismatch odmah zaustavlja objavu
- postojeći asseti uspoređuju se po nazivu, veličini i GitHub SHA-256 digestu; potvrđeni asset se ne prepisuje, nedostajući se dopunjuje, a različit sadržaj zaustavlja izdanje
- uklonjen je zastarjeli hardkodirani `workflow_dispatch` default/primjer verzije; prazni ručni unos ponovno koristi kanonski `VERSION`
- konačni Windows ZIP nakon `Compress-Archive` prolazi novu `verify_bundle.py` provjeru putanja, duplikata, obaveznih datoteka, potpunosti `BUNDLE-SHA256.txt` manifesta i svakog SHA-256
- dodane su stdlib Python regresije za valjani bundle, traversal, nepopisanu datoteku i integritetsku pogrešku
- uvedeni su zasebni `audit_docs.py`, `audit_security.py` i `audit_release.py` quality gateovi
- hrvatski audit sada dinamički čita `VERSION` umjesto ručnog hardkodiranja aktualnog broja
- version audit blokira povratak hardkodiranog release defaulta i aktualne verzije u bug predložak/audit
- CI, `BUILD-WINDOWS.ps1` i `BUILD-LOCAL.sh` pokreću isti prošireni skup docs/security/privacy/release provjera i Python regresija
- Windows ZIP i GitHub Package sada uključuju kompletnu `docs/*.md` dokumentaciju, a paket uključuje i release bilješke
- detaljni dokumenti postali su verzijski neutralni, prošireni stvarnim 2.14.2/2.14.3 granicama i automatski se provjeravaju za pokvarene lokalne poveznice i nepotpun indeks
- ispravljene su povijesne tipografske greške u CHANGELOG-u

## 2.14.2 — Točniji završni statusi i čvršći lokalni staging

- završni rezultat transfer adaptera postao je autoritativan: kasno otkazivanje ili prekid veze nakon stvarnog uspjeha više ne prepisuje posao u status `cancelled`
- `ErrSkipped` ostaje `skipped` čak i ako kontekst bude otkazan neposredno nakon odluke da se postojeća datoteka preskoči
- FTP/FTPS i SFTP download staging datoteke provjeravaju se s `Lstat`, moraju biti obične regularne datoteke i ne smiju biti Windows reparse pointovi
- atomska aktivacija preuzete datoteke dodatno odbija reparse-point cilj prije backup/rollback preimenovanja
- `RemoveTreeNoFollow` eksplicitno blokira korijen datotečnog sustava, uključujući Windows drive i UNC root putanje, kao vlastitu sigurnosnu granicu
- dodani su regresijski testovi za kasni cancel, skipped status, stvarno otkazivanje, staging symlink i filesystem-root zaštitu
- puni unit/race/vet, privatnost, hrvatski sadržaj, verzijski audit i Windows produkcijski build ostaju obavezni prije objave

## 2.14.1 — Završna konzistentnost verzije i službenog izvora

- `cmd/byftp` i `cmd/installer` koriste jasni `dev` fallback pri razvojnim buildovima umjesto zastarjele hardkodirane produkcijske verzije
- dodan je `scripts/audit_version.py` koji provjerava `VERSION`, README, CHANGELOG i oba build puta te blokira buduće razilaženje broja verzije
- CI, Windows build i lokalni build pokreću istu provjeru verzijske konzistentnosti
- službeni 2.14.1 release ponovno pakira završno stanje izvornog koda tako da `Source.zip` odgovara konačnom `main` stanju ove patch verzije
- funkcionalnost transfera i sigurnosni model ostaju kompatibilni s 2.14.0; ovo je završni patch za source/build/release konzistentnost

## 2.14.0 — Hrvatski projekt, uređeni repozitorij, event izolacija i obnovljeni resursi

- cijelo korisničko sučelje, GitHub predlošci, release bilješke, paketna dokumentacija i glavna projektna dokumentacija standardizirani su na hrvatski jezik
- detaljna dokumentacija premještena je iz root direktorija u uređenu mapu `docs/` s jednim indeksom i hrvatskim nazivima dokumenata
- dodan je `scripts/audit_croatian.py` koji u CI-ju sprječava povratak poznatih engleskih korisničkih/release površina
- `internal/transfer.Manager.Events` sada vraća duboke kopije `Job` pokazivača i `Jobs` sliceova pa pozivatelj ne može izmijeniti internu event povijest mutiranjem vraćene vrijednosti
- dodan je regresijski test koji namjerno mijenja vraćene evente i potvrđuje da spremnik ostaje nepromijenjen
- privremeni DPAPI-zaštićeni SFTP trust podaci automatski se brišu nakon završene potvrde, greške, otkaza ili uspješnog spajanja; zadržavaju se samo dok korisnik stvarno potvrđuje novi ključ
- uklonjen je nedostižni cleanup stare sesije iz `Connect` puta jer postojeća veza već ranije blokira novo spajanje
- `scripts/BUILD-LOCAL.sh` više nema zastarjelu hardkodiranu verziju 2.12.0 nego čita jedini kanonski `VERSION`
- lokalni build sada također prisilno isključuje Go telemetriju i provjerava hrvatski sadržaj i brand resurse
- zamijenjen je oštećeni `build/icon.png` iz 2.13.0; stari PNG imao je neispravan IDAT CRC/bitstream
- dodan je reproducibilni generator `scripts/generate_brand_assets.py` bez vanjskih Python ovisnosti koji izrađuje 512×512 PNG, višerezolucijski ICO i dokumentacijsko zaglavlje
- CI i produkcijski build provjeravaju da su PNG/ICO resursi točni, sinkronizirani i generirani iz istog izvora
- Windows release ZIP dobiva uređenu podmapu dokumentacije, a bundle SHA-256 pokriva rekurzivno sve datoteke uključujući podmape
- GitHub Actions nazivi koraka, release paketni opis i javne release bilješke prevedeni su na hrvatski
- FTPS oznake u UI-u sada glase `FTPS (eksplicitni)` i `FTPS (implicitni)`, a polje poslužitelja više ne koristi englesku riječ „Host”

## 2.13.0 — Sigurnije stanje, veliki redovi i podrijetlo izdanja

- state/config čitanje provjerava da je stvarno otvoren isti stabilni regularni objekt koji je prethodno validiran
- dodane su regresije za symlink i regular-file zamjenu između validacije i otvaranja
- desktop obrađuje batch transfer događaja preko ID indeksa umjesto punog scaniranja reda za svaki event
- `Očisti završene` uklanja i stale UI ID-eve završetka
- veliko lokalno sortiranje unaprijed računa case-folded ključeve jednom po stavci
- dodane su regresije s 50.000 stavki te 20.000 poslova + 1.000 eventa
- release bilješke generiraju se iz odgovarajućeg CHANGELOG odjeljka
- uveden je `BUILD-METADATA.txt` s commit/ref/toolchain podacima
- popravljeno je verzioniranje kompletnog GitHub Actions release artefakta

## 2.12.1 — Učvršćivanje korijena rekurzivnog slanja i kompletno pakiranje

- datoteke rekurzivnog uploada koriste roditelja odabranog korijena kao sigurnosnu granicu, pa je i sam korijen dio runtime provjere
- kasna symlink/junction/reparse zamjena upload korijena blokira se prije izvršenja queued posla
- release automatizacija proizvodi Setup, Portable, Uninstaller, Windows ZIP, Source ZIP, checksumove i verifikacijske datoteke

## 2.12.0 — Sigurnije ponavljanje, preskakanje postojećih i timeout veze

- dodana je politika preskakanja postojećih datoteka
- dodano je podesivo vrijeme čekanja veze od 5 do 60 sekundi za curl/OpenSSH
- automatsko ponavljanje ograničeno je na prolazne mrežne/timeout/partial greške
- queued prijenosi ponovno validiraju dopušteni lokalni korijen prije svakog pokušaja
- poboljšano je stvaranje ByFTP data/install/SFTP direktorija bez redirecta
- paralelizam se može mijenjati uživo
- poboljšani su settings cache, directory cache i `Clear Finished` cleanup

## 2.11.0 — Tipizirani engine, skupne operacije i retry sigurnost

- uklonjen je generički in-process JSON dispatcher i uveden tipizirani engine API
- batch cancel/retry prvo validira cijeli odabir
- transfer posao veže se uz opaque identitet aktivne veze i ne može se retryati na drugi server/account
- superseded directory refresh se aktivno otkazuje
- poboljšan je FTP MLSD fallback i hot-path ponašanje postavki

## 2.10.0 — Izolacija radnika, junction zaštita i transakcijske datoteke

- panic jednog transfer radnika više ne ruši aplikaciju
- blokiran je Windows junction/reparse traversal u direktnom i rekurzivnom radu
- staging nazivi postali su kriptografski nasumični
- rename/replace putovi dobili su no-replace zaštitu i bolji rollback
- current-directory DLL search uklonjen je iz Windows procesa

## 2.9.0 — Privatnost SFTP metapodataka i reconnect izolacija

- host, korisnik i privatni ključ uklonjeni su iz OpenSSH command line metapodataka
- session SSH config koristi nasumični alias i kratki lifecycle
- transfer batch rezervacije vežu se uz connection generation
- putanje su ujednačene između gumba i dvoklika

## 2.8.0 — Privatnost po zadanim postavkama i kriptirani profili

- uklonjen je trajni runtime activity/error log
- Windows Error Reporting isključen je za ByFTP proces
- profile metadata i credential blobovi zaštićeni su DPAPI envelopeom
- legacy plaintext profili automatski se migriraju i uklanjaju
- file picker privatnog ključa ne dodaje ključ u Windows Recent listu

## 2.7.0 — Windows lifecycle, integritet instalacije i učvršćene sesije

- Known Folder/System Directory API-ji zamijenili su oslanjanje na nepouzdane env putanje
- installer payload dobio je size + SHA-256 manifest provjeru
- nadogradnja dobiva rollback datoteka i Registry stanja
- PE verifikacija razlikuje i provjerava sva tri binarija

## 2.6.0 — Produkcijska privatnost i sigurnija preuzimanja

- Windows curl/OpenSSH biraju se samo iz stvarnog System32 direktorija
- blokirani su proxy/TLS/SSH helper i agent inheritance putovi
- lokalni download dobio je zaštitu od Windows rezerviranih naziva i traversal escapea
- uvedeni su transakcijski staging/rollback prijenosi

## 2.5.0 — Fluent UI i dubinska provjera pouzdanosti

- glavni desktop gumbi prešli su na centralizirane Windows Fluent/MDL2 glyphove
- poboljšani su DPI, tamna tema, file/folder prikaz i resursi
- uklonjeni su Windows `unsafe.Pointer` vet problemi i poboljšan installer feedback

## 2.4.0 — Pouzdanost i SFTP učvršćivanje

- poboljšana je validacija SFTP host ključa i session cleanup
- poboljšane su greške, timeouti i mrežna izolacija

## 2.3.0 — Brending i desktop dorade

- uveden je dosljedni ByFTP/Brendigo identitet, verzijski resursi i aplikacijska ikona
- poboljšani su statusi, dijalozi i vizualna dosljednost

## 2.2.0 — Tamni desktop i upravitelj datotekama

- dodan je dvopanelni tamni file manager, lokalne/udaljene operacije i transfer queue prikaz

## 2.1.0 — Izvorni desktop

- ByFTP prelazi na izvorni Win32 desktop proces bez browser/localhost sučelja

## 2.0.0 — Nova generacija ByFTP-a

- postavljena je nova Go/Win32 arhitektura za FTP, FTPS i SFTP klijent usmjeren na privatnost
