# Plan razvoja

**ByFTP se razvija prema jednom jasnom cilju: biti jednostavan alat za svakodnevni hosting rad, ali s ozbiljnim sigurnosnim granicama ispod površine.**

Plan razvoja zato ne mjeri kvalitetu samo brojem novih opcija. Prioritet imaju pouzdano spajanje, stabilni transferi, jasne poruke korisniku, privatnost i platforme koje se mogu stvarno održavati.

## Smjer proizvoda

ByFTP treba biti posebno jak u scenarijima kao što su:

- shared hosting i `public_html` workflow;
- WordPress/PHP deploy i održavanje;
- ručni prijenos web datoteka;
- FTPS hosting računi;
- SFTP serveri s provjerenim host ključem;
- siguran overwrite, backup i rollback;
- rad bez obaveznog cloud računa.

## Završene produkcijske cjeline

### Stabilna veza

- FTP, explicit FTPS, implicit FTPS i SFTP adapteri;
- stvarni login + listing probe prije stanja povezano;
- session lifecycle s kontroliranim disconnectom;
- reconnect zaštita dok se prethodna sesija zatvara;
- shared-hosting FTP login/home semantika za listing i control naredbe.

### Shared-hosting kompatibilnost

- pasivni EPSV/PASV rad;
- ignoriranje nepouzdane PASV IP adrese u NAT scenarijima;
- MLSD s LIST fallbackom;
- cache fallbacka za ostatak sesije;
- username oblici poput `account@domain`;
- control-only raw FTP operacije bez nepotrebnog data transfera.

### Sigurnost transfera

- privatni lokalni upload snapshot;
- SHA-256 stabilnost izvora;
- remote temp staging;
- post-upload remote revalidacija;
- backup/rollback commit;
- local no-replace aktivacija;
- bounded rekurzivno brisanje;
- symlink/junction/reparse zaštite.

### Queue i concurrency

- pause/resume;
- cancel;
- retry;
- batch rezervacija;
- connection identity binding;
- generation guard kroz reconnect lifecycle;
- race-detector regresije.

### Privatnost

- bez aplikacijske telemetrije;
- bez oglasa i analytics SDK-ova;
- bez fiksnog Brendigo runtime API-ja;
- DPAPI za Windows spremljene profile;
- sanitizirani child-process environment;
- produkcijski `go telemetry off` gate.

### Release kvaliteta

- jedan kanonski `VERSION`;
- version-drift CI guard;
- Windows x64/x86 build;
- Linux amd64/arm64/i386 paketi;
- macOS Universal paket;
- checksum i release metadata;
- GitHub Package vezan uz istu verziju.

## Sljedeći prioriteti

### 1. Još bolja shared-hosting dijagnostika

Cilj je korisniku jasnije razlikovati:

- DNS problem;
- port/firewall problem;
- FTP login grešku;
- TLS/certifikat grešku;
- permission problem nakon logina;
- server koji ne podržava određenu FTP naredbu.

Dobra poruka o grešci često vrijedi više od još jedne opcije u postavkama.

### 2. Unix SFTP credential broker

Linux/macOS SFTP password i private-key passphrase neće biti uključeni preko nesigurnog command-line ili običnog environment transporta.

Prioritet je siguran broker koji može zadovoljiti postojeći privacy model i proći regresijske testove.

### 3. Potpisivanje distribucija

Kada budu dostupni stvarni certifikati:

- Windows Authenticode;
- macOS Developer ID;
- notarizacija macOS paketa.

Projekt neće simulirati Verified Publisher ili notarized status bez stvarnog identiteta.

### 4. UX i brzina rada

Daljnji Windows GUI razvoj treba pojednostaviti:

- brzo spremanje profila;
- jasnije stanje transfera;
- jednostavniji retry;
- bolji fokus/keyboard workflow;
- razumljivije poruke za shared-hosting korisnika.

### 5. Šira server kompatibilnost bez slabljenja sigurnosti

Novi fallbackovi imaju smisla samo kada se mogu razlikovati od stvarne greške i kada ne zahtijevaju gašenje TLS/host-key/path zaštita.

## Što nije cilj

ByFTP se ne razvija prema modelu:

- obaveznog korisničkog računa;
- cloud sync platforme;
- analytics-first proizvoda;
- automatskog “prihvati sve certifikate” povezivanja;
- skrivanja važnih grešaka samo da bi UI izgledao uspješno.

## Mjerilo kvalitete

Nova mogućnost je spremna tek kada:

1. ima jasan korisnički problem koji rješava;
2. ne spušta postojeću sigurnosnu granicu;
3. ima regresijski test gdje je moguće;
4. prolazi unit/race/vet i platformske buildove;
5. dokumentacija objašnjava korist i ograničenje.

---

**Smjer ByFTP-a: manje trenja za korisnika, više dokaza ispod površine.**
