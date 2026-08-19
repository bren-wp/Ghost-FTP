# Obavijesti trećih strana

**ByFTP želi biti jasan o alatima na koje se oslanja — bez skrivanja mrežnih komponenti iza vlastitog brenda.**

## Mrežni alati

ByFTP koristi provjerene sistemske alate za određene protokole:

- `curl` za FTP/FTPS operacije;
- OpenSSH alate za SFTP.

ByFTP ih ne predstavlja kao vlastite komponente. Njihove licence i autorska prava pripadaju njihovim autorima i projektima.

## Go standardna biblioteka

Glavni ByFTP engine pisan je u Go-u i projekt namjerno ne koristi vanjske Go module u runtime dependency grafu. CI to provjerava kao dio privacy/supply-chain politike.

## Operativni sustav

ByFTP koristi platformske API-je operativnog sustava, uključujući Windows DPAPI i datotečne primitive potrebne za sigurniji lifecycle.

## Zašto je ova transparentnost važna

Korisnik treba znati koji dio sustava obavlja mrežni protokol, koji dio pripada ByFTP-u i koji dio dolazi iz operativnog sustava.

**Manje skrivenih slojeva znači lakšu provjeru, jasniju odgovornost i bolju kontrolu nad proizvodom koji koristite.**
