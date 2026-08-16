# ByFTP — izdavanje na GitHubu

Službena automatizacija nalazi se u `.github/workflows/release.yml`. Kanonski broj izdanja uvijek je u root datoteci `VERSION`.

## Tijek

1. `VERSION` određuje broj izdanja; ručni `workflow_dispatch` može opcionalno navesti isti broj, a prazno polje ponovno koristi `VERSION`.
2. CI provjerava kod, race, vet, privatnost, sigurnosne invarijante, hrvatski sadržaj, dokumentaciju, brand resurse, verzijsku konzistentnost i release ugovor.
3. Python release regresije provjeravaju ZIP verifier bez mreže i vanjskih paketa.
4. Windows job pokreće isti `BUILD-WINDOWS.ps1` koji se koristi za produkciju.
5. Release workflow izrađuje Portable, Setup i Uninstaller te generira hrvatske `RELEASE-NOTES.txt` i `BUILD-METADATA.txt`.
6. Izrađuje se uređeni Windows ZIP koji sadrži cijelu `docs/*.md` dokumentaciju i rekurzivni `BUNDLE-SHA256.txt`.
7. `scripts/verify_bundle.py` ponovno otvara konačni ZIP, bez raspakiravanja na disk, i provjerava putanje, duplikate, obavezne datoteke, potpunost manifesta i svaki SHA-256.
8. Source ZIP nastaje isključivo iz `git archive HEAD`.
9. Generira se release `SHA256.txt` za javne artefakte.
10. `scripts/publish_release.ps1` stvara novo izdanje ili sigurno dovršava postojeće djelomično izdanje.
11. Isti skup provjerenih artefakata ide u verzionirani Actions artefakt.
12. Izrađuje se i objavljuje `ByFTP.Windows` GitHub Package s release bilješkama i kompletnom Markdown dokumentacijom.

## Siguran rerun

Release job namjerno je idempotentan. Ponovno pokretanje nakon prekida ne koristi slijepi overwrite:

- ako release/tag već postoji, tag se razrješava do stvarnog Git commita i mora odgovarati `GITHUB_SHA`
- postojeći asset mora imati isti naziv, veličinu i GitHub `sha256:` digest kao lokalno ponovno izgrađeni artefakt
- potvrđeni postojeći asset ostaje netaknut
- nedostajući asset ponovno se uploadira
- neočekivani asset ili isti naziv s drugim sadržajem zaustavlja pipeline
- nakon uploada ponovno se čita GitHub Release i zahtijeva točan skup svih očekivanih asseta

Ovaj model omogućuje popravak djelomične objave bez automatskog `--clobber` prepisivanja već javno objavljenog binarnog sadržaja.

## Ugovor javnog izdanja

GitHub Release mora sadržavati:

- `ByFTP-<verzija>-Portable-x64.exe`
- `ByFTP-<verzija>-Setup-x64.exe`
- `ByFTP-<verzija>-Uninstall-x64.exe`
- `ByFTP-<verzija>-Windows-x64.zip`
- `ByFTP-<verzija>-Source.zip`
- `SHA256.txt`
- `verification.txt`
- `RELEASE-NOTES.txt`
- `BUILD-METADATA.txt`

GitHub Releases je preporučeni kanal za krajnje Windows korisnike. GitHub Packages je dodatni paketni/arhivski kanal.

Šira javna distribucija treba koristiti stvarni Brendigo Authenticode identitet; repozitorij ne generira lažni publisher certifikat.