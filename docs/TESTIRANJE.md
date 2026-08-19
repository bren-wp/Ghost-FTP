# Testiranje i kvaliteta

**ByFTP ne tretira “build je prošao” kao dovoljan dokaz kvalitete.**

Prije produkcijskog izdanja projekt prolazi više slojeva provjere: funkcionalne testove, race detector, vet, sigurnosne/privacy audite i stvarne platformske buildove.

## Što korisnik dobiva tim pristupom

Testovi su usmjereni na probleme koji mogu stvarno pokvariti rad:

- pogrešno stanje “povezano”;
- upload koji aktivira krivi remote objekt;
- reconnect race;
- lokalni overwrite race;
- symlink/junction izlaz;
- pogrešno vezane spremljene vjerodajnice;
- SFTP host-key regresiju;
- release koji nosi pogrešnu verziju;
- shared-hosting FTP server bez MLSD podrške.

## Glavni Go testovi

Standardni test suite:

```bash
go test ./...
```

Provjerava engine, remote adaptere, profile, sigurnosne primitive, filesystem i transfer queue.

## Race detector

Konkurentni transferi i connect/disconnect lifecycle zahtijevaju dodatnu provjeru:

```bash
go test -race ./...
```

Race detector je posebno važan za generation/connection-identity binding, queue state i session lifecycle.

## Go vet

```bash
go vet ./...
```

Vet služi kao dodatna statička provjera prije releasea.

## Python auditi i regresije

`scripts/` sadrži fail-closed provjere koje zaključavaju projektna pravila koja obični unit test ne mora lako vidjeti.

Primjeri:

- hrvatski korisnički tekst;
- privatnost i zabrana telemetry/vendor runtime hookova;
- version konzistentnost;
- release drift guard;
- filesystem hardening;
- FTPS certificate revocation politika;
- shared-hosting FTP home-relative control naredbe;
- MLSD→LIST fallback ponašanje;
- transfer generation ordering.

## Shared-hosting regresije u 1.0.5

1.0.5 dodaje konkretne testove za scenarije koji su česti kod hostinga:

- username oblika `account@example.com`;
- `public_html` kao login-home relativna putanja;
- `MKD` bez pogrešnog server-absolute `/public_html/...` operanda;
- `no-body` za control-only FTP quote operacije;
- MLSD odgovor koji nije valjan strukturirani listing;
- automatski LIST fallback;
- pamćenje fallbacka tako da se MLSD ne ponavlja pri svakom refreshu.

## Process-smoke testovi

Dio testova koristi lažne `curl` ili `sftp` executable skripte kako bi provjerio stvarni child-process ugovor:

- koje argumente alat dobiva;
- ide li tajna kroz očekivani kanal;
- koristi li se stdin konfiguracija;
- jesu li raw FTP naredbe ispravne;
- je li SFTP batch/AskPass ponašanje očekivano.

To testira granicu između Go koda i vanjskog mrežnog alata, ne samo interne helper funkcije.

## Platformski buildovi

CI prije produkcijskog mergea provjerava:

- Windows x64;
- Windows x86;
- Linux amd64;
- Linux arm64;
- Linux i386;
- macOS Universal.

Time platform-specific kod ne ostaje “teoretski podržan”.

## Release provjere

Release kvaliteta uključuje:

- `VERSION` kao kanonski broj;
- README/CHANGELOG usklađenost;
- package version usklađenost;
- zaštitu od prepisivanja postojećeg taga drugim sadržajem;
- checksumove i release metadata;
- provjeru artefakata prije objave.

## Kako dodati novi test

Dobar regresijski test treba:

1. reproducirati stvarni problem ili sigurnosnu granicu;
2. pasti na starom ponašanju;
3. proći nakon popravka;
4. biti deterministički koliko je moguće;
5. ne sadržavati stvarne vjerodajnice ili produkcijske servere.

## Zašto nema “100% bez greške” tvrdnje

Nijedan ozbiljan mrežni klijent ne može obećati da nikada neće naići na grešku, jer rezultat ovisi i o mreži, hostingu, filesystemu, TLS infrastrukturi i korisničkim pravima.

ByFTP umjesto toga pokušava dokazati nešto korisnije: **poznate kritične klase grešaka imaju testove, a novi release ne ide van dok produkcijski gateovi ne prođu.**
