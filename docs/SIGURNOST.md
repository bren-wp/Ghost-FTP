# ByFTP — sigurnost

## Cilj

ByFTP je klijent kojem korisnik izričito daje endpoint i vjerodajnicu. Sigurnosni model zato prioritet daje pravilnom vezivanju tajni uz endpoint, strogoj SFTP host-key provjeri, no-follow datotečnim operacijama i fail-closed release procesu.

## Pouzdano povezivanje

Veza nije „uspješna” kada je samo pokrenut curl/OpenSSH proces. `remote.Manager.Connect` mora dovršiti autentikaciju i početni remote `List` probe. Tek tada vraća `Connected=true`.

### OpenSSH BatchMode regresija

ByFTP ne koristi `sftp -b`. Aktualni OpenSSH pri `-b` postavlja batch način rada i dodaje `BatchMode=yes`, što može onemogućiti password/passphrase AskPass čak i ako je ranije naveden `BatchMode=no`.

Sigurnosna invarijanta zato zahtijeva:

- `-oBatchMode=no`
- zabranu `-b` u SFTP command args
- naredbe preko stdin-a
- regresijski test koji pada čim se `-b` ponovno uvede

### Process-level connect smoke

Od 2.16.1 nije dovoljan samo unit test sastavljenih argumenata. Ne-Windows testni sloj stvarno pokreće lokalne child procese preko produkcijskog `exec.CommandContext` puta:

- FTP/FTPS smoke potvrđuje runtime-secret token, curl config preko stdin-a, MLSD odgovor, parser i wipe-on-close
- SFTP smoke prekida test ako vidi `-b`, zahtijeva `ls -la` na stdin-u i vraća listing koji prolazi produkcijski parser
- testni procesi ne kontaktiraju mrežu i ne koriste stvarne vjerodajnice
- Linux i macOS CI izvršavaju iste smoke regresije na vlastitim runnerima prije izrade install paketa

`scripts/audit_security.py` zahtijeva postojanje tih testova i ključnih markera, pa ih refaktor ne može neprimjetno ukloniti.

### Timeout i otkazivanje

`curl`, `sftp`, `ssh-keyscan` i `ssh-keygen` vraćaju stvarni `context.Canceled` ili `context.DeadlineExceeded` kada ih context prekine. UI tako ne mora zaključivati uzrok samo iz teksta vanjskog alata.

### IPv6

Bracketirani IPv6 unos prihvaća se kao korisnički host, ali se `[]` uklanjaju prije OpenSSH `HostName` i `ssh-keyscan` ulaza.

## Vjerodajnice

### Windows

- profilne lozinke/passphrase koriste Windows DPAPI
- aktivni Windows adapteri koriste DPAPI runtime blob
- plaintext se ne stavlja u command line
- AskPass helper provjerava vlastiti executable, jednokratni token i očekivani System32 OpenSSH parent
- AskPass daje password samo `password` promptu, passphrase samo `passphrase` promptu
- MFA, OTP, security-key i nepoznati promptovi ne dobivaju spremljenu tajnu
- UI ne briše upravo unesenu tajnu prije rezultata spajanja; kontrola je zaključana tijekom pokušaja, a sadržaj se briše nakon potvrđenog `Connected` stanja

### Linux/macOS

FTP/FTPS aktivna lozinka ne sprema se u profil ili datoteku. `ProtectRuntimeString` stvara kriptografski nasumični token i čuva vrijednost samo u procesu. Adapter drži token, `run()` dobiva kratkotrajnu kopiju i briše je, a `Close()` uklanja i briše spremljenu procesnu vrijednost. Process-level smoke test dodatno potvrđuje da tajna više nije dostupna nakon `Close()`.

SFTP na Linuxu/macOS-u trenutačno je namjerno ograničen na eksplicitni privatni ključ bez passphrasea. Password/passphrase SFTP odbija se prije mrežnog pokušaja dok Unix AskPass broker nije dovršen. To je sigurnije od privremenog slanja tajne kroz argument ili običan environment.

## Profilni identitet

Windows profilne tajne koriste `internal/profilebinding`:

- endpoint: protokol + normalizirani host + port
- account: endpoint + korisničko ime
- private-key identitet: account + privatni ključ

Lozinka se ne prenosi na drugi account, passphrase na drugi ključ, a SFTP host-key pin na drugi endpoint. Prazna key putanja je autoritativna i ne smije vratiti stari spremljeni ključ.

## SFTP host-key trust

SFTP spajanje prvo skenira host key, računa SHA-256 fingerprint i veže sesiju uz konkretni odabrani algoritam. `known_hosts` i OpenSSH session config kratkotrajni su i privatni.

Spremljeni pin koristi se samo za isti endpoint. Privremena promjena hosta/porta ne može prepisati pin originalnog profila.

## Vanjski mrežni alati

- Windows koristi sistemski System32 curl/OpenSSH
- Linux/macOS koriste nativni `curl`, `sftp`, `ssh-keyscan` i `ssh-keygen`
- curl ne nasljeđuje proxy/TLS override varijable i dobiva izričiti no-proxy
- OpenSSH blokira ProxyCommand, ProxyJump, agent, PKCS#11 provider, KnownHostsCommand, local command i forwarding
- bez eksplicitnog SFTP ključa koristi se `IdentitiesOnly yes` i `IdentityFile none`
- host, korisnik i private-key putanja nisu na OpenSSH command lineu

## Datoteke i putanje

- udaljene i lokalne putanje prolaze traversal/control-character validaciju
- upload ne prati symlink/reparse objekt
- download staging mora biti regularna datoteka i ne smije biti reparse/symlink
- atomska zamjena koristi no-replace i rollback
- `RemoveTreeNoFollow` ne prati symlink/junction/reparse točke i blokira filesystem root
- rekurzivne operacije imaju depth/item limite
- queued transfer ponovno validira lokalni root prije svakog pokušaja

## Session lifecycle

Svaka `Operation` registrira aktivnu referencu. Disconnect prvo blokira nove operacije, zatim otkazuje session context i čeka postojeće reference. Adapter se ne zatvara ispod aktivnog `List`, rename, chmod ili transfer poziva.

Ako caller deadline istekne, cleanup nastavlja odvojeno. Reconnect je blokiran dok prethodna sesija nije stvarno očišćena. Ponovljeni disconnect koristi isti close-state.

## Transfer izolacija

Transfer posao pamti connection generation i opaque connection identity. Retry na drugi server/account nije dopušten. Event API vraća duboke kopije. Kasni cancel nakon uspješnog ili preskočenog transfera ne mijenja autoritativni završni status.

## State/config

State safe-open provjerava regularnost i stabilnost stvarno otvorene datoteke. Nepouzdani current zapis ne mora srušiti startup: koristi se provjerena prethodna generacija ili zadane vrijednosti.

## Release sigurnost

- `VERSION` je jedini kanonski broj
- quality job: docs/security/privacy/release auditi + Python testovi + Go unit/race/vet
- Windows job: x64 i x86 production build
- Linux job: Go test/vet na Linuxu + amd64, arm64 i i386 DEB
- macOS job: Go test/vet na macOS-u + Universal Intel+Apple Silicon PKG
- Windows ZIP se verificira nakon kompresije i ne smije sadržavati interni uninstaller/verification report
- centralni publisher veže tag uz točan commit i uspoređuje postojeći asset po veličini i SHA-256 digestu
- release rerun smije nadopuniti samo nedostajući potvrđeni asset
- custom Source ZIP, standalone Uninstaller i `verification.txt` nisu javni asseti

## Potpisivanje

Windows paketi nisu Authenticode/Verified Publisher dok ne postoji stvarni Brendigo code-signing certifikat. macOS PKG nije Developer ID potpisan/notariziran bez stvarnog Apple certifikata. Certifikati i privatni ključevi ne smiju biti pohranjeni u repozitoriju.

## Automatizirani gate

`scripts/audit_security.py` zaključava SFTP BatchMode, process-level connect smoke, AskPass prompt, context propagation, IPv6 normalizaciju, runtime tajne, profile binding, session lifecycle i filesystem granice. `scripts/audit_release.py` zasebno zaključava platforme i javni asset ugovor.
