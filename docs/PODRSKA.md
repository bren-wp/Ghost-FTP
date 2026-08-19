# Podrška i rješavanje problema

**Kada se ByFTP ne spoji, cilj nije nagađati — cilj je brzo utvrditi je li problem u hostu, portu, vjerodajnicama, TLS-u ili pravilima hostinga.**

## Najbrža provjera za shared hosting

Prije svega usporedite podatke iz ByFTP-a s hosting panelom:

- **Host** mora biti FTP/FTPS hostname koji je dao provider;
- **Korisničko ime** mora biti potpuno, uključujući `@domena` ako ga hosting koristi;
- **Lozinka** mora pripadati tom FTP računu, ne nužno glavnom hosting korisniku;
- **Port** mora odgovarati protokolu;
- ako provider traži TLS, koristite FTPS umjesto plain FTP-a.

Najčešće kombinacije:

- FTP — port 21;
- FTPS (eksplicitni) — port 21;
- FTPS (implicitni) — port 990;
- SFTP — port 22.

Ako provider navodi drugačiji port, koristite njegovu vrijednost.

## “Povezivanje nije uspjelo”

ByFTP stanje **POVEZANO** postavlja tek nakon autentikacije i početnog udaljenog listinga. Zbog toga poruka o grešci može značiti jednu od nekoliko stvari.

### 1. Host se ne može pronaći

Provjerite:

- jeste li slučajno upisali `https://` ili `ftp://` umjesto samog hosta;
- postoji li tipfeler u nazivu servera;
- radi li DNS i internet veza;
- koristi li hosting poseban FTP hostname koji se razlikuje od domene web stranice.

### 2. Connection refused / timeout

Najčešći uzroci:

- pogrešan port;
- pogrešan protokol;
- firewall ili antivirus blokira izlaznu vezu;
- hosting privremeno blokira IP;
- FTP servis nije aktivan na tom hostu.

### 3. Login denied / authentication failed

Provjerite:

- korisničko ime u cijelosti;
- je li FTP lozinka promijenjena u hostingu;
- postoji li račun i ima li pristup traženom direktoriju;
- koristi li provider zasebnu FTP lozinku.

## Spoji se, ali ne vidim `public_html`

To ne mora biti greška.

Shared hosting može koristiti:

- `public_html`;
- `www`;
- `httpdocs`;
- `htdocs`;
- direktorij domene;
- home direktorij FTP subračuna ograničen samo na jednu mapu.

ByFTP poštuje direktorij u koji vas server smjesti nakon prijave. Ako FTP subaccount ima root postavljen izravno na web mapu, možda uopće nećete vidjeti `public_html` — već ćete odmah vidjeti sadržaj web stranice.

## Listing ne radi ili server je stariji

ByFTP preferira MLSD kada ga server pravilno podržava jer daje strukturiraniji listing.

Ako server ne podržava MLSD ili vraća neupotrebljiv odgovor, ByFTP 1.0.5 prelazi na klasični LIST i taj fallback zadržava do kraja sesije. Time se izbjegava ponavljanje iste neuspjele MLSD naredbe pri svakom refreshu.

## Upload počne, ali završna aktivacija ne uspije

ByFTP upload nije samo “pošalji datoteku pod finalnim imenom”. Koristi temp objekt, provjeru izvora i završni remote commit.

Završna faza može pasti ako:

- hosting ne dopušta rename u toj mapi;
- drugi klijent je u međuvremenu stvorio datoteku ili direktorij istog imena;
- račun nema write prava;
- server prekine vezu;
- shared hosting ograničava pojedine FTP naredbe.

U 1.0.5 `RNFR/RNTO`, `DELE`, `MKD`, `RMD` i `SITE CHMOD` koriste isti login-home namespace kao listing i upload/download, što uklanja klasu pogrešaka na non-chrooted shared-hosting računima.

## CHMOD ne radi

Neki hosting provideri:

- ne podržavaju `SITE CHMOD`;
- dopuštaju ga samo na određenim računima;
- ignoriraju ga jer permission model nije klasični Unix.

Ako upload/download rade, a samo CHMOD ne radi, problem je vrlo često ograničenje servera, ne problem veze.

## FTPS certifikat ili TLS greška

Provjerite da:

- koristite hostname za koji je certifikat izdan;
- niste zamijenili explicit i implicit FTPS;
- port odgovara načinu rada;
- vrijeme i certifikacijska infrastruktura operativnog sustava rade ispravno.

ByFTP ne koristi globalno `ssl-no-revoke` za “popravljanje” certifikata. Sigurnosna provjera se ne isključuje samo da bi veza izgledala uspješna.

## SFTP fingerprint se promijenio

Ako spremljeni fingerprint više ne odgovara serveru, ByFTP blokira vezu.

To može biti legitimna promjena host ključa nakon migracije servera, ali može značiti i da endpoint nije onaj koji očekujete. Fingerprint provjerite s hosting providerom prije prihvaćanja nove vrijednosti.

## Datoteka se ne može preuzeti ili prepisati lokalno

ByFTP blokira neke lokalne putanje kada otkrije:

- symlink;
- junction;
- reparse-point preusmjeravanje;
- cilj koji je direktorij umjesto obične datoteke;
- konkurentno zauzeto no-replace odredište.

Ovo je namjerno ponašanje radi zaštite od neželjenog prepisivanja izvan odabrane mape.

## Što pripremiti za kvalitetan bug report

Ne šaljite lozinku, private key ni druge tajne.

Pošaljite:

- verziju ByFTP-a;
- operativni sustav i arhitekturu;
- protokol i port;
- opći tip hostinga, primjerice shared hosting/VPS;
- točnu poruku greške bez vjerodajnica;
- radnju koja je prethodila grešci;
- radi li isti FTP račun u drugom klijentu;
- događa li se problem samo na uploadu, listingu, renameu, downloadu ili svemu.

## Što ByFTP namjerno ne radi

ByFTP ne šalje runtime activity log na Brendigo server, nema aplikacijsku telemetriju i nema vanjski crash-reporting servis iz kojeg bi podrška mogla “sama dohvatiti” vašu sesiju. To štiti privatnost, ali znači da kvalitetan opis problema ostaje važan.

---

**Za većinu shared-hosting problema prvo provjerite host + puni username + protokol + port. Ako login prođe, ByFTP 1.0.5 je posebno pooštren za home-relative FTP rad nakon prijave.**
