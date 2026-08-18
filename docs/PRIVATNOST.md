# ByFTP — privatnost

ByFTP prenosi datoteke prema poslužitelju koji korisnik sam odabere. Runtime nema paralelno slanje Brendigu, analitičkim servisima ili oglašivačkim mrežama.

## ByFTP nema

- telemetriju i analitiku korištenja
- oglašavanje
- third-party crash reporting
- obavezni cloud račun
- skriveni update API
- browser/localhost upravljački server
- trajni runtime activity/error log

## Mrežni promet

ByFTP mrežni adapteri razgovaraju s korisnički odabranim FTP/FTPS/SFTP endpointom. Operativni sustav, DNS, firewall, antivirus/EDR ili sam poslužitelj mogu imati vlastito zapisivanje koje nije ByFTP telemetrija.

curl dobiva sanitizirano okruženje bez naslijeđenog proxy/TLS overridea. OpenSSH dobiva ByFTP-managed session config koji blokira ProxyCommand, ProxyJump, agent, PKCS#11 provider, KnownHostsCommand i forwarding.

## Windows podaci

Windows profili i postavke ostaju u ByFTP korisničkoj data mapi. Profilne tajne koriste DPAPI. UI prikazuje samo činjenicu da tajna postoji, ne vraća stvarnu spremljenu vrijednost u edit kontrolu.

Lozinka se automatski koristi samo za isti endpoint i korisnika; passphrase dodatno za isti privatni ključ. Promjena identiteta briše stale blob koji više ne pripada novom profilu.

Unesena lozinka/passphrase tijekom povezivanja ostaje samo u zaključanoj edit kontroli dok pokušaj traje i briše se nakon stvarno uspješnog povezivanja. To omogućuje retry bez ponovnog unosa, ali ne uvodi trajno spremanje.

## Linux i macOS

Terminalni frontend ne sprema profile ni terminalne vjerodajnice na disk.

FTP/FTPS aktivna lozinka pohranjuje se samo u procesu iza kriptografski nasumičnog runtime tokena. Adapter ne drži plaintext; kratkotrajna kopija nastaje neposredno prije curl poziva i briše se nakon uporabe. `Close()` uklanja i briše procesnu vrijednost.

SFTP password/passphrase na Linuxu/macOS-u trenutačno nije uključen. Umjesto privremenog slanja tajne kroz argument ili običan environment, terminalno izdanje fail-closed zahtijeva eksplicitni privatni ključ bez passphrasea dok se ne uvede siguran Unix AskPass broker.

## Connect smoke testovi i privatnost

2.16.1 process-level connect smoke testovi **nemaju vanjski mrežni promet**. Testovi stvaraju lokalni kratkotrajni fake `curl` ili `sftp` proces i koriste testnu vrijednost vjerodajnice. Time se provjerava produkcijski child-process/stdin/parser put bez kontaktiranja stvarnog servera, bez stvarnih korisničkih podataka i bez zapisivanja tajne u repozitorij.

FTP smoke dodatno potvrđuje da runtime token više nije moguće razriješiti nakon `Close()`. Testni executable i njegovi privremeni podaci nalaze se u testnom privremenom direktoriju koji runner uklanja nakon testa.

## SFTP metapodaci

Host, korisnik i private-key putanja nisu na OpenSSH command lineu. Session koristi nasumični lokalni alias, privatni `known_hosts` i kratkotrajnu config datoteku. Bracketirani IPv6 host normalizira se prije OpenSSH unosa.

Windows AskPass dobiva samo zaštićene vrijednosti kroz kontrolirani child environment i daje tajnu isključivo prepoznatom password/passphrase promptu. MFA/OTP/nepoznati prompt ne dobiva spremljenu tajnu.

Host-key fingerprint pripada endpointu. Privremeni endpoint ne može naslijediti ili prepisati pin drugog profila.

## Lokalno stanje

ByFTP nema trajni runtime log servera, putanja i transfer pogrešaka. State/config čitanje koristi bounded safe-open model i provjerenu prethodnu generaciju. ByFTP-owned direktoriji ne smiju biti preusmjereni ispod kanonske podatkovne lokacije.

Windows podatkovni korijen je stvarni LocalAppData. macOS koristi `~/Library/Application Support`, a Linux apsolutni `XDG_DATA_HOME` ili `~/.local/share`.

## Build i release podaci

`BUILD-METADATA.txt` sadrži samo verziju, release commit/ref, platforme i GitHub Actions identifikatore. Ne sadrži hostove, profile, lokalne putanje ni vjerodajnice.

`SHA256.txt` sadrži samo hash javnih paketa/metapodataka. Interni verification report nije javni release asset.

GitHub automatski `Source code` ZIP/TAR linkovi nastaju iz taga na GitHub platformi; ByFTP workflow ne objavljuje dodatni custom Source ZIP.

## Što ByFTP ne može kontrolirati

Odredišni poslužitelj nužno vidi mrežne i autentikacijske podatke potrebne za vezu. OS, firewall, EDR, backup/sync sustavi ili udaljeni poslužitelj mogu imati vlastita pravila zapisivanja. Lokalno brisanje nije isto što i forenzičko brisanje fizičkih SSD blokova.
