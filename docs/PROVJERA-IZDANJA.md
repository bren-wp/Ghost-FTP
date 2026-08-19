# Provjera izdanja

**Prije objave ByFTP-a cilj nije samo dobiti paket, nego dokazati da taj paket odgovara kodu, verziji i dokumentaciji koju korisnik vidi.**

Ovaj dokument je završna kontrolna lista za produkcijsko izdanje.

## 1. Verzija

Provjeriti:

- `VERSION` sadrži semantičku verziju;
- README prikazuje isti broj kao trenutačno izdanje;
- CHANGELOG ima odjeljak za tu verziju;
- GitHub Package koristi isti VERSION;
- tag `vX.Y.Z` ne postoji na drugom commitu.

## 2. Kvaliteta koda

Moraju proći:

```bash
go test ./...
go test -race ./...
go vet ./...
```

Uz njih prolaze Python auditi/regresije iz `scripts/`.

## 3. Privatnost i sigurnost

Provjeriti da release nije uveo:

- aplikacijsku telemetriju;
- fiksni runtime HTTP API;
- lozinku u command-line argumentima;
- nasljeđivanje proxy/TLS/SSH helper okruženja koje ruši postojeću politiku;
- `ssl-no-revoke`;
- zaobilaženje SFTP fingerprint zaštite;
- nezaštićeni overwrite ili `RemoveAll` na sigurnosno osjetljivim ByFTP putanjama.

## 4. Shared-hosting FTP

Za 1.0.5 i kasnije provjeriti:

- login username može sadržavati `@`;
- FTP listing `/public_html` ostaje u login/home namespaceu;
- raw `MKD`, rename, delete i CHMOD operandi ne postaju server-absolute samo zbog vodećeg `/` u UI putanji;
- quote-only operacije koriste `no-body`;
- MLSD fallback na LIST radi;
- nakon uspješnog LIST fallbacka MLSD se ne ponavlja pri svakom refreshu.

## 5. Windows build

Provjeriti:

- x64 build;
- x86 build;
- Setup;
- Portable;
- ZIP sadržaj;
- runtime verziju;
- installer/uninstaller osnovni lifecycle.

## 6. Linux build

Provjeriti DEB za:

- amd64;
- arm64;
- i386.

Paket mora sadržavati očekivanu verziju i izvršni program.

## 7. macOS build

Provjeriti Universal PKG i očekivane Intel/Apple Silicon komponente.

Ne tvrditi Developer ID/notarized status ako stvarna provjera nije moguća.

## 8. Release asseti

Očekuju se:

- platformski paketi;
- `SHA256.txt`;
- `RELEASE-NOTES.txt`;
- `BUILD-METADATA.txt`;
- GitHub Package gdje je definiran.

Broj i digest asseta trebaju biti provjereni nakon uploada.

## 9. Dokumentacija

Prije objave korisnik mora moći iz README-a doći do:

- instalacije;
- prvog shared-hosting spoja;
- podrške;
- sigurnosti;
- privatnosti;
- ograničenja izdanja.

Dokumentacija ne smije obećavati “radi na svakom hostingu” ako takva garancija tehnički nije moguća.

## 10. Završno pitanje

Prije publish koraka treba moći odgovoriti **da** na tri pitanja:

1. Je li kod prošao sve produkcijske gateove?
2. Jesu li svi paketi vezani uz isti VERSION i commit?
3. Može li korisnik provjeriti što je preuzeo?

Ako je odgovor na bilo koje pitanje “ne”, izdanje nije spremno.

---

**Dobar release je onaj koji korisniku daje i proizvod i dokaz da je dobio pravi proizvod.**
