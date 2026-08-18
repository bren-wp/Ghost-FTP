# ByFTP — plan razvoja

## Završeno do 2.16 linije

- izvorni Win32 desktop bez browser/localhost sloja
- zajednički tipizirani Engine za Windows, Linux i macOS frontend
- FTP, eksplicitni/implicitni FTPS i SFTP
- Windows DPAPI profili i endpoint/account/private-key binding
- SFTP host-key pinning, session-scoped trust i konkretnim algoritmom vezani `known_hosts`
- ispravljen OpenSSH `sftp -b`/`BatchMode=yes` konflikt; password/passphrase AskPass ostaje funkcionalan na Windowsu
- fail-closed AskPass koji ne šalje tajnu MFA/OTP/nepoznatom promptu
- Linux/macOS terminalni app nad istim remote/transfer coreom
- Linux/macOS procesno runtime spremište FTP/FTPS tajne bez disk persistencije
- Windows x64 + x86 produkcijski build
- Linux DEB amd64 + arm64 + i386 build
- macOS Universal Intel + Apple Silicon PKG build
- transfer queue, transient retry, skip-existing i cijela stabla mapa
- connection-generation i cross-server retry izolacija
- symlink/junction/reparse zaštite, transakcijski staging i rollback
- rekurzivni upload root revalidacija i filesystem-root delete blokada
- remote session active-operation lifecycle s bounded disconnectom
- state/config safe-open i provjerene prethodne generacije
- reproducibilni brand resursi
- hrvatski docs sustav i docs/security/privacy/release auditi
- fail-closed idempotentni GitHub Release publisher
- provjera konačnog Windows ZIP-a i rekurzivnog bundle SHA-256 manifesta
- javni release ugovor bez custom Source ZIP-a, standalone uninstallera i internog verification reporta

## Sljedeći prioriteti

1. **Unix SFTP AskPass broker** — siguran password/passphrase tok na Linuxu/macOS-u bez plaintext argumenta, datoteke ili običnog environmenta
2. **runtime interoperability smoke matriks** — automatizirani FTP/FTPS/SFTP test serveri s jednokratnim testnim credentialima
3. **macOS Developer ID/notarizacija** kada je stvarni Brendigo Apple signing identitet dostupan
4. **Windows Authenticode** kada je stvarni Brendigo code-signing certifikat dostupan
5. **Linux paketni kanali** — eventualni RPM/AppImage tek nakon sigurnosnog i update-model pregleda
6. **pristupačnost** — dodatna Win32 keyboard/focus poboljšanja i bolji terminal help/completion
7. **transfer observability bez telemetrije** — stvarna lokalna brzina/ETA uz bounded event churn i bez vanjskog slanja
8. **memory skaliranje** — dodatna optimizacija vrlo velikih rekurzivnih planova i queue eventa
9. **controlled reconnect** — recovery od mrežnog prekida bez slabljenja endpoint/account identity granice
10. **provenance/attestation** — build attestations bez runtime telemetrije ili tajni u repozitoriju

Svaka nova funkcija mora očuvati privatnost, hrvatski korisnički sadržaj, tipizirani in-process API, platform-specific auth granice, existing filesystem/session zaštite i fail-closed release ugovor.
