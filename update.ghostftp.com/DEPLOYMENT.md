# Self-contained update web root

The root archive contains only public PHP/CSS/icon files, a local catalogue, SHA-256 checksums and the actual Windows EXE/MSI and Linux DEB/RPM installers. No Portable or AppImage downloads and no external download links. Never publish an archive without all four validated installers.

The release pipeline generates a new root archive from the same exact-source installers before publication. The PR quality job also creates a tested archive from the most recent published predecessor. Deploy the ZIP into the document root over HTTPS, using a fresh staging directory and an atomic host-level directory switch. Do not mix versions. PHP 8, Apache 2.4, rewrite and headers modules are required.

The existing signed updater endpoint is not changed. The two web aliases deliberately return 204 until a signed, compatible local updater implementation is accepted. Manual packages do not enable unattended installation. Keep old client compatibility until native acceptance is verified.

The read-only `/manual-version.json` endpoint returns the version of a complete local installer set or HTTP 204. This is informational only; the desktop must never execute packages based on this endpoint. The signed updater service remains separate.
