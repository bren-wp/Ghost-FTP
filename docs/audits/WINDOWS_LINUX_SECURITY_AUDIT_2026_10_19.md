# Ghost FTP — Windows/Linux sigurnosni i stabilnosni audit (19. 10. 2026.)

**Opseg:** proizvodni desktop `ghostftp-desktop/` — Rust, Tauri, React/TypeScript, Windows i Linux. Audit je započeo na `main` SHA `232931cbc887f0d3a13ac049b91b3f6438191d66`. Sve predložene promjene nalaze se **isključivo u [PR #211](https://github.com/bren-wp/Ghost-FTP/pull/211)**. Ovaj dokument nije certifikat da cijeli proizvod nema grešaka.

## Potvrđeni nalazi / ispravci na razvojnoj grani

| Rizik | Izvorni problem | Uska mjera i negativni test |
| --- | --- | --- |
| Kritično — čuvanje postojećeg backupa | `backup::write_private_file` skraćivao je ciljni fajl prije završetka novog zapisa i na Linuxu mogao slijediti simboličku poveznicu | Nasumični privatni sibling temp, `create_new`, `sync_all`, `rename`, uklanjanje napuštenog tempa; testovi očuvanja link-targeta, nepravilnog cilja i UNIX 0600 moda |
| Visoko — nepotpun backup | Čitanje opcionalnih `profiles.json`, `bridge.json` i `foldersync.json` tiho je ignoriralo I/O greške putem `.ok()` | Fail-closed za nedostupne, direktorijske i simbolički preusmjerene izvore; samo stvarni `NotFound` znači da konfiguracija nije prisutna |
| Visoko — sudar temp datoteke | SQLite export koristio je predvidljiv `ghostftp.db.backup.tmp`, koji je naknadni `Drop` mogao obrisati čak i ako je već postojao | UUID snapshot temp; test očuvanja starog imena i uklanjanja novog tempa |
| Kritično — SQLite restore | Startup je brisao postojeći `ghostftp.db` i WAL/SHM prije provjere `rename` te ignorirao pogreške | `PRAGMA quick_check`, odbijanje staged WAL/symlinka, čuvanje stare baze i pripadnih WAL/SHM u jedinstvenim oporavnim datotekama, povrat pri pogrešci; startup propagira grešku |
| Visoko — downgrade sheme | `migrate` nije odbijao negativan ili noviji SQLite `user_version` | Provjera raspona prije DDL-a i regresija koja čuva prethodnu tablicu i user_version |
| Visoko — keyring brisanje | `set_secret("...")` pri praznoj vrijednosti ignorirao je grešku brisanja i mogao netočno ukloniti SQLite manifest | `clear_secret_checked` prihvaća samo uspjeh ili `NoEntry`; zaključani keyring se tretira kao neuspjeh; bez izlaganja detalja tajne |
| Visoko — dijagnostika | Izvedeni Rust `Debug` za `AuthMethod` prikazivao je in-memory password, passphrase, putanju SSH privatnog ključa i keyref | Ručni redaktirani `Debug`; serde/legacy migration nepromijenjeni; negativni testovi s jedinstvenim sentinelima |
| Srednje — editor race | Dva brza poziva `startBuiltinEditing` mogla su proći `builtinEditId` provjeru prije prvog `await` | Sinkroni single-flight gate uz `finally`; TypeScript gate test i UI source ugovor |

## Credential Manager, Secret Service i Argon2id

- Windows: `keyring` 3 s `windows-native` značajkom pristupa Windows Credential Manageru. Nema namjerne promjene identiteta, naziva entryja ni API/IPC ugovora.
- Linux: postojeća Cargo konfiguracija `linux-native` **znači kernel keyutils**, ne desktop Secret Service. Zamjena flagova bez migracije mogla bi učiniti stare spremljene ključeve nedostupnima. Za transakcijski, idempotentan, postupni prijelaz ostaje [issue #207](https://github.com/bren-wp/Ghost-FTP/issues/207). Obvezni ručni testovi: GNOME Keyring, KWallet, locked/unavailable DBus, logout/reboot, headless i oporavak nakon djelomične migracije.
- `backup.rs`: postojeći šifrirani format AES-256-GCM s Argon2id `m=64 MiB, t=3, p=1` i granicama veličine/lozinke nije promijenjen. Provjera stvarnog korisničkog Windows/Linux keyringa i povrata svakog providera još nije potvrđena.
- `profiles/mod.rs`: čuvanje profila i migracije stare JSON autentikacije neizmijenjeni su; `ProfileStore::write` odbija pohranu ne-praznih lozinki i privatnih passphraseova. `Debug` više ne smije izložiti in-memory tajnu. SQLite **nije enkripcija**; manifest sadrži reference, ne vrijednosti tajni.

## Full dead-code audit — provjerljive granice

- `npm run check:dead` provjerava dohvatljivost frontend modula i Rust modula iz stvarnih produkcijskih entrypointova. Nezakonito bi bilo smatrati nepotražene dinamičke reference automatski mrtvima.
- `npm run check` pokreće TypeScript `strict`/`noUnusedLocals`; `npm run check:ui` ima zasebne regresijske ugovore, uključujući single-flight testove.
- `cargo fmt --all -- --check`, `cargo test --workspace --locked` i `cargo clippy --workspace --all-targets --locked -- -D warnings` su Rust sigurnosni filteri u CI-ju.
- Nema masovnog uklanjanja public API-ja ili starih migracija: **nijedna datoteka nije uklonjena bez dokaza**. Ovi statički alati nisu zamjena za dinamičku pokrivenost svih grana i OS implementacija.

## Nezatvoreni rizici i obvezni dokazi prije spajanja

1. Oporavak SQLite-a sada čuva izvornu bazu, ali potpuna više-datotečna transakcija (`profiles.json` / `bridge.json` / `foldersync.json` / SQLite / OS keyring) **nije implementirana**. Ukinuti svaku tvrdnju da restore ima garantirani all-or-nothing commit preko tih sustava.
2. Postojeće druge putanje `delete_secret()` još imaju best-effort API i trebaju posebno dokazivanje bez orphaned tajni. Izvoz velikih backupa i realni power-cut/fault-injection scenariji također su otvoreni.
3. Nisu završeni ručni testovi Windows Credential Managera, Linux Secret Servicea, locked-kernel-keyring, NTFS/SMB raceova, distributivnih varijanti i 59 pojedinačnih UX screenshot usporedbi.
4. Na polaznom `main` `version.json` navodi `0.95.44`, ali nepromjenjivi `v0.95.44` tag već pokazuje drugi source SHA. `Ghost FTP quality` stoga završava na immutable-tag preflightu neovisno o kodu. **Ne izmišljati zeleno stanje i ne dirati javni tag.** Sljedeće koordinirano povećanje release traina mora poštivati jedinstveni repo version-policy i zahtjeve izvan Windows/Linux opsega.
5. PR zahtijeva sve točne SHA rezultate šest CI familija (quality, native build, protocol E2E, Android/macOS kompatibilnost, Windows hardening) i nakon spajanja odgovarajuće provjere na `main`. Nikakav automatski merge/release nije dio ovog audita.

**Status ovog dokumenta:** evidencija izvornog nalaza i predloženih izmjena, a ne tvrdnja o konačno uspješnom CI-ju ili funkcionalnoj certifikaciji. Svaki budući rezultat treba vezati uz konkretni SHA.
