# ByFTP — privatnost

ByFTP je projektiran da prenosi korisničke datoteke prema poslužitelju koji korisnik sam odabere, bez paralelnog slanja podataka Brendigu ili analitičkim servisima.

## ByFTP ne sadrži

- telemetriju
- analitiku korištenja
- oglašavanje
- third-party crash reporting
- cloud korisnički račun
- automatski update API
- skriveni HTTP/HTTPS runtime endpoint
- trajni activity/error log
- browser ili localhost upravljačko sučelje

## Mrežni promet

Mrežni adapteri razgovaraju s FTP/FTPS/SFTP odredištem koje korisnik odabere. Sistem može zasebno raditi DNS, firewall, antivirus/EDR, proxy politiku ili druge OS funkcije koje nisu dio ByFTP telemetrije.

curl se pokreće bez naslijeđenih proxy/TLS override postavki. OpenSSH se pokreće s ByFTP-managed konfiguracijom koja blokira helper routing, forwarding i globalne trust izvore koji bi mogli nenamjerno promijeniti odredište ili izložiti metapodatke.

## Lokalni podaci i profili

Profili i postavke ostaju u korisničkoj ByFTP data mapi. Osjetljivi profilni podaci zaštićeni su Windows DPAPI mehanizmom. ByFTP nema persistent runtime log koji bi bilježio servere, putanje, nazive datoteka ili greške iz svakodnevnog rada.

Spremljena lozinka ne koristi se automatski izvan istog protokola, hosta, porta i korisničkog imena za koje je profil spremljen. Spremljeni passphrase dodatno je vezan uz isti privatni ključ. Privremena izmjena odredišta ili korisnika stoga ne može nenamjerno poslati staru profilnu lozinku drugom serveru ili računu.

Kada se profil trajno promijeni na drugi endpoint, korisnika ili ključ, ByFTP uklanja stare credential blobove koji više ne pripadaju tom identitetu. Ako se privatni ključ ukloni, uklanja se i njegova spremljena zaporka. Novi identitet dobiva spremljenu tajnu samo nakon ponovnog izričitog unosa i korisničke odluke da je spremi.

Windows UI za uređivanje profila prikazuje samo činjenicu da spremljena vjerodajnica postoji; nikada ne vraća stvarnu lozinku/passphrase u edit kontrolu. Pri spremanju korisnik može zadržati ili ukloniti vjerodajnice koje još pripadaju istom identitetu. Promijenjeni identitet jasno se označava i stare vrijednosti se ne prenose.

Windows Error Reporting isključen je za ByFTP proces kako rušenje aplikacije ne bi pokrenulo automatsko slanje memorijskog izvještaja kroz WER.

State/config zapis koristi ograničeno, provjereno čitanje regularne datoteke i sigurnu prethodnu generaciju. ByFTP-owned direktoriji ne smiju biti preusmjereni symlinkom/junctionom ispod kanonske korisničke mape.

## SFTP

Host, korisnik i privatni ključ nisu na OpenSSH command lineu. Kratkoživa session konfiguracija koristi nasumični alias i uklanja se nakon sesije/startup cleanupa. AskPass dobiva samo DPAPI-zaštićene vrijednosti kroz child environment i ne stvara credential datoteku.

Spremljeni host-key fingerprint koristi se samo kada se protokol, host i port podudaraju sa spremljenim profilom. Privremeno promijenjeni endpoint može biti prihvaćen za svoju sesiju, ali njegov fingerprint ne prepisuje originalni profil. Promjena spremljenog endpointa resetira stari pin i traži novu potvrdu.

Pending trust vjerodajnice ostaju u memoriji samo između prikaza novog host ključa i korisničke odluke. Nakon potvrde, greške, otkaza ili uspješnog spajanja brišu se.

## Release i CI podaci

`BUILD-METADATA.txt` sadrži samo verziju, Git commit/ref, Go verziju, platformu i GitHub Actions run identifikatore. Ne uključuje profile, hostove, korisnička imena, lokalne putanje ni vjerodajnice.

Release bilješke nastaju iz `CHANGELOG.md`. Asset, hrvatski, dokumentacijski, security, privacy i release auditi čitaju samo datoteke repozitorija i ne dodaju runtime mrežne pozive aplikaciji.

Javni release rerun provjerava Git tag i digest već objavljenih asseta. To je build/release integritetska provjera; ne uvodi telemetriju u ByFTP runtime.

## Što ByFTP ne može kontrolirati

Odredišni poslužitelj nužno vidi podatke potrebne za uspostavu veze i izvršenje transfera. Windows, antivirus/EDR, firewall, backup/sync softver ili sam poslužitelj mogu imati vlastita pravila zapisivanja. Lokalno brisanje datoteke nije isto što i forenzičko brisanje fizičkih SSD blokova niti može povući kopiju koju je drugi backup sustav već napravio.
