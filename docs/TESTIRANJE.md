# ByFTP 2.14.0 — testiranje

## Obavezne provjere

```text
go test ./...
go test -race ./...
go vet ./...
python scripts/generate_brand_assets.py --check
python scripts/audit_croatian.py
python scripts/audit_privacy.py
```

Na Windowsu dodatno:

```powershell
.\BUILD-WINDOWS.ps1
```

## Što testovi pokrivaju

- validaciju FTP/FTPS/SFTP veza i korisničkog unosa
- profile i DPAPI migracijske granice
- settings normalizaciju i cache
- transfer queue, parallelism, pause/resume, cancel/retry i auto-retry
- connection generation i cross-server retry blokadu
- worker panic containment
- path traversal, Windows rezervirane nazive, symlink/junction/reparse zaštite
- rekurzivne upload/download planove i rollback
- installer payload integritet i upgrade rollback
- event stream fallback i velike queue burstove
- neovisnost `Events` povratnih snimki u 2.14
- velike lokalne popise od 50.000 stavki
- hrvatske UI/release/GitHub površine
- determinističku generaciju PNG/ICO resursa

## CI

GitHub CI ima dva glavna joba:

1. Linux kvalitetu: unit, race, vet, privacy, hrvatski audit, asset audit i release metadata.
2. Windows produkcijski build: isti `BUILD-WINDOWS.ps1` put koji koristi službeno izdanje.

Merge se ne smatra spremnim dok oba joba nisu zelena.

## Release metadata

CI čita `VERSION`, generira bilješke iz odgovarajućeg `CHANGELOG.md` odjeljka i zahtijeva da izlaz nije prazan. Time se sprječava objava verzije bez dokumentirane povijesti promjena.
