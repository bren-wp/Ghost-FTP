# Sigurnost

**ByFTP je dizajniran tako da najrizičniji trenutak nije klik na “Poveži”, nego ono što se događa kada datoteka treba biti stvarno aktivirana, prepisana ili obrisana.**

Zato sigurnosni model ne završava na TLS-u. Obuhvaća vjerodajnice, host identitet, lokalne putanje, transfer queue, temp datoteke, remote commit i release proces.

## Što to znači za korisnika

- pogrešna ili prekinuta operacija ne bi trebala izgledati kao uspješan transfer;
- ByFTP pokušava raditi preko staging/commit faze umjesto neposrednog destruktivnog overwritea;
- symlink, junction i reparse-point preusmjeravanja blokiraju se na osjetljivim lokalnim putanjama;
- SFTP host-key promjena blokira vezu dok korisnik ne provjeri novi fingerprint;
- spremljene tajne ne prenose se automatski na drugi endpoint ili račun;
- release workflow ne smije tiho prepisati već objavljenu verziju drugim sadržajem.

## FTP i shared hosting

ByFTP 1.0.5 dodatno usklađuje FTP ponašanje s tipičnim shared-hosting računima.

FTP URL listing i upload/download koriste login/home namespace servera. Raw FTP control naredbe poput `MKD`, `RNFR`, `RNTO`, `DELE`, `RMD` i `SITE CHMOD` sada koriste isti logički namespace bez vodećeg `/` koji bi na non-chrooted serveru mogao značiti fizički server root.

Quote-only operacije koriste `no-body`, pa se nakon uspješne control naredbe ne pokreće nepotreban directory data transfer koji bi mogao pasti i pretvoriti uspješnu mutaciju u lažno prijavljenu grešku.

## FTPS

Explicit FTPS koristi TLS zahtjev nad FTP vezom, a implicit FTPS TLS od početka sesije.

ByFTP:

- ne koristi `ssl-no-revoke` za globalno isključivanje provjere opoziva certifikata;
- na Windowsu capability-checkom koristi `ssl-revoke-best-effort` kada curl to podržava;
- na starijem curlu ne dodaje opciju koju alat ne razumije;
- ne nasljeđuje proizvoljne proxy/TLS environment varijable koje bi promijenile mrežnu politiku child procesa.

## SFTP host-key povjerenje

Prije prvog povjerenja ByFTP prikazuje SHA-256 fingerprint poslužitelja.

Ako je fingerprint spremljen, promjena ključa blokira vezu. Pin je vezan uz endpoint i ne prenosi se na privremeno izmijenjeni host ili port.

Privatni `known_hosts` artefakti koriste kontrolirani session lifecycle.

## Privatni ključ

SFTP private key prije korištenja mora ostati isti regularni filesystem objekt tijekom bounded čitanja.

ByFTP zatim izrađuje privatni session snapshot i OpenSSH-u predaje snapshot, ne originalnu korisničku putanju. Session snapshot se uklanja pri zatvaranju veze.

## Sigurnost lokalnog uploada

Upload source prolazi više granica:

1. original se otvara i verificira kao regularna datoteka;
2. sadržaj se kopira u privatni ByFTP snapshot;
3. tijekom kopiranja računa se SHA-256;
4. isti otvoreni izvor ponovno se čita i digest mora biti identičan;
5. curl/OpenSSH dobiva samo snapshot putanju;
6. nakon mrežnog čitanja snapshot se ponovno verificira i hashira;
7. snapshot se uklanja prije remote commit faze.

Ako se sadržaj promijeni ili cleanup ne uspije, finalni remote commit se blokira.

Ovaj model koristi dodatni privremeni disk prostor približno veličini upload datoteke. To je namjerna cijena za stabilniji sadržaj.

## Remote commit i overwrite

FTP/FTPS/SFTP upload prvo šalje privremeni remote objekt. Neposredno prije finalnog rename/backup commita ByFTP ponovno lista odredišni direktorij.

Time se provjerava što se dogodilo tijekom potencijalno dugog uploada:

- ako se pojavila nova mapa ili symlink pod finalnim imenom, commit se blokira;
- ako se pojavila datoteka i uključen je `SkipExisting`, temp se čisti i transfer se preskače;
- overwrite i backup koriste svježe remote stanje.

## Lokalni download i no-replace

ByFTP koristi platform-specific no-replace aktivaciju kako konkurentno odredište ne bi bilo tiho prepisano.

- Windows koristi exclusive `MoveFileExW` model;
- Linux službene arhitekture koriste `renameat2(RENAME_NOREPLACE)`;
- macOS i generički fallback za regularne staging datoteke koriste exclusive hard-link aktivaciju gdje je primjenjivo.

## Rekurzivno brisanje

Brisanje je ograničeno maksimalnom dubinom i brojem stavki.

Lokalni direktorij se otvara i provjerava kroz stabilniji handle model, a identitet se ponovno provjerava prije destruktivnih koraka. Remote delete ne prati stavke koje su označene kao symlink.

## Transfer queue i reconnect

Posao u transfer queueu vezan je uz identitet veze.

`ReserveBatch` i `RetryBatch` captureaju monotonu transfer generation prije `ConnectionIdentity()`, zatim ponovno provjeravaju istu generation nakon identity poziva. Disconnect/reconnect usred te granice odbija mutaciju reda umjesto da stari connection identity završi na novoj sesiji.

Identity lookup se izvodi izvan transfer mutexa kako sigurnosna provjera ne bi uvodila novi lock-order/deadlock rizik.

## Stabilnost state zapisa

ByFTP state/config zapis koristi privremenu datoteku, `fsync`, atomski replace i platformske durability korake.

Na Unix sustavima nakon renamea sinkronizira se i direktorijski metadata entry gdje platforma to podržava; Windows zadržava write-through replace mehanizam.

Recovery `.previous` generacija učvršćuje se prije aktivacije novog current stanja.

## Vjerodajnice

Na Windowsu spremljene profilne tajne koriste DPAPI. Tajna se ponovno koristi samo kada identitet endpointa/računa odgovara profilu.

Aktivne FTP/FTPS tajne ne šalju se kao command-line argumenti. Child-process environment je sanitiziran od proxy, TLS keylog i sličnih varijabli koje bi mogle promijeniti sigurnosnu granicu.

## Fail-closed ograničenja

Neke mogućnosti su namjerno blokirane dok nemaju dovoljno siguran transport tajne.

Linux/macOS SFTP password i private-key passphrase ne provode se kroz obični command-line ili nezaštićenu environment varijablu samo radi funkcionalnog pariteta.

## Sigurnost produkcijskog izdanja

Aktualna produkcijska bazna linija dodatno učvršćuje release granicu:

- `VERSION` je kanonski izvor produkcijskog broja;
- CI blokira produkcijske promjene pod već objavljenim tagom bez promjene verzije;
- GitHub Package, runtime i release koriste isti VERSION izvor;
- publisher ne smije prepisati postojeći tag koji pokazuje na drugi commit;
- release asseti dobivaju checksum i metapodatke za provjeru.

## Što sigurnost ne može obećati

ByFTP ne može jamčiti ponašanje svakog FTP servera, filesystema, diska, antivirusa ili hosting providera.

Ne može ni pretvoriti server bez write prava u writable server, niti ukloniti administrativna ograničenja hostinga.

Cilj sigurnosnog modela je konkretniji: **jasnije granice, manje implicitnih trust putova i fail-closed ponašanje kada se ključna provjera ne može dokazati.**
