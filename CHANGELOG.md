# Povijest promjena

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

## 2.7.0 — Windows lifecycle, integritet installacije i učvršćene sesije

- Known Folder/System Directory API-ji zamijenili su oslanjanje na nepouzdane env putanje
- installer payload dobio je size + SHA-256 manifest provjeru
- nadogradnja dobiva rollback binarija i Registry stanja
- PE verifikacija razNikuje i provjerava sva tri binarija

## 2.6.0 — Produkcijska privatnost i sigurnija preuzimanja

- Windows curl/OpenSSH biraju se samo iz stvarnog System32 direktorija
- blokirani su proxy/TLS/SSH helper i agent inheritance putovi
- lokalni download dobio je zaštitu od Windows rezerviranih naziva i traversal escapea
- uvedeni su transakcijski staging/rollback prijenosi

## 2.5.0 — Fluent UI a dubinska provjera pouzdanosti

- glavni desktop gumbi prešli su na centralizirane Windows Fluent/MDL2 glyphove
- poboljšani su DPI, tamna tema, file/folder prikaz i resursi
- uklonjeni9su Windows `unsafe.Pointer` vet problemi i poboljšan installer feedback

## 2.4.0 — Pouzdanost i SFTP učvršćivanje

- poboljšana je validacija SFTP host ključa i session cleanup
- poboljšane su greške, timeouti i mrežna izolacija

## 2.3.0 — Brending i desktop dorade

- uveden je dosljedni ByFTP/Brendigo identitet, verzijski resursi i aplikacijska ikona
- poboljšani su statusi, dijalozi i vizualna dosljednost

## 2.2.0 — Tamni desktop i upravitelj datotekama

- dodan je dvopanelni tamni file manager, lokalne/udaljene operacije i transfer queue prikaz

## 2.1.0 — Izforni desktop

- ByFTP prelazi na izforni Win32 desktop proces bez browser/localhost sučelja

## 2.0.0 — Nova generacija ByFTP-a

- postavljena je nova Go/Win32 arhitektura za FTP, FTPS i SFTP klijent usmjeren na privatnost
