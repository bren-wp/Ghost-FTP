# Privatnost

**Vaši FTP podaci služe za spajanje na vaš server — ne za izgradnju marketinškog profila.**

Privatnost u ByFTP-u nije dodatna postavka koju treba uključiti. Projekt je oblikovan tako da runtime nema tipične mehanizme za praćenje korištenja aplikacije.

## Što korisnik dobiva

ByFTP runtime nema:

- aplikacijsku analitiku korištenja;
- oglase i oglašivačke SDK-ove;
- obavezni Brendigo cloud račun;
- vanjski crash-reporting servis;
- fiksni Brendigo HTTP API kojem se šalju aktivnosti;
- trajni activity/error log koji bilježi FTP sesije.

Mrežni promet aplikacije namijenjen je serveru koji je korisnik odabrao, uz standardne OS/certifikacijske provjere potrebne za siguran TLS rad.

## Vjerodajnice nisu command-line argument

FTP/FTPS lozinke ne predaju se curlu kao običan command-line argument.

ByFTP generira curl konfiguraciju kroz standardni input procesa, a memorijske buffer-e nakon upotrebe pokušava očistiti. Vanjske proxy i TLS environment postavke koje bi mogle preusmjeriti promet ili izvesti TLS session tajne uklanjaju se iz child-process okruženja.

## Windows spremljeni profili

Na Windowsu spremljene profilne tajne koriste DPAPI zaštitu.

Dodatno se primjenjuje identity binding:

- FTP/FTPS lozinka vrijedi samo za isti protokol, host, port i korisničko ime;
- SFTP passphrase dodatno mora pripadati istom private-key identitetu;
- promjena endpointa ili računa ne prenosi staru tajnu na novu konfiguraciju.

To smanjuje rizik da korisnik uredi profil, promijeni host, a aplikacija tiho pošalje staru spremljenu lozinku drugom serveru.

## Aktivne tajne tijekom sesije

Na Windowsu runtime tajne koriste zaštitne mehanizme platforme. Na Linuxu/macOS-u FTP/FTPS aktivne lozinke drže se samo u memoriji procesa kroz kratkotrajni runtime-secret model i brišu se pri zatvaranju sesije.

Linux/macOS terminal ne nudi trajno spremanje tih tajni.

## SFTP i AskPass

Windows SFTP password/passphrase tok koristi kontrolirani AskPass model. Tajna se ne zapisuje u običnu privremenu datoteku samo zato da bi je OpenSSH mogao pročitati.

OpenSSH child proces dodatno ne nasljeđuje SSH agent, proxy helper, PKCS#11 provider ili slične implicitne mehanizme iz korisničkog environmenta ako ByFTP nije namjerno konfigurirao takav tok.

## Privatni ključevi

ByFTP ne predaje OpenSSH-u originalnu private-key putanju nakon jednostavnog path checka.

Ključ se prije korištenja verificira i kopira u privatni session snapshot. Time kasnija zamjena izvorne putanje ne mijenja ključ aktivne sesije.

## Upload datoteke

Za FTP/FTPS/SFTP upload child proces ne dobiva originalnu lokalnu korisničku putanju. ByFTP izrađuje privatni byte-for-byte snapshot iz verificiranog otvorenog handlea i provjerava sadržaj SHA-256 digestom.

To je sigurnosni i privatnosni izbor: vanjski mrežni alat vidi samo ByFTP-ov session snapshot, ne promjenjivu izvornu putanju korisnika.

## Go telemetrija u produkcijskom buildu

Produkcijski CI i build skripte zahtijevaju stvarno stanje `go telemetry off` i odbijaju build ako taj uvjet nije potvrđen.

Projekt se ne oslanja na marketinšku tvrdnju “telemetrija je isključena”; to je provjerljiv build gate.

## TLS certificate provjere nisu telemetrija

Kod FTPS-a operativni sustav ili CA infrastruktura može provjeravati opoziv certifikata prema podacima ugrađenim u certifikat. To nije ByFTP analitika niti Brendigo tracking — to je dio sigurnosne provjere certifikata.

ByFTP ne koristi `ssl-no-revoke` kako bi potpuno isključio tu zaštitu.

## Što korisnik i dalje šalje

Kada se spojite na FTP/FTPS/SFTP server, tom serveru nužno šaljete podatke potrebne za protokol, uključujući korisničko ime i odgovarajuću autentikacijsku informaciju.

ByFTP ne može kontrolirati što vaš hosting provider radi sa svojim server logovima. Politiku privatnosti hostinga treba promatrati odvojeno od privatnosti samog ByFTP klijenta.

## Načelo proizvoda

**Manje skrivenih servisa, manje nepotrebnog prikupljanja, jasnija mrežna granica.**

ByFTP je namijenjen korisnicima koji žele alat za upravljanje datotekama, a ne dodatni cloud ekosustav oko običnog FTP zadatka.
