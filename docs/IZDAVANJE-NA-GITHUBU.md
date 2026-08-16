# ByFTP — izdavanje na GitHubu

Službena automatizacija nalazi se u `.github/workflows/release.yml`.

## Tijek

1. `VERSION` određuje kanonski broj izdanja.
2. CI provjerava kod, race, vet, privatnost, hrvatski sadržaj, brand resurse i release metadata.
3. Windows job pokreće isti `BUILD-WINDOWS.ps1` koji se koristi za produkciju.
4. Release workflow izrađuje Portable, Setup i Uninstaller.
5. Izrađuje se uređeni Windows ZIP s podmapom `Dokumentacija/` i rekurzivnim bundle checksumovima.
6. Source ZIP nastaje iz `git archive HEAD`.
7. Generiraju se hrvatske `RELEASE-NOTES.txt`, `SHA256.txt`, `verification.txt` i `BUILD-METADATA.txt`.
8. GitHub Release veže se uz točan commit.
9. Isti skup provjerenih artefakata ide u verzionirani Actions artefakt.
10. Izrađuje se i objavljuje `ByFTP.Windows` GitHub Package.

GitHub Releases je preporučeni kanal za krajnje Windows korisnike. GitHub Packages je dodatni paketni/arhivski kanal.

Šira javna distribucija treba koristiti stvarni Brendigo Authenticode identitet; repozitorij ne generira lažni publisher certifikat.
