# ByFTP — potpisivanje

ByFTP build namjerno **ne fabricira** Verified Publisher, Developer ID ili notarizacijski identitet.

## Windows Authenticode

Za javnu produkcijsku distribuciju kada postoji stvarni Brendigo code-signing identitet:

1. Potpišite x64 i x86 Portable, Setup i interni Uninstaller **nakon** završne PE/resource obrade.
2. Koristite SHA-256 i pouzdani timestamp prema pravilima izdavatelja certifikata/potpisne usluge.
3. Na čistom Windows računalu provjerite svaki binarij s `signtool verify /pa /v`.
4. Ponovno izgradite Windows ZIP-ove iz potpisanih javnih binarija.
5. Ponovno generirajte `BUNDLE-SHA256.txt` i završni `SHA256.txt`.

Standalone Uninstaller nije javni GitHub Release asset; potpis je i dalje važan jer je uninstaller ugrađen u Setup payload i završava u instaliranoj aplikaciji.

## macOS Developer ID i notarizacija

Službeni 2.16 macOS paket je Universal Intel+Apple Silicon PKG, ali bez stvarnog Brendigo Apple certifikata ne smije se tvrditi da je Developer ID potpisan ili notariziran.

Kada certifikat bude dostupan:

1. potpisati Universal CLI binarij i `ByFTP.app` sadržaj odgovarajućim Developer ID Application identitetom
2. provjeriti `codesign --verify --deep --strict --verbose=2`
3. izgraditi PKG i potpisati Developer ID Installer identitetom
4. poslati paket Apple notarizacijskoj usluzi
5. pričekati uspješnu notarizaciju i napraviti `stapler staple`
6. provjeriti `spctl`/`pkgutil` rezultat na čistom Macu
7. tek tada generirati završni SHA-256 i objaviti paket

## Tajne za potpisivanje

Privatni certifikat, token, P12, keychain password, Apple app-specific password/API key, Windows signing secret ili hardware-token credential **ne smije biti u repozitoriju, javnom Actions artefaktu ni GitHub Releaseu**.

CI bez signing identiteta mora jasno prijaviti `unsigned` umjesto stvaranja self-signed zamjene.
