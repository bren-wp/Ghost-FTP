# Povijest promjena

Aktivna javna ByFTP release linija počinje s verzijom **1.0.0**. Starija razvojna i 2.x povijest ostaje dostupna kroz Git commit povijest repozitorija, ali nije dio aktualnog javnog release kataloga.

## 1.0.0 — Nova stabilna client-suite linija

- javna verzijska linija resetirana je na 1.0.0; buduća funkcionalna izdanja koriste 1.1.0, 1.2.0 i dalje, uz patch verzije kada je potreban hitan kompatibilni/sigurnosni popravak
- uvedena je odvojena ByFTP obitelj: All-in-One, FTP Client, SFTP Client, SSH Client i S3 Client
- FTP Client dopušta samo FTP, eksplicitni FTPS i implicitni FTPS; SFTP Client dopušta samo SFTP
- FTP i SFTP klijenti koriste zaseban process identity i zasebno profil/state spremište kako se profili i vjerodajnice različitih proizvoda ne bi miješali
- zajednički startup/AskPass hardening premješten je u `internal/appstart` kako zasebni EXE-ovi ne bi dobili divergentne kopije sigurnosne logike
- zadržan je ByFTP All-in-One radi kompatibilnosti i dvopanelnog FTP/FTPS/SFTP workflowa
- dodan je zasebni SSH Client koji koristi sistemski OpenSSH s privatnim ByFTP configom i `known_hosts` datotekom
- SSH Client blokira implicitni agent, ProxyCommand/ProxyJump, forwarding, GSSAPI, PKCS#11 i default identitete; bez eksplicitnog ključa koristi `IdentityFile none`
- SSH password, keyboard-interactive/MFA i passphrase unose se izravno u OpenSSH terminal, pa ByFTP ne mora presretati ili spremati te tajne
- dodan je zasebni S3 Client s vlastitom Go standard-library AWS Signature Version 4 implementacijom, bez AWS SDK-a i novih Go modula
- S3 SigV4 koristi AWS canonical URI/query enkodiranje, uključujući `%20` za razmak, `%2B` za plus i sortiranje query parametara nakon enkodiranja
- S3 podržava HTTPS endpoint, a plaintext HTTP je dopušten samo za lokalni loopback testni/S3-kompatibilni servis
- S3 core podržava access key, secret key i temporary session token; tajne ostaju samo u memoriji aktivnog procesa
- S3 ListObjectsV2 podržava prefix/delimiter prikaz i continuation-token paginaciju
- S3 upload/download su streaming; download koristi privatni `.part` staging i `RenameNoReplace` kako postojeća lokalna datoteka ne bi bila tiho prepisana
- S3 single `PutObject` je fail-closed ograničen na 5 GB; veći objekt vraća jasnu poruku dok se ne implementira zaseban multipart transfer put
- S3 rename koristi server-side copy + delete, prvo provjerava postoji li odredište i dokumentira da S3 nema atomski rename
- produkcijski Windows build sada stvara i verificira točno 12 javnih EXE datoteka: ByFTP Setup/Portable x64+x86 te FTP/SFTP/SSH/S3 Portable x64+x86
- interna komponenta uklanjanja i interni build dokazi više nisu javni Release asseti
- GitHub Release 1.x sadrži samo ugovorenih 12 EXE datoteka; GitHub automatski prikazuje standardne Source code ZIP/TAR.GZ poveznice
- jednokratni 1.0.0 release reset briše stare GitHub Release zapise i stare verzijske `v*` tagove bez prepisivanja Git commit povijesti
- uvedeno je pet novih GitHub Packages ID-jeva: `ByFTP.Suite`, `ByFTP.FTP.Client`, `ByFTP.SFTP.Client`, `ByFTP.SSH.Client` i `ByFTP.S3.Client`
- svaki budući release mora objaviti istu verziju u svih pet GitHub Packages paketa; završni workflow provjerava njihovo postojanje preko GitHub API-ja
- stari `ByFTP.Windows` 2.x package ID više se ne koristi za novu 1.x liniju kako semver 2.x ne bi ostao prikazan kao noviji od 1.0.0
- release workflow pojednostavljen je na obavezni quality/race/security gate, Windows 12-EXE build, Linux runtime smoke, macOS runtime smoke te jedan serijalizirani publisher
- produkcijski build i dalje zahtijeva stvarno isključenu Go telemetry postavku (`go telemetry off`), Go 1.26.5+ i offline Go dependency način
- Windows PE verifier sada razlikuje GUI i Console subsystem, ali za sve EXE-ove zahtijeva ASLR/NX/PE resurse i odsutnost poznatih telemetry vendor markera
- detaljni README i dokumentacija prepisani su za novu obitelj klijenata, novi 1.x release ugovor, standardni Windows removal lifecycle, GitHub Packages i produkcijski connect model
- WinSCP se koristi samo kao funkcionalna/UX referenca; njegov GPLv3 izvorni kod nije kopiran u ByFTP i ByFTP nije WinSCP fork
