# ByFTP GitHub Packages

ByFTP 1.x objavljuje GitHub Packages zajedno sa svakim službenim izdanjem. Package verzija **uvijek mora biti ista** kao kanonski `VERSION` i GitHub Release tag `v<verzija>`.

## Package linije

| GitHub Package | Sadržaj |
|---|---|
| `ByFTP.Suite` | All-in-One Setup x64/x86 + Portable x64/x86 |
| `ByFTP.FTP.Client` | FTP/FTPS Client Portable x64/x86 |
| `ByFTP.SFTP.Client` | SFTP Client Portable x64/x86 |
| `ByFTP.SSH.Client` | SSH Client Portable x64/x86 |
| `ByFTP.S3.Client` | S3 Client Portable x64/x86 |

Paketi se objavljuju na GitHub Packages NuGet registru vlasnika repozitorija. Ne predstavljaju zasebnu verzijsku liniju: `1.0.0` Release mora imati `1.0.0` svih pet paketa, `1.1.0` mora imati `1.1.0` svih pet paketa i tako dalje.

## Automatska objava

`.github/workflows/release.yml` nakon zelenih quality i platformskih build gateova:

1. objavi GitHub Release;
2. izgradi točno pet `.nupkg` datoteka preko `scripts/make_github_packages.ps1`;
3. objavi ih u GitHub Packages koristeći `GITHUB_TOKEN` s `packages: write` dopuštenjem;
4. preko GitHub API-ja provjeri da svaka package linija sadrži upravo aktualni `VERSION`;
5. zaustavlja cijelo izdanje ako nedostaje ijedan paket ili verzija.

`--skip-duplicate` služi samo za sigurno ponovno pokretanje istog release workflowa. Nova verzija se uvijek objavljuje kao nova package verzija.

## Pravila

- Package verzija dolazi isključivo iz `VERSION`.
- Ne upisivati verziju ručno u `.csproj` ili naziv paketa.
- Ne objavljivati package prije zelenog produkcijskog builda.
- Ne miješati EXE datoteke različitih klijenata u pogrešnu package liniju.
- `ByFTP.Suite` sadrži samo All-in-One distribuciju.
- Odvojeni klijenti imaju zasebne package ID-eve.
- GitHub Release i GitHub Packages moraju završiti u istom release jobu; neuspjeh package provjere znači neuspjelo izdanje.

## Ručna provjera

Na GitHubu otvorite **Packages** za korisnika/repozitorij i provjerite da svih pet package linija na vrhu imaju isti broj kao `VERSION` i najnoviji Release tag.

Za razvojnu liniju 1.x očekivani redoslijed je `1.0.0`, zatim funkcionalne verzije `1.1.0`, `1.2.0` itd.; kompatibilni hitni popravci mogu koristiti `1.0.1`, `1.1.1` i slično.
