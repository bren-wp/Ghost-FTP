# ByFTP — Windows potpisivanje

ByFTP build namjerno ne generira lažni ili self-signed „verified publisher” identitet.

Za javnu produkcijsku distribuciju:

1. Nabavite i provjerite stvarni Brendigo code-signing identitet.
2. Potpišite Portable, Setup i Uninstaller nakon završne PE/resource obrade.
3. Koristite SHA-256 i pouzdani timestamp prema pravilima izdavatelja certifikata/potpisne usluge.
4. Na čistom Windows računalu provjerite potpis naredbom `signtool verify /pa /v`.
5. Nakon potpisivanja ponovno generirajte i objavite SHA-256 manifest.

Privatni certifikat, token, lozinka ili signing secret ne smije biti u repozitoriju, Source ZIP-u ili javnom Actions artefaktu.
