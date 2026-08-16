<p align="center">
  <img src="docs/slike/byftp-zaglavlje.png" alt="ByFTP — siguran prijenos datoteka" width="900">
</p>

<p align="center">
  <strong>Brz, privatan i izvorni FTP / FTPS / SFTP klijent za Windows.</strong><br>
  ByFTP je fokusirani desktop alat tvrtke Brendigo za razvojne timove, agencije, administratore hostinga i sve koji žele izravan pristup poslužitelju bez browser sučelja, telemetrije ili obaveznog cloud računa.
</p>

<p align="center">
  <a href="https://github.com/bren-wp/by-ftp/releases"><strong>Izdanja</strong></a> ·
  <a href="https://github.com/users/bren-wp/packages?repo_name=by-ftp"><strong>Paketi</strong></a> ·
  <a href="LICENSE"><strong>Licenca</strong></a> ·
  <a href="docs/PRIVATNOST.md"><strong>Privatnost</strong></a> ·
  <a href="docs/SIGURNOST.md"><strong>Sigurnost</strong></a>
</p>

<p align="center">
  <a href="../../actions/workflows/ci.yml"><img alt="CI" src="../../actions/workflows/ci.yml/badge.svg"></a>
</p>

## Prenosite datoteke. Zadržite kontrolu.

ByFTP je izvorni Windows x64 klijent za **FTP, FTPS i SFTP**. Spaja poznati dvopanelni upravitelj datotekama s učvršćenim transfer engineom, višestrukim operacijama i arhitekturom usmjerenom na privatnost.

Nema ugrađeni preglednik, localhost web nadzornu ploču, analitički SDK, oglašavanje ni obavezni korisnički račun. ByFTP se povezuje s poslužiteljem **koji korisnik odabere**, a aplikacijske podatke drži lokalno u Windows korisničkom profilu.

**Trenutačno izdanje: 2.14.0**

## Preuzimanje

### [GitHub izdanja](https://github.com/bren-wp/by-ftp/releases)

To je preporučeni kanal za Windows korisnike. Službeno izdanje sadrži:

- `ByFTP-<verzija>-Setup-x64.exe` — instalacijski program
- `ByFTP-<verzija>-Portable-x64.exe` — prijenosna verzija bez instalacije
- `ByFTP-<verzija>-Uninstall-x64.exe` — samostalni program za uklanjanje
- `ByFTP-<verzija>-Windows-x64.zip` — kompletan Windows paket
- `ByFTP-<verzija>-Source.zip` — točna snimka praćenog izvornog koda
- `SHA256.txt` — SHA-256 kontrolne vrijednosti
- `verification.txt` — PE i sigurnosna provjera
- `RELEASE-NOTES.txt` — hrvatske bilješke iz odgovarajućeg CHANGELOG odjeljka
- `BUILD-METADATA.txt` — podrijetlo izvornog commita i build okruženja

### [GitHub paketi](https://github.com/users/bren-wp/packages?repo_name=by-ftp)

GitHub Packages služi kao dodatni paketni/arhivski kanal. Za uobičajenu instalaciju preporučuje se GitHub Releases.

> Službena su samo neizmijenjena izdanja koja Brendigo objavi kroz službeni ByFTP/Brendigo kanal.

## Mogućnosti

### Veze

- FTP
- FTPS — eksplicitni način
- FTPS — implicitni način
- SFTP
- autentikacija lozinkom
- autentikacija privatnim ključem uz zaporku ključa
- potvrda i pinning SFTP otiska ključa poslužitelja
- podesivo vrijeme čekanja pri spajanju od 5 do 60 sekundi
- lokalno spremljeni profili zaštićeni Windows DPAPI mehanizmom

### Upravitelj datotekama

- dvopanelni prikaz lokalnog računala i poslužitelja
- višestruki odabir uz Ctrl/Shift
- navigacija dvoklikom i brzi prijenos
- slanje/preuzimanje pojedinačnih datoteka ili cijelih stabala mapa
- stvaranje mapa, preimenovanje, brisanje i osvježavanje
- udaljene dozvole (CHMOD), uključujući skupni odabir
- Windows ikone vrsta datoteka i mapa
- lokalni prikaz ograničen na 50.000 stavki radi kontrolirane potrošnje memorije
- zaštita od traversal putanja, rezerviranih Windows naziva, symlinkova, junctiona i reparse-point izlaza

### Red prijenosa

- 1–8 paralelnih prijenosa
- pauziranje i nastavak reda
- skupno otkazivanje i ponavljanje
- opcionalno automatsko ponavljanje samo prolaznih mrežnih grešaka
- podesiva pauza između pokušaja
- način „preskoči postojeće datoteke”
- blokada ponavljanja posla prema drugom poslužitelju/računu
- ponovna validacija lokalnog korijena prije svakog pokušaja
- zaštita rekurzivnog slanja od kasne symlink/junction zamjene korijena
- indeksirana obrada događaja za velike redove
- `Očisti završene` uklanja završene poslove, event backing podatke i UI deduplikacijske ID-eve
- izolacija panic greške radnika kako pojedinačni prijenos ne bi srušio cijelu aplikaciju

## Što donosi 2.14.0

- događaji reda prijenosa sada se vraćaju kao duboke, neovisne snimke; vanjski potrošač više ne može mutacijom povratne vrijednosti promijeniti spremljenu event povijest
- privremene SFTP trust vjerodajnice brišu se nakon svake završene potvrde, greške ili otkaza i ostaju u memoriji samo dok korisnik stvarno odlučuje o novom ključu
- uklonjen je nedostižni cleanup prethodne sesije iz `Connect` puta
- popravljena je zastarjela hardkodirana verzija u `scripts/BUILD-LOCAL.sh`; svi buildovi sada čitaju jedini izvor istine iz `VERSION`
- potpuno obnovljeni PNG/ICO resursi; stari `build/icon.png` imao je oštećen IDAT CRC/bitstream
- dodan reproducibilni generator i CI provjera slikovnih resursa
- hrvatski je postavljen kao jedini korisnički jezik za UI, GitHub predloške, release bilješke, paketnu dokumentaciju i glavne projektne dokumente
- dokumentacija je premještena iz root direktorija u uređeni `docs/` indeks, a release bundle dokumentaciju slaže u vlastitu podmapu

## Privatnost

ByFTP namjerno nema:

- telemetriju i analitiku
- oglašivačke SDK-ove
- vanjsko automatsko slanje izvještaja o rušenju
- pozadinsku sinkronizaciju računa
- automatski update API
- browser/localhost upravljački server
- trajni runtime activity/error log

Normalan mrežni promet ByFTP-a namijenjen je **FTP/FTPS/SFTP poslužitelju koji odabere korisnik**. Windows i sigurnosni softver mogu zasebno provoditi DNS, firewall, antivirus/EDR ili druge sistemske funkcije izvan kontrole ByFTP-a.

Detalji: [Privatnost](docs/PRIVATNOST.md) i [Sigurnost](docs/SIGURNOST.md).

## Sigurnosne značajke

- SFTP host-key pinning vezan uz provjereni ključ i algoritam
- Windows curl/OpenSSH samo iz stvarnog System32 direktorija
- blokirani naslijeđeni proxy, SSH agent, ProxyJump/ProxyCommand i vanjski helperi
- lozinke i zaporke privatnog ključa ne ulaze u command-line argumente
- spremljene vjerodajnice ne dešifriraju se rano u connection manageru
- server-controlled lokalni nazivi prolaze sigurnu child-path validaciju
- zaštita od junction/symlink/reparse-point traversala
- transakcijski upload/download staging i rollback
- kriptografski nasumični interni staging nazivi
- sigurnije stvaranje ByFTP direktorija bez preusmjeravanja
- izolacija generationa aktivne veze za transfer batch
- blokada cross-server retryja
- SHA-256 provjera embedded instalacijskog payloada
- rollback binarija i Registry stanja pri neuspjeloj nadogradnji
- strogo ograničen uninstall put
- state safe-open provjerava stvarno otvoreni regularni objekt prije parsiranja
- event API u 2.14 vraća duboke kopije i ne izlaže internu event povijest mutaciji pozivatelja

## Zahtjevi

### Pokretanje

- Windows 10 ili Windows 11 x64
- sistemski `curl.exe` za FTP/FTPS
- Windows OpenSSH Client za SFTP

### Izgradnja iz izvornog koda

- Go **1.26.5+**
- Python 3
- Windows x64 za puni produkcijski build

Produkcijski build je namjerno offline:

```text
GOTOOLCHAIN=local
GOPROXY=off
GOSUMDB=off
GOTELEMETRY=off
```

Projekt nema vanjske Go module.

## Izgradnja

Kanonska verzija nalazi se isključivo u [`VERSION`](VERSION).

Na Windowsu:

```powershell
.\BUILD-WINDOWS.ps1
```

ili:

```cmd
BUILD-WINDOWS.cmd
```

Za lokalnu cross-build provjeru iz Unix/Linux okruženja:

```bash
./scripts/BUILD-LOCAL.sh
```

Build provjerava slikovne resurse, hrvatski korisnički sadržaj, privatnost, testove, `go vet`, PE resurse, sigurnosne mitigacije i SHA-256 vrijednosti.

## Provjere kvalitete

```text
go test ./...
go test -race ./...
go vet ./...
python scripts/generate_brand_assets.py --check
python scripts/audit_croatian.py
python scripts/audit_privacy.py
python scripts/release_notes.py --version 2.14.0 --output RELEASE-NOTES.test.txt
```

GitHub Actions dodatno izvršava puni Windows produkcijski build.

## Struktura repozitorija

```text
README.md              glavni hrvatski pregled projekta
LICENSE                vlasnička ByFTP licenca
CHANGELOG.md            povijest izdanja
VERSION                 jedini izvor release verzije
BUILD-WINDOWS.*         produkcijski Windows entrypointi
cmd/                    aplikacija, instalacija i uklanjanje
internal/               tipizirani runtime moduli
build/                  službeni PNG/ICO resursi aplikacije
docs/                   sva detaljna dokumentacija i slike
scripts/                build, audit, release i PE alati
.github/                CI, release workflow i hrvatski GitHub predlošci
```

Root direktorij namjerno ostaje kratak; detaljna dokumentacija više nije razbacana po vrhu repozitorija.

## Dokumentacija

- [Indeks dokumentacije](docs/README.md)
- [Arhitektura](docs/ARHITEKTURA.md)
- [Sigurnost](docs/SIGURNOST.md)
- [Privatnost](docs/PRIVATNOST.md)
- [Testiranje](docs/TESTIRANJE.md)
- [Provjera izdanja](docs/PROVJERA-IZDANJA.md)
- [Potpisivanje](docs/POTPISIVANJE.md)
- [Plan razvoja](docs/PLAN-RAZVOJA.md)
- [Podrška](docs/PODRSKA.md)
- [Doprinos](docs/DOPRINOS.md)
- [Obavijesti trećih strana](docs/OBAVIJESTI-TRECIH-STRANA.md)
- [Izdavanje na GitHubu](docs/IZDAVANJE-NA-GITHUBU.md)
- [Povijest promjena](CHANGELOG.md)

## Licenca

**Copyright © 2026 Brendigo. Sva prava pridržana.**

ByFTP je **vlasnički/source-available softver, nije open-source softver**. Objavljeni izvorni kod može se pregledavati radi transparentnosti, sigurnosne provjere i evaluacije, ali se bez prethodnog pisanog dopuštenja tvrtke Brendigo ne daje opće pravo izmjene, redistribucije, rebrandinga, prodaje, sublicenciranja, izrade izvedenica ili ponovne uporabe koda u drugom projektu.

Pročitajte cijelu [ByFTP vlasničku licencu](LICENSE).

## Sigurnosne prijave

Za osjetljive sigurnosne probleme slijedite [SIGURNOST.md](docs/SIGURNOST.md). U javni issue nikada ne stavljajte lozinke, privatne ključeve, stvarne produkcijske hostove, korisnička imena ili povjerljive podatke klijenata.

---

<p align="center">
  <a href="https://github.com/bren-wp/by-ftp/releases"><strong>Preuzmi ByFTP</strong></a><br><br>
  <strong>ByFTP</strong><br>
  Izvorni prijenos datoteka za Windows · Brendigo
</p>
