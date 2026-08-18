# ByFTP — privatnost

ByFTP prenosi datoteke prema poslužitelju koji korisnik sam odabere. Aplikacijski runtime nema paralelno slanje Brendigu, analitičkim servisima, oglašivačkim mrežama ili skrivenom cloud backendu.

## ByFTP runtime nema

- telemetriju korištenja
- analitiku
- oglašavanje
- vanjski crash-reporting servis
- obavezni cloud račun
- skriveni update API
- browser/localhost upravljački server
- trajni runtime activity/error log

## Mrežni promet aplikacije

ByFTP mrežni adapteri razgovaraju s korisnički odabranim FTP/FTPS/SFTP endpointom. Operativni sustav, DNS resolver, firewall, antivirus/EDR ili sam udaljeni poslužitelj mogu imati vlastito zapisivanje koje nije ByFTP telemetrija.

curl dobiva sanitizirano okruženje bez naslijeđenog proxy/TLS overridea. OpenSSH dobiva ByFTP-managed session config koji blokira ProxyCommand, ProxyJump, agent, PKCS#11 provider, KnownHostsCommand i forwarding.

## Windows podaci

Windows profili i postavke ostaju u ByFTP korisničkoj podatkovnoj mapi. Profilne tajne koriste DPAPI. UI prikazuje samo činjenicu da je spremljena tajna dostupna, ne vraća stvarnu spremljenu vrijednost u edit kontrolu.

Lozinka se automatski koristi samo za isti endpoint i korisnika; passphrase dodatno za isti privatni ključ. Promjena identiteta profila čisti credential vrijednost koja više ne pripada novom profilu.

Unesena lozinka/passphrase tijekom povezivanja ostaje u zaključanoj kontroli dok pokušaj traje i briše se nakon stvarno uspješnog povezivanja. To omogućuje retry bez ponovnog unosa, ali ne uvodi trajno spremanje.

## Linux i macOS

Terminalni frontend trenutačno ne sprema terminalne profile ni vjerodajnice na disk.

FTP/FTPS aktivna lozinka pohranjuje se samo u procesu iza kriptografski nasumičnog runtime tokena. Adapter drži token umjesto plaintext vrijednosti; kratkotrajna kopija nastaje neposredno prije curl poziva i briše se nakon uporabe. `Close()` uklanja i briše procesnu vrijednost.

SFTP password/passphrase na Linuxu/macOS-u trenutačno nije uključen. Umjesto slanja tajne kroz argument naredbenog retka ili običan environment, terminalno izdanje fail-closed zahtijeva eksplicitni privatni ključ bez passphrasea dok siguran Unix credential broker nije dovršen.

## Connect smoke testovi i privatnost

Process-level connect smoke testovi nemaju vanjski mrežni promet. Testovi stvaraju lokalni kratkotrajni fake `curl` ili `sftp` proces i koriste isključivo testne vrijednosti vjerodajnica.

Time se provjerava produkcijski child-process/stdin/parser put bez:

- stvarnog udaljenog poslužitelja
- stvarnih korisničkih računa
- produkcijskih tajni
- zapisivanja credentiala u repozitorij

FTP smoke dodatno potvrđuje da runtime token više nije moguće razriješiti nakon `Close()`.

## SFTP metapodaci

Host, korisnik i private-key putanja nisu na OpenSSH command lineu. Sesija koristi nasumični lokalni alias, privatni `known_hosts` i kratkotrajnu config datoteku. Bracketirani IPv6 host normalizira se prije OpenSSH unosa.

Windows AskPass dobiva samo zaštićene vrijednosti kroz kontrolirani child environment i daje tajnu isključivo prepoznatom password/passphrase promptu. MFA, OTP, security-key i nepoznati prompt ne dobivaju spremljenu tajnu.

Host-key fingerprint pripada endpointu. Privremeni endpoint ne može naslijediti ili prepisati pin drugog profila.

## Lokalno stanje

ByFTP nema trajni runtime log servera, putanja i transfer pogrešaka. State/config čitanje koristi bounded safe-open model i provjerenu prethodnu generaciju. ByFTP-owned direktoriji ne smiju biti preusmjereni izvan kanonske podatkovne lokacije.

Platformne podatkovne lokacije koriste OS-specifični model:

- Windows: stvarni LocalAppData
- macOS: `~/Library/Application Support`
- Linux: apsolutni `XDG_DATA_HOME` ili `~/.local/share`

## Go toolchain telemetrija nije ByFTP runtime telemetrija

Go alatni lanac ima vlastitu telemetry postavku odvojenu od ByFTP aplikacije. Produkcijski build zbog privatnosti ne smije ostati u zadanom lokalnom prikupljanju.

ByFTP GitHub Actions prije Go testova/builda izvršava:

```text
go telemetry off
```

i zahtijeva da `go telemetry` vrati `off`.

Produkcijske build skripte ponovno čitaju stvarni način rada i fail-closed odbijaju build ako nije `off`. Time dokumentacija više ne ovisi o običnoj istoimenoj OS environment varijabli koja ne predstavlja stvarno stanje Go telemetry postavke.

Lokalna skripta ne mijenja globalnu Go postavku potajno. Ako telemetry nije `off`, build se prekida i korisniku se navodi potrebna naredba.

## Offline Go build

ByFTP nema vanjske Go module. Produkcijski build koristi:

- `GOTOOLCHAIN=local`
- `GOPROXY=off`
- `GOSUMDB=off`
- `CGO_ENABLED=0`

Time standardni ByFTP build ne preuzima module ili zamjenski Go toolchain tijekom izgradnje.

## Build i release metapodaci

`BUILD-METADATA.txt` sadrži samo informacije potrebne za provenance:

- ByFTP verziju
- release commit/ref
- platformsku matricu
- činjenicu da je release quality gate prošao
- GitHub Actions run/attempt identifikatore

Ne sadrži hostove, profile, lokalne korisničke putanje ni vjerodajnice.

`SHA256.txt` sadrži samo kriptografske sažetke javnih paketa i zajedničkih release metapodataka.

Tehnički verification output i instalacijske interne komponente ostaju unutar build/CI sloja gdje su potrebni; glavni javni distribucijski ugovor sastoji se samo od službenih platformskih paketa i zajedničkih release metapodataka.

## Release privatnost i stabilnost

Release workflow ima jedan automatski okidač: promjenu `VERSION` na `main`. Tag koji publisher izradi ne pokreće drugi release proces. Svi release runovi dijele jednu serijaliziranu concurrency grupu.

To smanjuje istodobne publish operacije i čini provenance jednog izdanja jednostavnijim za provjeru.

## Što ByFTP ne može kontrolirati

Odredišni poslužitelj nužno vidi mrežne i autentikacijske podatke potrebne za vezu. OS, DNS, firewall, EDR, backup/sync sustavi ili udaljeni poslužitelj mogu imati vlastita pravila zapisivanja.

Lokalno brisanje datoteke ili procesne tajne nije isto što i forenzičko brisanje fizičkih SSD blokova ili svih kopija koje je napravio operativni sustav, backup ili sigurnosni proizvod.
