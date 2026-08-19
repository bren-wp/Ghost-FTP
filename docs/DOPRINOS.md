# Doprinos projektu

**Dobar doprinos ByFTP-u nije onaj koji samo dodaje opciju — nego onaj koji korisniku uklanja stvaran problem bez spuštanja sigurnosti i stabilnosti.**

ByFTP je Brendigo vlasnički/source-available projekt. Prije izmjena pročitajte `LICENSE` i poštujte prava koja su tamo izričito dana.

## Kakve doprinose posebno cijenimo

- reprodukciju stvarnog FTP/FTPS/SFTP problema;
- shared-hosting kompatibilnost;
- jasnije poruke o greškama;
- regresijske testove za postojeće bugove;
- sigurnosno učvršćivanje bez “disable verification” prečaca;
- poboljšanje dostupnosti i UX-a;
- kvalitetniju hrvatsku dokumentaciju;
- smanjenje dupliciranog koda kada ne mijenja sigurnosne granice.

## Prije pisanja koda

Dobro definirajte problem:

1. što korisnik pokušava napraviti;
2. što ByFTP trenutno radi;
3. na kojoj platformi/protokolu;
4. je li problem reproduktibilan;
5. može li popravak utjecati na vjerodajnice, path handling ili overwrite semantiku.

Kod mrežnog klijenta “mala” promjena može imati velik sigurnosni učinak.

## Pravila arhitekture

Doprinos ne bi trebao:

- dodati fiksni vanjski runtime API bez jasnog product/security razloga;
- uključiti telemetry/analytics SDK;
- staviti lozinku u command-line argument;
- isključiti TLS ili SFTP host-key provjeru radi kompatibilnosti;
- vratiti `RemoveAll` na sigurnosno osjetljive ByFTP temp/state putanje;
- zaobići transfer staging samo radi kraćeg koda;
- dodati vanjski Go modul bez arhitekturne i sigurnosne provjere.

## Shared-hosting promjene

FTP kompatibilnost treba promatrati kroz stvarni login/home model.

Ako mijenjate URL, raw `QUOTE` naredbu, MLSD/LIST parser ili passive-mode ponašanje, test mora pokriti barem jedan realan hosting scenarij.

Posebno pazite da raw FTP control operand ne dobije drukčiji root semantički model od listing/upload putanje.

## Obavezna kvaliteta

Prije PR-a očekuje se prolaz sljedećih provjera gdje su dostupne:

```bash
go test ./...
go test -race ./...
go vet ./...
```

Uz to projekt koristi Python audite iz `scripts/` i platformske CI buildove.

## Regresijski test je dio popravka

Ako popravljate bug koji se može deterministički reproducirati, dodajte test koji bi pao na starom kodu.

Dobri primjeri postojećih regresija uključuju:

- IPv6 bracket validaciju;
- MLSD→LIST fallback;
- shared-hosting home-relative `MKD`;
- upload source SHA-256 stabilnost;
- remote revalidation;
- transfer generation reconnect race;
- filesystem no-replace.

## Hrvatski korisnički sadržaj

Korisničke poruke, dokumentacija, issue/PR predlošci i release površine trebaju ostati na hrvatskom jeziku.

Tehnički nazivi protokola, API-ja i standardnih naredbi mogu ostati u izvornom obliku kada je to jasnije.

## Pull request

PR treba jasno navesti:

- problem;
- rješenje;
- sigurnosni utjecaj;
- testove;
- platforme koje su pogođene;
- postoji li promjena produkcijskog sadržaja koja zahtijeva novu verziju.

## Release disciplina

Ako je `v$VERSION` već objavljen, novi produkcijski kod ne smije tiho ući pod istim brojem verzije. CI release-version guard to namjerno blokira.

## Cilj doprinosa

**ByFTP treba postati jednostavniji za korisnika, a ne nepredvidljiviji ispod površine.**

Ako doprinos smanjuje trenje, dodaje dokaz kroz test i poštuje postojeće granice privatnosti/sigurnosti, ide u pravom smjeru.
