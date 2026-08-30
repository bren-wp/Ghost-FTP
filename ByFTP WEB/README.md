# ByFTP WEB 1.7.0

ByFTP WEB je samostalni, višekorisnički FTP/FTPS/SFTP file manager za PHP shared hosting. Smješten je u zasebnoj mapi unutar glavnog ByFTP repozitorija i prati isti release broj kao Windows, Linux, Android i iOS klijenti.

## Što je uključeno

- FTP i FTPS preko PHP `ext-ftp`
- SFTP preko PHP `ext-ssh2`, uključujući password i public-key autentikaciju
- sigurni spremljeni profili po korisniku; username/password/ključni materijal šifriraju se Sodium secretboxom ili AES-256-GCM/OpenSSL fallbackom
- višekorisnički računi, admin upravljanje korisnicima i izolirani workspace po korisniku
- pregled direktorija, upload datoteka i mapa, download, create/rename/delete, copy/move/duplicate, CHMOD, editor, search, favorites i batch operacije
- ZIP download/kreiranje/raspakiranje kada je `ext-zip` dostupan
- PWA instalacija bez cacheiranja autentificiranih stranica, API-ja i download/preview odgovora
- responzivni desktop/mobitel UI bez CDN-a i bez obveznog Node/Docker/cron runtimea

## Sigurnosne promjene u 1.7.0

Ova verzija prelazi na fail-closed unos i putanje. Remote putanje više ne pretvaraju `\` u `/`, ne prihvaćaju `a/../b`, `./` ni dvostruke separatore. Host, port, timeout i SFTP fingerprint provjeravaju se u izvornom obliku prije normalizacije; unos poput `22junk`, rubni razmaci ili CR/LF više se ne pretvaraju u valjanu vrijednost. FTP/SFTP password i privatni ključ uklanjaju se iz aktivnog client profila odmah nakon autentikacije.

SSRF zaštita je uključena prema zadanim postavkama: privatne, loopback i rezervirane IP adrese nisu dopuštene dok administrator eksplicitno ne uključi private-host pristup. DNS se razrješava jednom i protokol se spaja na provjerenu adresu kako bi se smanjio DNS-rebinding/TOCTOU prostor.

Aplikacija koristi `SameSite=Strict`, HttpOnly session cookie, CSRF tokene, `Sec-Fetch-Site` zaštitu za cross-site POST, CSP, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, no-store za osjetljive odgovore i `X-Robots-Tag`/`robots.txt` zabranu indeksiranja.

## Minimalni zahtjevi

- PHP 8.1+
- `ext-ftp` za FTP/FTPS
- Sodium ili OpenSSL za šifrirano spremanje vjerodajnica
- zapisiv `storage/` direktorij
- `ext-ssh2` za SFTP
- `ext-zip` za ZIP funkcije
- HTTPS za produkciju

## Instalacija na shared hosting

1. Kopiraj sadržaj mape `ByFTP WEB/` na odabranu web putanju.
2. Provjeri da PHP može zapisivati u `storage/`, `storage/tmp/`, `storage/logs/` i `storage/users/`.
3. Otvori `/setup` i izradi administratorski račun.
4. Nakon prijave otvori **Diagnostics** i provjeri FTP/SFTP/ZIP/crypto mogućnosti hostinga.
5. U produkciji koristi HTTPS. `.htaccess` već zabranjuje direktan pristup `app/`, `storage/`, README-u, Composer metapodacima i JSON/backup/lock datotekama.

Ako Apache `mod_rewrite` nije dostupan, PHP datoteke i dalje mogu raditi direktno (`login.php`, `index.php`, `api.php`), ali clean URL-ovi neće biti aktivni.

## Verzija i cache

`VERSION` u ovoj mapi jedini je kanonski ByFTP WEB broj izdanja. `app/bootstrap.php` ga učitava pri pokretanju, a repository audit provjerava da je usklađen s root `VERSION`. Service Worker cache koristi isti release broj i automatski uklanja prethodni `byftp-static-*` cache.

## Legacy podaci

Migracija stare single-admin web instalacije ostaje podržana bez zadržavanja stare verzijske numeracije u aktivnom sučelju. Ako postoji stari `password_hash` i legacy profile/preferences JSON, uspješna administratorska prijava prebacuje podatke u izolirani korisnički workspace. Backup datoteke ostaju fail-closed zaštićene kroz `storage/.htaccess`.

## Testiranje

Iz mape `ByFTP WEB/`:

```bash
find . -name '*.php' -print0 | xargs -0 -n1 php -l
node --check assets/js/api.js
node --check assets/js/utils.js
node --check assets/js/settings.js
node --check assets/js/pwa.js
node --check assets/js/app.js
node --check service-worker.js
php tests/unit.php
```

Glavni ByFTP CI dodatno pokreće repository-wide audit i `scripts/audit_web.py`, koji provjerava version binding, zabranjene legacy verzijske reference, osjetljive PWA cache granice, fail-closed remote path/host input i credential-lifetime markere.

## Ograničenja web modela

Browser/PHP shared-hosting aplikacija ne može imati isti neograničeni lokalni filesystem pristup niti trajni background socket kao native FileZilla/WinSCP tip klijenta. Folder upload koristi browser file/directory API, a veliki transferi ovise o PHP i hosting limitima. ByFTP WEB namjerno nema daemon, cron ni obvezni vanjski backend.
