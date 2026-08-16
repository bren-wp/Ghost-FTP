# ByFTP — plan razvoja

## Završeno do aktualne 2.14 linije

- izvorni Win32 desktop bez browser/localhost sloja
- FTP, eksplicitni/implicitni FTPS i SFTP
- DPAPI profili i session-scoped SFTP trust
- skupni transferi i cijela stabla mapa
- transfer queue s paralelizmom, transient retryjem i preskakanjem postojećih datoteka
- symlink/junction/reparse zaštite i transakcijski staging
- rekurzivni upload root runtime revalidacija
- download staging `Lstat`/regular-file/reparse validacija
- filesystem-root zaštita za no-follow rekurzivno brisanje
- autoritativni završni status transfera pri late-cancel raceu
- connection-generation i cross-server retry izolacija
- duboke event snimke i veliki queue/list performance regresijski testovi
- state/config safe-open s identitetom stvarno otvorene datoteke
- reproducibilni PNG/ICO brand resursi
- uređeni `docs/` sustav i hrvatski kao jedini korisnički jezik
- docs link/index audit bez ručnog verzioniranja naslova
- zasebni privacy, security i release quality gateovi
- provjera konačnog Windows ZIP-a i rekurzivnog bundle SHA-256 manifesta
- idempotentni GitHub Release rerun s tag/commit i asset-digest provjerom
- kompletni GitHub Release + Source/Windows ZIP + Package + provenance

## Sljedeći prioriteti

1. runtime smoke-test matriks za različite FTP/FTPS/SFTP poslužitelje i Windows 10/11 okruženja
2. kvalitetniji napredak vrlo velikih transfera uz ograničen event churn
3. dodatna optimizacija memorije za vrlo velike rekurzivne planove
4. proširena pristupačnost tipkovnicom i bolji fokusni feedback u Win32 UI-u
5. produkcijski Authenticode potpis kada je Brendigo signing identitet dostupan
6. kontrolirani recovery od mrežnog prekida bez slabljenja endpoint identity zaštite
7. dodatni release provenance/attestation sloj koji ne uvodi runtime telemetriju ni tajne u repozitorij
8. standardizirani interoperability smoke fixturei za FTP/FTPS/SFTP bez stvarnih vjerodajnica

Svaka nova funkcija mora očuvati privatnost, hrvatski korisnički sadržaj, tipizirani in-process API, postojeće sigurnosne granice i fail-closed release ugovor.