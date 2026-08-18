# ByFTP — potpisivanje

ByFTP build namjerno **ne fabricira** Verified Publisher, Developer ID ili notarizacijski identitet.

## Windows Authenticode

Za javnu produkcijsku distribuciju, kada postoji stvarni Brendigo code-signing identitet:

1. potpisati x64 i x86 Portable i Setup binarije nakon završne PE/resource obrade;
2. potpisati i internu Windows komponentu uklanjanja koja se ugrađuje u Setup lifecycle, jer će završiti na korisničkom sustavu nakon instalacije;
3. koristiti SHA-256 i pouzdani timestamp prema pravilima izdavatelja certifikata ili potpisne usluge;
4. na čistom Windows računalu provjeriti sve instalirane/javne binarije s `signtool verify /pa /v`;
5. ponovno izgraditi Windows ZIP-ove iz potpisanih javnih binarija;
6. ponovno generirati `BUNDLE-SHA256.txt` i završni `SHA256.txt`.

Korisnik za normalno uklanjanje koristi standardni Windows **Postavke → Aplikacije → Instalirane aplikacije → ByFTP** lifecycle. Interna komponenta uklanjanja nije zaseban korisnički distribucijski paket.

## macOS Developer ID i notarizacija

macOS paket je Universal Intel+Apple Silicon PKG. Bez stvarnog Brendigo Apple certifikata ne smije se tvrditi da je Developer ID potpisan ili notariziran.

Kada identitet bude dostupan:

1. potpisati Universal CLI binarij i `ByFTP.app` sadržaj odgovarajućim Developer ID Application identitetom;
2. provjeriti `codesign --verify --deep --strict --verbose=2`;
3. izgraditi PKG i potpisati ga Developer ID Installer identitetom;
4. poslati paket Apple notarizacijskoj usluzi;
5. nakon uspješne notarizacije napraviti `stapler staple`;
6. provjeriti `spctl` i `pkgutil` rezultat na čistom Macu;
7. tek tada generirati završni SHA-256 i objaviti paket.

## Linux

Linux DEB trenutačno nema zaseban Brendigo paketni repository-signing kanal. Integritet GitHub Release DEB-a provjerava se preko službenog `SHA256.txt` i release provenance podataka.

Ako se kasnije uvede vlastiti APT/RPM repozitorij, repository metadata i signing key lifecycle moraju biti zasebno sigurnosno dizajnirani; nije dovoljno samo potpisati jednu paketnu datoteku.

## Tajne za potpisivanje

Privatni certifikat, P12, keychain password, Apple API/app-specific credential, Windows signing secret ili hardware-token credential **ne smije biti u repozitoriju, javnom Actions artefaktu ni GitHub Releaseu**.

Potpisni secrets moraju koristiti odgovarajući GitHub/organizacijski secret management ili vanjski hardware/cloud signing mehanizam s minimalnim ovlastima.

## Fail-closed pravilo

CI bez pravog signing identiteta mora jasno prijaviti nepotpisano stanje umjesto stvaranja self-signed zamjene ili lažnog publishera.

Dokumentacija smije tvrditi:

- **Verified Publisher** tek kada Windows provjera stvarno uspije s pravim Brendigo Authenticode identitetom;
- **Developer ID/notarized** tek kada Apple provjere stvarno uspiju za objavljeni PKG.

SHA-256 i tehnički build testovi dokazuju integritet artefakta, ali nisu zamjena za platformni identitet izdavača.
