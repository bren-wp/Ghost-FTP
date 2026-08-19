# Arhitektura

**ByFTP arhitektura je napravljena s jednim praktičnim ciljem: korisnik ne bi trebao dobiti potpuno drukčije ponašanje samo zato što koristi FTP, FTPS, SFTP, Windows ili terminal.**

Zato projekt koristi jedan zajednički Go engine, a platforme i protokoli se priključuju kroz jasno odvojene slojeve.

## Zašto je to važno korisniku

Zajednički engine omogućuje da ključna pravila budu ista:

- validacija putanja;
- transfer queue;
- retry/cancel pravila;
- remote staging i commit;
- sigurnosne granice;
- session lifecycle;
- profile i connection identity model.

To smanjuje mogućnost da se primjerice upload na FTP-u ponaša potpuno drukčije od SFTP-a u dijelovima koji ne ovise o samom protokolu.

## Glavni slojevi

### 1. Desktop / terminal sučelje

Windows koristi izvorni Win32 GUI s dvopanelnim prikazom.

Linux i macOS koriste terminalno sučelje.

Oba pristupa komuniciraju s istim aplikacijskim engineom umjesto da svaki frontend implementira vlastitu mrežnu logiku.

### 2. API engine

Engine koordinira:

- profile;
- postavke;
- remote manager;
- local filesystem servis;
- transfer manager;
- operacije nad datotekama.

Nema generički skriveni web servis ili browser backend. Aplikacija ostaje lokalni desktop/terminal alat.

### 3. Remote manager

Remote manager upravlja aktivnom sesijom i povezuje korisničke postavke s odgovarajućim adapterom.

Veza se ne smatra uspostavljenom samo zato što je napravljen adapter. Nakon kreiranja sesije izvodi se početni `List` probe; tek uspješan rezultat vodi u stanje povezano.

Disconnect prekida session context, blokira novi rad dok se prethodna veza sigurno zatvara i čeka aktivne operacije prije konačnog `Close()` adaptera.

### 4. FTP / FTPS adapter

FTP/FTPS koristi pouzdani sistemski curl koji ByFTP pokreće s kontroliranom konfiguracijom preko standardnog inputa.

Adapter vodi računa o:

- FTP/FTPS URL-ovima;
- passive EPSV/PASV radu;
- TLS konfiguraciji;
- MLSD/LIST kompatibilnosti;
- upload/downloadu;
- raw control naredbama;
- remote staging i commit modelu.

U 1.0.5 URL i raw control naredbe dijele istu login-home semantiku, što je posebno važno za shared hosting.

### 5. SFTP adapter

SFTP koristi OpenSSH alate uz privatnu session konfiguraciju.

ByFTP postavlja vlastiti `known_hosts`, ograničava implicitne helper mehanizme, provjerava fingerprint i kontrolira private-key lifecycle.

### 6. Transfer manager

Transfer manager je centralno mjesto za:

- queue;
- batch rezervaciju;
- concurrency;
- pause/resume;
- cancel;
- retry;
- vezanje posla uz connection identity/generation;
- obradu retryable i non-retryable grešaka.

To odvaja korisnički workflow od detalja konkretnog mrežnog protokola.

### 7. Security sloj

Security paket sadrži zajedničke primitive za:

- host/port/protokol validaciju;
- remote path i filename validaciju;
- lokalne root granice;
- symlink/junction/reparse zaštitu;
- DPAPI i runtime secrets;
- sigurno rekurzivno brisanje.

### 8. Platform sloj

Platform-specific kod sadrži operacije koje nije korektno simulirati generičkim Go kodom.

Primjeri:

- Windows `MoveFileExW` no-replace/write-through ponašanje;
- Linux `renameat2(RENAME_NOREPLACE)`;
- macOS exclusive activation fallback;
- Windows UI i OS integracija.

## Shared-hosting FTP putanja

ByFTP logički prikazuje FTP putanje s vodećim `/` jer je to korisniku prirodan root prikaza.

Ipak, FTP URL s jednom početnom kosom crtom u curlu predstavlja putanju unutar direktorija u koji je server smjestio korisnika nakon logina. Raw `QUOTE` naredbe šalju se ranije u protokolskom toku, pa vodeći `/` na non-chrooted hostingu može imati drukčije značenje.

1.0.5 zato mapira logički `/public_html/site` u control operand `public_html/site`, kako bi listing, upload, rename, delete i mkdir govorili o istom objektu.

## MLSD i LIST strategija

Moderniji FTP serveri često nude MLSD. ByFTP ga preferira jer je strukturiraniji.

Ako MLSD nije podržan ili server vrati odgovor koji nije stvarni MLSD listing, obični LIST se koristi kao kompatibilni fallback. Nakon uspješnog fallbacka adapter pamti odluku za ostatak sesije.

To smanjuje nepotrebne neuspjele naredbe na starijim i shared-hosting FTP servisima.

## Upload arhitektura

Upload prolazi kroz više slojeva:

1. lokalni source se verificira;
2. izrađuje se privatni snapshot;
3. snapshot se sadržajno potvrđuje;
4. adapter šalje remote temp objekt;
5. source snapshot se ponovno provjerava i uklanja;
6. remote odredište se ponovno lista;
7. staging objekt se transakcijski aktivira kroz rename/backup/rollback logiku.

Ta arhitektura je skuplja od običnog direktnog overwritea, ali daje jaču kontrolu nad sadržajem i konkurentnim promjenama.

## Download arhitektura

Download ide u lokalni `part` sibling, zatim se part validira i aktivira no-replace/backup logikom.

Cilj je spriječiti da djelomično preuzeta datoteka bude predstavljena kao završna datoteka.

## Privatnost u arhitekturi

Projekt nema vanjske Go module u runtime engineu i CI to provjerava.

Child procesi dobivaju sanitizirani environment, bez naslijeđenih proxy/TLS/SSH helper varijabli koje bi mogle promijeniti mrežnu politiku.

## Release arhitektura

`VERSION` je jedini produkcijski izvor verzije.

CI provjerava kod, testove, race detector, vet, sigurnosne audite i platformske buildove. Produkcijski release tek nakon toga izrađuje platform packages, checksumove, release assets i GitHub Package.

## Dizajnersko načelo

**Jedan engine, jasno odvojene platforme, što manje implicitnog ponašanja.**

ByFTP arhitektura nije optimizirana za najmanje linija koda, nego za to da kritična pravila budu dovoljno centralizirana da ih je moguće testirati i braniti od regresija.
