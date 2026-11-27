# Ghost FTP — Windows/Linux development roadmap

**Development candidate:** v0.99.92 Stable; previously published v0.99.91. **Stable channel:** v0.99.90 onward after successful verification; signing and installed-device evidence are tracked separately.

## Development priorities

See the [capability and acceptance roadmap](product/CAPABILITY_ROADMAP.md) for implementation order, product gaps and verification requirements.

1. File transfer integrity: reject unsolicited partial HTTP/cloud GET responses, validate stream length before writes, sync staging, then advance remote hash verification where available, transfer resume/restart, orphan cleanup and stress tests.
2. Security and dead-code quality: strict SSH/FTPS identity, protected credentials, safe file operations, dependency analysis and independent audits.
3. Windows/Linux quality: working UI controls, keyboard and accessibility support, multi-size layouts, genuine installed native-app validation and real protocol tests.
4. Stable distribution: signed artifacts, verified updater, target-OS installation and upgrade/rollback, live provider acceptance, exact-source provenance.

The former mandatory design-reference pixel comparison is retired. Real screenshot provenance, responsive behavior, file-safety protections and functional validation remain mandatory.

See [Stable release policy](release/STABLE_RELEASE_POLICY.md) and [production readiness](status/PRODUCTION_READINESS.md).
