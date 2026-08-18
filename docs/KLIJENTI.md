# ByFTP klijenti

ByFTP 1.x distribuira jednu zajedničku jezgru kroz pet jasno odvojenih aplikacijskih ulaza. Razdvajanje nije kopiranje koda: sigurnosne primitive, FTP/SFTP adapteri, parseri i transfer engine ostaju zajednički gdje je to ispravno, dok svaki proizvod ima vlastiti procesni identitet i samo svoje dopuštene protokole.

## ByFTP All-in-One

Glavni Windows file manager za korisnika koji želi FTP, FTPS i SFTP u jednom programu.

- dvopanelni Commander-style raspored;
- lokalni panel lijevo, udaljeni desno;
- profili;
- transfer queue;
- upload/download;
- mkdir/rename/delete;
- CHMOD gdje ga protokol podržava;
- FTP, eksplicitni FTPS, implicitni FTPS i SFTP.

All-in-One zadržava postojeći profilni prostor radi kompatibilnosti s ranijim instalacijama.

## ByFTP FTP Client

Odvojeni file manager samo za FTP obitelj.

Dopušteni protokoli:

- FTP — port 21 prema zadanoj vrijednosti;
- eksplicitni FTPS — port 21 prema zadanoj vrijednosti;
- implicitni FTPS — port 990 prema zadanoj vrijednosti.

SFTP se u ovom EXE-u ne prikazuje kao izbor. Profil/state prostor je odvojen od All-in-One i SFTP Clienta.

FTP Client koristi zajednički `CurlFTP` adapter i iste transfer/security provjere kao All-in-One. Lozinka ne ide u argumente procesa.

## ByFTP SFTP Client

Odvojeni file manager samo za SFTP.

- protokol je fiksiran na SFTP;
- zadani port 22;
- podrška za password i privatni ključ na Windowsu;
- fail-closed AskPass prompt odabir;
- host-key fingerprint potvrda;
- privatni session config i known_hosts;
- agent, ProxyCommand, ProxyJump, PKCS#11 i forwarding helperi blokirani su na file-transfer putu.

SFTP Client ima vlastiti profil/state prostor. Njegov EXE ujedno može raditi kao kontrolirani Windows AskPass helper samo za vlastitu SFTP sesiju; startup provjerava path, token i roditeljski proces prije izdavanja tajne.

## ByFTP SSH Client

SSH je namjerno odvojen od file-transfer `Session` sučelja. Puni terminal ima drugačiji lifecycle i interaktivnu autentikaciju od prijenosa datoteka.

SSH Client:

- koristi sistemski OpenSSH `ssh`;
- stvara privatni kratkotrajni config;
- koristi ByFTP `known_hosts`;
- `StrictHostKeyChecking ask`;
- `IdentityAgent none`;
- `ProxyCommand none` / `ProxyJump none`;
- `ClearAllForwardings yes`;
- `ForwardAgent no` / `ForwardX11 no`;
- `GSSAPIAuthentication no`;
- `IdentitiesOnly yes`;
- eksplicitni privatni ključ ili `IdentityFile none`.

Lozinku, MFA/keyboard-interactive kod i passphrase traži OpenSSH izravno u terminalu. ByFTP ih ne kopira u vlastiti profilni store.

## ByFTP S3 Client

S3 nije filesystem protokol i zato se ne ugurava u FTP/SFTP `Session` interface. Ima vlastiti object-storage core.

Konfiguracija:

- HTTPS endpoint;
- regija;
- bucket;
- access key;
- secret key;
- opcionalni temporary session token u coreu.

Operacije 1.0.0:

- ListObjectsV2;
- delimiter/prefix navigacija;
- streaming PutObject do 5 GB;
- streaming GetObject;
- DeleteObject;
- prefix marker (`mkdir` UX);
- rename kao server-side CopyObject + DeleteObject, uz provjeru postoji li odredište.

S3 koristi vlastitu AWS Signature Version 4 implementaciju iz Go standardne biblioteke. AWS SDK nije runtime/build ovisnost.

S3 nema atomski rename. Ako copy uspije, a delete izvora ne uspije, mogu privremeno postojati i stari i novi objekt; ByFTP takav slučaj vraća kao pogrešku, ne kao lažni puni uspjeh.

## Zašto zajednička jezgra?

Dupliranje čitavog FTP/SFTP koda u svaki EXE stvorilo bi četiri ili pet kopija istih sigurnosnih grešaka. Zato se dijeli ono što pripada istoj domeni:

- `internal/security` — putanje, tajne, no-follow granice;
- `internal/remote` — FTP/FTPS/SFTP adapteri;
- `internal/transfer` — queue i transfer lifecycle;
- `internal/profilebinding` — identitet profila/vjerodajnice;
- `internal/appstart` — process hardening i file-client AskPass startup;
- `internal/clientmode` — product/protocol allowlist i zasebni process identity.

SSH i S3 imaju vlastite adapter slojeve jer njihove semantike nisu iste kao FTP/SFTP.

## WinSCP kao referenca

WinSCP dokumentacija korištena je za razumijevanje očekivanog file-manager workflowa, Commander/Explorer obrazaca i funkcionalne razlike FTP/SFTP/SSH/S3. WinSCP repozitorij je GPLv3; njegov izvorni kod nije kopiran u ByFTP. ByFTP nije fork WinSCP-a.
