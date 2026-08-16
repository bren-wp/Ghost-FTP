## Ovlaštenje

- [ ] Ovu izmjenu izvornog koda Brendigo je izričito zatražio ili odobrio.
- [ ] Pročitao/la sam `LICENSE` i `docs/DOPRINOS.md` te ću ih poštovati.

> ByFTP je vlasnički/source-available softver. Otvaranje pull requesta ili forka samo po sebi ne daje pravo izmjene, redistribucije, rebrandinga, prodaje, sublicenciranja ili izrade izvedenih distribucija izvan prava koja je Brendigo izričito dao i ograničenih GitHub platformskih prava.

## Sažetak

Opišite izmjenu i razlog zbog kojeg je potrebna.

## Sigurnost i privatnost

- [ ] Nije dodana telemetrija, analitika, oglašivački SDK ni vanjski servis za izvještavanje o rušenju.
- [ ] Nije dodan automatski vanjski API ni skriveno mrežno odredište.
- [ ] Lozinke, zaporke i privatni ključ ne završavaju u argumentima naredbenog retka ni trajnim zapisima.
- [ ] SFTP host-key provjera i pinning ostaju aktivni.
- [ ] Lokalna path traversal, symlink, junction i reparse-point zaštita ostaju aktivni.
- [ ] Nije dodan vanjski Go modul bez izričite arhitekturne/sigurnosne provjere.
- [ ] Testovi, slike i fixturei ne sadrže stvarne vjerodajnice, produkcijske poslužitelje ili podatke klijenata.
- [ ] Korisnički tekst i nova dokumentacija napisani su na hrvatskom.

## Provjera

- [ ] `go test ./...`
- [ ] `go test -race ./...`
- [ ] `go vet ./...`
- [ ] `python scripts/generate_brand_assets.py --check`
- [ ] `python scripts/audit_croatian.py`
- [ ] `python scripts/audit_privacy.py`
- [ ] Windows produkcijski build provjeren je ako izmjena dira Windows-specifičan kod.

## Regresijski testovi

Opišite testove dodane ili izmijenjene za vezu, transfere, putanje, profile, instalaciju/uklanjanje ili drugo pogođeno ponašanje.
