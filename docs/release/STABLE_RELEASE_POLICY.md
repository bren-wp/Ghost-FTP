# Ghost FTP — Stable distribution policy

Owner: BRENDIGO — https://brendigo.com

From v0.99.90 onward, all successfully verified new public releases use the Stable channel and GitHub prerelease=false. This owner-approved distribution policy supersedes the earlier requirement to publish unsigned releases only as Preview. Historical tags and published binary bytes remain unchanged.

Stable here describes the verified distribution channel, not an assertion of complete device coverage or cryptographic publisher identity. All existing exact-SHA CI, package-format, checksum, release lineage, protocol, native build, credential and recovery gates remain mandatory. Never promote a failing build. Explicitly document untested scenarios, unsigned OS packages and limitations; do not claim independent signing or native acceptance unless proved.

Windows Authenticode and Linux package signing remain desirable and are reported separately. Unsigned manual downloads remain permitted. Never install unsigned automatic updates. Keep the embedded updater verification key, signed-version and downgrade checks; incomplete signature sets fail closed. No signing bypass is permitted.

The update web root contains actual release-specific Windows EXE/MSI and Linux DEB/RPM installers, excludes portable packages, and serves local URLs only. Verify SHA-256 and exact-source provenance before publication.
