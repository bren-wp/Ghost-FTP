# ByFTP — izdavanje na GitHubu

Kanonski workflow je `.github/workflows/release.yml`, a broj verzije dolazi isključivo iz root datoteke `VERSION`.

## Platform buildovi

Release workflow prvo ponovno gradi tri platform grupe:

1. **Windows** — x64 i x86 Setup/Portable + provjereni ZIP za svaku arhitekturu
2. **Linux** — DEB za amd64, arm64 i i386
3. **macOS** — Universal Intel+Apple Silicon PKG

Svaki job sprema samo platform-specific javne pakete kao privremeni Actions artefakt. Završni `publish` job preuzima sve tri grupe i tek tada smije stvarati GitHub Release.

## Windows ZIP

Za x64 i x86 zasebno:

- Setup i Portable ulaze u bundle
- kompletna `docs/*.md` dokumentacija ide u `Dokumentacija/`
- `RELEASE-NOTES.txt`, `BUILD-METADATA.txt`, README, CHANGELOG i LICENSE ulaze u bundle
- generira se rekurzivni `BUNDLE-SHA256.txt`
- `verify_bundle.py --arch <x64|x86>` ponovno otvara konačni ZIP bez ekstrakcije na disk
- traversal, absolute/drive putanje, duplikati, encrypted entries, nepotpun manifest i hash mismatch zaustavljaju release
- standalone Uninstaller i verification report ne smiju biti u javnom ZIP-u

## Završni javni asseti

ByFTP 2.16 release ugovor ima točno 13 custom asseta:

- Windows x64: Setup EXE, Portable EXE, ZIP
- Windows x86: Setup EXE, Portable EXE, ZIP
- Linux: amd64 DEB, arm64 DEB, i386 DEB
- macOS: Universal PKG
- `SHA256.txt`
- `RELEASE-NOTES.txt`
- `BUILD-METADATA.txt`

Ne objavljuju se kao custom asseti:

- `verification.txt`
- `ByFTP-<verzija>-Source.zip`
- `ByFTP-<verzija>-Uninstall-*.exe`

Windows uninstaller ostaje interni dio Setup payload-a.

GitHub automatski generira vlastite `Source code (zip)` i `Source code (tar.gz)` poveznice za tag. To nije ByFTP custom asset i workflow ih ne može ukloniti.

## Migracija v2.15.0

Prvi 2.16 publish job prije nove objave provjerava v2.15.0 i uklanja tri stara custom asseta ako postoje:

- `verification.txt`
- `ByFTP-2.15.0-Source.zip`
- `ByFTP-2.15.0-Uninstall-x64.exe`

Nakon brisanja ponovno čita release i zahtijeva da nijedan od njih više nije prisutan. Tek tada nastavlja s 2.16 objavom.

## Siguran rerun

`scripts/publish_release.ps1` ostaje fail-closed i idempotentan:

- tag se razrješava do stvarnog Git commita i mora odgovarati release SHA-u
- postojeći asset uspoređuje se po nazivu, veličini i GitHub SHA-256 digestu
- identičan asset ostaje netaknut
- nedostajući asset može se dopuniti
- isti naziv s drugačijim sadržajem zaustavlja izdanje
- neočekivani asset zaustavlja izdanje
- nakon svakog uploada ponovno se zahtijeva točan kompletan asset set

Ne koristi se slijepi `--clobber`.

## Metapodaci

`RELEASE-NOTES.txt` generira se iz odgovarajućeg CHANGELOG odjeljka. `BUILD-METADATA.txt` bilježi verziju, commit/ref, platforme i Actions run. `SHA256.txt` pokriva javne platform pakete i zajedničke metapodatke.

## GitHub Package

`ByFTP.Windows` package ostaje dodatni distribucijski kanal samo za Windows. Sadrži x64+x86 Setup/Portable/ZIP, SHA256, release notes, build metadata, licencu i dokumentaciju; ne sadrži standalone Uninstaller ni interni verification report.

## Potpisivanje

Workflow ne fabricira publisher identitet. Windows Authenticode zahtijeva stvarni Brendigo code-signing certifikat. macOS Developer ID/notarizacija zahtijeva stvarni Apple certifikat i odgovarajuće secrets. Bez njih paketi se objavljuju kao provjereni, ali nepotpisani, uz jasnu napomenu.
