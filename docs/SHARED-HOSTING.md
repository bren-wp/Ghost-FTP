# ByFTP za shared hosting

**Imate host, FTP korisnika i lozinku iz hosting panela? Ovaj vodič vodi vas od tih podataka do web datoteka.**

ByFTP 1.0.5 posebno je usmjeren na tipične shared-hosting scenarije u kojima FTP račun nakon prijave dobiva vlastiti home/root direktorij i u njemu upravlja web sadržajem.

## Podaci koje trebate

Iz hosting panela prepišite:

- **FTP host** — npr. `ftp.domena.hr` ili server hostname;
- **FTP username** — često `korisnik@domena.hr`;
- **FTP password**;
- **port**;
- informaciju koristi li hosting FTP, explicit FTPS ili implicit FTPS.

Nemojte upisivati `https://` u Host polje.

## Preporučeni redoslijed

Ako hosting dokumentacija navodi FTPS, koristite ga.

1. `FTPS (eksplicitni)` + port 21 ako provider kaže FTP over TLS / explicit TLS.
2. `FTP` + port 21 ako provider izričito nudi samo plain FTP.
3. `FTPS (implicitni)` + port 990 samo ako je tako navedeno.
4. `SFTP` + port 22 samo ako hosting zaista nudi SSH/SFTP pristup.

FTP i SFTP nisu isti protokol čak i kada oba služe prijenosu datoteka.

## Prvi spoj

Na Windowsu:

1. Odaberite protokol.
2. Upišite host.
3. Provjerite port.
4. Upišite puni username.
5. Upišite lozinku.
6. Kliknite **Poveži**.

ByFTP potvrđuje spoj tek nakon autentikacije i početnog listinga.

## Gdje se nalazi web stranica

Najčešći web root nazivi su:

- `public_html`;
- `www`;
- `httpdocs`;
- `htdocs`;
- direktorij domene.

FTP subaccount može biti ograničen izravno na jednu od tih mapa. U tom slučaju nakon spajanja možda odmah vidite `index.php`, `wp-admin`, `wp-content` i slične datoteke bez dodatnog `public_html` direktorija.

## Zašto ByFTP 1.0.5 bolje odgovara shared hostingu

### Jedan login-home model

FTP URL operacije i raw control naredbe sada koriste isti logički home prostor.

Primjer: ono što korisnik vidi kao `/public_html/slika.jpg` za raw FTP naredbu postaje `public_html/slika.jpg`, a ne fizički server-absolute `/public_html/slika.jpg`.

To je posebno važno kada FTP server nije potpuno chrootan ili kada provider koristi kompleksniji filesystem layout.

### MLSD i LIST

Ako server podržava MLSD, ByFTP koristi strukturirani listing.

Ako ga ne podržava ili vrati neupotrebljiv format, ByFTP prelazi na klasični LIST i ne ponavlja neuspjeli MLSD pri svakom refreshu iste sesije.

### Pasivni FTP i NAT

Shared-hosting FTP server može biti iza NAT-a. ByFTP/curl koristi passive EPSV/PASV mehanizam i ne vjeruje slijepo IP adresi koju PASV odgovor može vratiti.

### Control naredbe bez nepotrebnog data transfera

`MKD`, rename, delete i CHMOD operacije koriste control-only quote način. Nakon uspješne mutacije curl ne mora otvarati dodatni directory data transfer samo da bi završio request.

## Ako FTP login ne radi

### 530 Login incorrect

Najprije provjerite puni username. Kod shared hostinga često nije dovoljan samo `korisnik`; potreban je cijeli `korisnik@domena`.

### 421 Too many connections

Zatvorite druge FTP klijente ili stare sesije. Hosting često ograničava broj istodobnih FTP veza po računu ili IP adresi.

### 425 / 426 data connection problem

Login može biti ispravan, ali passive FTP data-channel portovi mogu biti blokirani ili pogrešno konfigurirani na hostingu/firewallu.

### TLS / certificate error

Provjerite koristite li hostname koji je provider naveo za FTPS. IP adresa ili pogrešan alias može ne odgovarati certifikatu.

## Upload web stranice

Za upload većeg broja datoteka:

- otvorite lokalni direktorij projekta;
- otvorite remote web root;
- označite datoteke/mape;
- koristite **Pošalji**;
- pratite red prijenosa.

ByFTP koristi privatni lokalni snapshot i remote staging/commit pristup. Zbog toga transfer može koristiti dodatni lokalni privremeni prostor, ali završna datoteka ne nastaje jednostavnim direktnim overwriteom.

## WordPress korisnici

Prije ručne zamjene plugin/theme/core datoteka preporučuje se imati hosting backup ili uključiti ByFTP backup gdje workflow to podržava.

Kod ručnog održavanja posebno pazite na:

- `wp-config.php`;
- `.htaccess`;
- upload direktorije;
- permission/owner pravila hostinga.

## Kada kontaktirati hosting

Kontaktirajte provider ako:

- isti račun ne radi ni u drugom FTP klijentu;
- port je zatvoren;
- 421 se stalno vraća i nema drugih aktivnih sesija;
- 425/426 se ponavlja na više mreža;
- login radi, ali write/rename/CHMOD je zabranjen;
- FTPS certifikat je istekao ili ne odgovara službenom hostnameu.

## Najkraća preporuka

**Koristite točno FTP podatke iz hosting panela, puni username i protokol koji provider navodi. ByFTP 1.0.5 zatim zadržava login-home semantiku kroz listing, upload, rename, delete i ostale FTP operacije.**
