# Ghost FTP Desktop — instalacija na Mac bez Apple Developer potpisa

## Hrvatski

Podržani su macOS 13+ i zasebni paketi za Apple Silicon (arm64) i Intel (x86_64). Nije potreban Apple Developer račun, certifikat ni notarizacija. DMG je **nepotpisan i nenotariziran**, a aplikacija u njemu ima samo besplatni **ad-hoc potpis integriteta**, bez identificiranog razvojnog programera. Paket ostaje razvojni Preview.

1. Preuzmite `GhostFTP-macOS-Shared-arm64-vX.Y.Z-Installable-Preview.dmg` za Apple Silicon ili `GhostFTP-macOS-Shared-x86_64-vX.Y.Z-Installable-Preview.dmg` za Intel na [službenoj stranici GitHub Releases](https://github.com/bren-wp/Ghost-FTP/releases).
2. U Finderu otvorite DMG i **povucite Ghost FTP Desktop.app u Applications**. Aplikaciju pokrenite **iz Applications**, a ne izravno iz DMG-a.
3. Ako macOS odbije otvaranje, pokušajte otvoriti aplikaciju jednom, zatim otvorite **System Settings → Privacy & Security → Open Anyway** (Postavke sustava → Privatnost i sigurnost → Svejedno otvori) i potvrdite **Open**, ali samo ako vjerujete izvoru.
4. Provjerite preuzeti DMG pomoću datoteke `GhostFTP-vX.Y.Z-SHA256SUMS.txt` priložene istom GitHub izdanju.

**Ne isključujte Gatekeeper** na cijelom Macu (npr. `sudo spctl --master-disable`), ne uklanjajte `com.apple.quarantine` koristeći `xattr` i ne zaobilazite upozorenje da je aplikacija oštećena ili je potpis neispravan. U tom slučaju ponovno preuzmite službeni paket i prijavite pogrešku. Poslovni Macovi s MDM pravilima mogu zahtijevati administratora.

Stara SwiftUI `Ghost FTP.app` i nova `Ghost FTP Desktop.app` odvojeni su programi. Instalacija novoga ne smije brisati postojeće profile, Keychain lozinke ili korisničke podatke.

## English

Ghost FTP Desktop Preview runs on **macOS 13+** with separate Intel x86_64 and Apple Silicon arm64 disk images. The DMG itself is **unsigned and not notarized**. The enclosed application only has a free ad-hoc integrity signature; it does **not** use Apple Developer ID.

1. Download the matching `GhostFTP-macOS-Shared-ARCH-vX.Y.Z-Installable-Preview.dmg` from the official [GitHub Releases](https://github.com/bren-wp/Ghost-FTP/releases).
2. Open the DMG in Finder; **drag Ghost FTP Desktop.app into Applications** and launch that installed copy.
3. After a blocked first attempt, choose **System Settings → Privacy & Security → Open Anyway**, and confirm **Open** if you trust the app. This is a per-app exception only.
4. Check the versioned SHA-256 release manifest for the downloaded disk image.

Never disable Gatekeeper globally or strip quarantine flags. Do not bypass damaged-app or invalid-signature errors. The DMG is an unnotarized development **Preview**, not Apple-certified software; real Mac acceptance remains to be performed.

Official information: https://support.apple.com/102445 and https://developer.apple.com/documentation/xcode/packaging-mac-software-for-distribution.
