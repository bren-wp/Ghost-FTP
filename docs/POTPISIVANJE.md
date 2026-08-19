# Potpisivanje distribucija

**Digitalni potpis ima vrijednost samo ako pripada stvarnom izdavaču. ByFTP zato ne simulira Verified Publisher ili notarized status.**

Potpisivanje nije marketinška oznaka nego kriptografska veza između paketa i identiteta izdavača.

## Windows Authenticode

Kada je dostupan stvarni Brendigo code-signing certifikat, Windows distribucije trebaju biti potpisane Authenticode mehanizmom.

To korisniku omogućuje provjeru:

- tko je potpisao binarij;
- je li sadržaj promijenjen nakon potpisa;
- je li certifikat valjan prema Windows trust modelu.

Dok stvarni certifikat nije dostupan, release ne smije prikazivati lažni `Verified Publisher` status.

## macOS Developer ID i notarizacija

Za puni macOS trust lifecycle potreban je stvarni Apple Developer identitet.

Ciljani proces je:

1. potpisivanje aplikacijskih/paketnih komponenti;
2. izrada finalnog PKG-a;
3. Apple notarizacija;
4. stapling gdje je primjenjivo;
5. provjera prije objave.

Bez stvarnih Apple credentials release ne smije tvrditi da je notariziran.

## Zašto checksum nije zamjena za potpis

SHA-256 potvrđuje da je datoteka identična očekivanoj datoteci iz releasea, ali sam po sebi ne dokazuje identitet izdavača ako checksum dolazi iz nepouzdanog izvora.

Digitalni potpis i checksum rješavaju različite probleme i najbolje rade zajedno.

## Što ByFTP već daje bez certifikata

I bez publisher certifikata release proces koristi:

- kanonski VERSION;
- nepromjenjive tagove;
- CI gateove;
- SHA-256 checksumove;
- build metadata;
- kontrolirani skup release asseta.

To ne zamjenjuje code signing, ali daje provjerljivu release disciplinu dok stvarni signing identitet nije dostupan.

## Pravilo za budućnost

Potpisivanje se smije uključiti tek kada se tajne mogu sigurno držati u CI/release okruženju i kada pipeline može dokazati da je finalni artefakt stvarno potpisan.

**ByFTP će radije jasno napisati “nije potpisano” nego korisniku prikazati povjerenje koje tehnički ne postoji.**
