# ByFTP documentation

English is the canonical language for ByFTP source documentation and repository maintenance. Runtime user-facing text is localized independently and currently supports English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Simplified Chinese, Russian, Hindi and Japanese.

The historical Croatian filenames in this directory are intentionally kept stable for 1.0.12 so existing links and packaged documentation do not break. Their contents are English-first.

## Start here

- [Installation and first connection](INSTALACIJA.md)
- [Shared hosting](SHARED-HOSTING.md)
- [Support and troubleshooting](PODRSKA.md)

## Security and privacy

- [Security model](SIGURNOST.md)
- [Privacy model](PRIVATNOST.md)
- [Testing and quality gates](TESTIRANJE.md)
- [Release verification](PROVJERA-IZDANJA.md)
- [Distribution signing](POTPISIVANJE.md)

## Development and maintenance

- [Architecture](ARHITEKTURA.md)
- [Contribution policy](DOPRINOS.md)
- [GitHub release process](IZDAVANJE-NA-GITHUBU.md)
- [Development roadmap](PLAN-RAZVOJA.md)
- [Third-party notices](OBAVIJESTI-TRECIH-STRANA.md)

## Product scope

ByFTP is a telemetry-free FTP, FTPS and SFTP client. Windows provides the native dual-pane desktop interface. Linux and macOS provide a terminal interface backed by the same engine. The codebase intentionally keeps protocol execution, transfer management, security validation, profile storage and presentation separated so UI changes cannot silently weaken transfer or credential invariants.

For release-specific changes, see the root [CHANGELOG](../CHANGELOG.md).