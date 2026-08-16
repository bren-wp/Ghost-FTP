# ByFTP — plan razvoja

## Završeno do 2.14

- izvorni Win32 desktop bez browser/localhost sloja
- FTP, eksplicitni/implicitni FTPS i SFTP
- DPAPI profili i session-scoped SFTP trust
- skupni transferi i cijela stabla mapa
- transfer queue s paralelizmom, transient retryjem i preskakanjem postojećih datoteka
- symlink/junction/reparse zaštite i transakcijski staging
- connection-generation i cross-server retry izolacija
- duboke event snimke i veliki queue/list performance regresijski testovi
- reproducibilni PNG/ICO brand resursi
- uređeni `docs/` sustav i hrvatski kao jedini korisnički jezik
- kompletni GitHub Release + Source/Windows ZIP + Package + provenance

## Sljedeći prioriteti

1. još dublji runtime smoke-test matriks za različite FTP/FTPS/SFTP poslužitelje
2. kvalitetniji napredak velikih transfera uz ograničen event churn
3. dodatna optimizacija memorije za vrlo velike rekurzivne planove
4. proširena pristupačnost tipkovnicom i bolji fokusni feedback u Win32 UI-u
5. produkcijski Authenticode potpis kada je Brendigo signing identitet dostupan
6. kontrolirani recovery od mrežnog prekida bez slabljenja endpoint identity zaštite
7. daljnja automatizacija release smoke-provjera bez telemetrije i vanjskih runtime API-ja

Svaka nova funkcija mora očuvati privatnost, hrvatski korisnički sadržaj, tipizirani in-process API i postojeće sigurnosne granice.
