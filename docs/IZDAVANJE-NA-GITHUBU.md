# ByFTP — izdavanje na GitHubu

Kanonski workflow je `.github/workflows/release.yml`, a broj verzije dolazi isključivo iz root datoteke `VERSION`.

## Osnovno pravilo

Javno izdanje ne nastaje ručnim preslagivanjem datoteka. Release workflow mora ponovno izvesti produkcijske provjere, ponovno izgraditi sve podržane platforme, provjeriti staging, generirati zajedničke metapodatke i tek tada pozvati centralni publisher.

## Okidač izdanja

Automatsko izdanje pokreće se kada se `VERSION` promijeni na `main` grani. Dostupan je i `workflow_dispatch` za kontrolirani idempotentni rerun iste kanonske verzije.

Tag koji publisher izradi **ne pokreće novi release workflow**. Time se uklanja mogućnost da isti release sam sebi pokrene drugi paralelni publisher.

Svi release runovi koriste jednu concurrency grupu `byftp-release` uz `cancel-in-progress: false`, pa se dva release procesa ne izvršavaju paralelno.

## Produkcijski gateovi

Prije `publish` joba moraju uspjeti četiri neovisna joba.

### 1. Quality

Quality job na Linux runneru izvršava:

- stvarno `go telemetry off` i provjeru da `go telemetry` vraća `off`
- provjeru slikovnih resursa
- hrvatski audit
- version audit
- docs audit
- security audit
- privacy audit
- release audit
- Python release regresije
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- probno generiranje release bilješki iz aktualnog CHANGELOG odjeljka

Release se ne može objaviti samo zato što su se platformni binariji uspjeli kompilirati.

### 2. Windows

Windows job:

- eksplicitno gasi Go toolchain telemetriju
- pokreće kanonski `BUILD-WINDOWS.ps1`
- gradi x64 i x86 Setup i Portable
- provjerava PE format, resurse, manifest i mitigacije
- izrađuje zasebni Windows ZIP za svaku arhitekturu
- dodaje README, CHANGELOG, LICENSE, kompletnu Markdown dokumentaciju, release bilješke i build metapodatke
- generira rekurzivni `BUNDLE-SHA256.txt`
- ponovno otvara konačni ZIP s `verify_bundle.py` i provjerava sadržaj bez ekstrakcije nepouzdanih putanja na disk

### 3. Linux

Linux job:

- eksplicitno gasi Go toolchain telemetriju
- izvršava unit/process smoke testove i `go vet`
- gradi DEB za amd64, arm64 i i386
- preko `dpkg-deb` potvrđuje package ID, verziju i arhitekturu svakog paketa

### 4. macOS

macOS job:

- eksplicitno gasi Go toolchain telemetriju
- izvršava unit/process smoke testove i `go vet` na stvarnom macOS runneru
- gradi Intel i Apple Silicon binarij
- spaja ih s `lipo` u Universal binarij
- izrađuje ByFTP.app resurse i Universal PKG
- širi završni PKG s `pkgutil --expand` i potvrđuje strukturu prije uploada artefakta

## Go toolchain telemetrija

`GOTELEMETRY` je read-only Go environment vrijednost i nije valjan način za gašenje telemetrije običnim OS env postavljanjem. GitHub Actions zato izvršava:

```text
go telemetry off
```

i zatim provjerava rezultat prije testova ili builda.

Produkcijske build skripte dodatno fail-closed odbijaju rad ako aktualni `go telemetry` način nije `off`. Lokalne skripte namjerno ne mijenjaju globalnu Go postavku bez znanja korisnika.

## Javni platformni staging

`publish` job prvo preuzima tri platformna Actions artefakta u `release/` i prije generiranja zajedničkih metapodataka zahtijeva **točno 10 platformskih datoteka**:

1. Windows x64 Setup
2. Windows x64 Portable
3. Windows x64 ZIP
4. Windows x86 Setup
5. Windows x86 Portable
6. Windows x86 ZIP
7. Linux amd64 DEB
8. Linux arm64 DEB
9. Linux i386 DEB
10. macOS Universal PKG

Nedostajući, dodatni ili pogrešno imenovani paket zaustavlja release prije stvaranja `SHA256.txt`.

## Završni javni asseti

Nakon dodavanja zajedničkih release metapodataka, izdanje ima točno 13 custom asseta:

- šest Windows paketa
- tri Linux DEB paketa
- jedan macOS Universal PKG
- `SHA256.txt`
- `RELEASE-NOTES.txt`
- `BUILD-METADATA.txt`

Tehničke datoteke koje služe samo build/verifikacijskom sloju ne ulaze u javni asset allowlist.

GitHub za svaki tag može prikazati vlastite ugrađene source archive poveznice. One nisu ByFTP build artefakti i ne ulaze u ovaj custom asset ugovor.

## SHA-256 i metapodaci

`RELEASE-NOTES.txt` generira se iz točno odgovarajućeg `CHANGELOG.md` odjeljka.

`BUILD-METADATA.txt` bilježi:

- verziju
- release commit
- izvorni ref
- podržane platforme/arhitekture
- činjenicu da je release quality gate prošao
- GitHub Actions run ID i attempt

`SHA256.txt` obuhvaća svih 10 platformskih paketa, release bilješke i build metapodatke.

## Centralni publisher

`scripts/publish_release.ps1` ostaje jedino mjesto koje smije stvarati ili nadopunjavati GitHub Release.

Publisher je fail-closed:

- tag se razrješava do stvarnog Git commita
- tag mora odgovarati release SHA-u
- postojeći asset uspoređuje se po nazivu, veličini i GitHub SHA-256 digestu
- identičan asset ostaje netaknut
- nedostajući asset može se nadopuniti
- isti naziv s drugačijim sadržajem zaustavlja izdanje
- neočekivani asset zaustavlja izdanje
- nakon uploada ponovno se čita GitHub stanje i zahtijeva točan završni skup

Slijepi overwrite nije dopušten.

## Rerun nakon djelomičnog pada

Ako platformni build ili upload djelomično padne, `workflow_dispatch` može ponovno pokrenuti kanonsku verziju.

Siguran rerun:

1. ponovno izvršava quality i sve platformne buildove;
2. ponovno generira lokalne očekivane assete;
3. uspoređuje već objavljene assete po veličini i digestu;
4. nadopunjuje samo ono što nedostaje;
5. odbija svaki mismatch.

## Kompletni Actions artefakt

Nakon uspješne objave `release/*` sprema se i kao verzionirani GitHub Actions artefakt s duljim retentionom. To nije zamjena za GitHub Release, nego build/provenance dokaz iste objave.

## GitHub Windows Package

`ByFTP.Windows` ostaje dodatni distribucijski kanal samo za Windows. Paket sadrži javne x64/x86 Windows distribucijske datoteke, checksumove, release bilješke, build metapodatke, licencu i dokumentaciju.

## Potpisivanje

Workflow ne fabricira publisher identitet.

- Windows Authenticode zahtijeva stvarni Brendigo code-signing certifikat.
- macOS Developer ID/notarizacija zahtijeva stvarni Apple certifikat i odgovarajuće secrets.

Bez tih identiteta release smije biti tehnički provjeren i hash-verificiran, ali dokumentacija ne smije tvrditi Verified Publisher, Developer ID ili notarizaciju.

## Kontrolna lista

Prije promjene `VERSION` provjerite [PROVJERA-IZDANJA.md](PROVJERA-IZDANJA.md). Za detalje platformnih buildova pogledajte [TESTIRANJE.md](TESTIRANJE.md), a za sigurnosne granice [SIGURNOST.md](SIGURNOST.md).
