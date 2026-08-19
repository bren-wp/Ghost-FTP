# Izdavanje na GitHubu

**Korisnik ne preuzima samo EXE ili DEB — preuzima povjerenje da verzija, kod, checksum i paket pripadaju istom izdanju.**

ByFTP release proces je zato namjerno stroži od ručnog “upload file” postupka.

## Jedan broj verzije

Datoteka `VERSION` je kanonski izvor produkcijskog broja.

Isti broj mora završiti u:

- runtime aplikaciji;
- README-u;
- CHANGELOG-u;
- Windows/Linux/macOS paketima;
- GitHub Releaseu;
- GitHub Packageu `ByFTP.Windows`.

Ako se ti slojevi raziđu, CI treba pasti prije objave.

## Zašto se stari tag ne prepisuje

Objavljeni `vX.Y.Z` predstavlja konkretan sadržaj.

Ako nakon taga nastane novi produkcijski popravak, ispravan put je nova semantička verzija — ne pomicanje starog taga na drugi commit.

Release-version guard to provjerava prije mergea.

## Što se provjerava prije objave

Produkcijski tok uključuje:

- sigurnosne i privacy audite;
- hrvatski audit;
- version audit;
- Python regresije;
- `go test ./...`;
- `go test -race ./...`;
- `go vet ./...`;
- Windows x64/x86 build;
- Linux amd64/arm64/i386 build;
- macOS Universal build.

Tek nakon zelenih gateova publisher smije nastaviti.

## Produkcijski paketi

### Windows

Release može sadržavati:

- Setup x64/x86;
- Portable x64/x86;
- ZIP distribucije;
- GitHub Package `ByFTP.Windows`.

### Linux

DEB paketi za:

- amd64;
- arm64;
- i386.

### macOS

Universal PKG za Intel i Apple Silicon.

## Checksum i metapodaci

Release uključuje `SHA256.txt` i build/release metapodatke.

Publisher nakon uploada ponovno provjerava očekivani skup artefakata i njihove digest vrijednosti. Cilj je da “upload je uspio” ne bude jedini dokaz ispravnog izdanja.

## GitHub Packages

`ByFTP.Windows` paket koristi isti `VERSION` izvor kao GitHub Release.

Publish koristi `--skip-duplicate`, čime ponovni run iste verzije ne pokušava stvoriti nepotrebnu duplikaciju.

## 1.0.5 release fokus

1.0.5 je shared-hosting kompatibilnosno izdanje i treba sadržavati zajedno:

- home-relative FTP control naredbe;
- control-only quote operacije;
- MLSD→LIST session fallback;
- shared-hosting regresijske testove;
- marketinški preuređenu korisničku dokumentaciju.

Ti elementi ne smiju biti objavljeni pod starim `v1.0.4` tagom.

## Potpisivanje

Release pipeline ne smije tvrditi da je paket Verified Publisher ili Apple notarized ako stvarni certifikat/identitet nije dostupan.

Detalji su u [POTPISIVANJE.md](POTPISIVANJE.md).

## Načelo izdanja

**Jedna verzija, jedan sadržaj, provjerljivi artefakti.**

To korisniku daje jednostavniji odgovor na pitanje: “Je li paket koji sam preuzeo zaista ByFTP izdanje koje README opisuje?”
