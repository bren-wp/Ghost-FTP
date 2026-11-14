# Ghost FTP — Windows/Linux development roadmap

**Development candidate:** v0.99.65 Preview; previous published v0.99.64. **Future Stable target:** v1.0.0, after acceptance.

## Development priorities

1. File transfer integrity: reject unsolicited partial HTTP/cloud GET responses, validate stream length before writes, sync staging, then advance remote hash verification where available, transfer resume/restart, orphan cleanup and stress tests.
2. Security and dead-code quality: strict SSH/FTPS identity, protected credentials, safe file operations, dependency analysis and independent audits.
3. Windows/Linux quality: working UI controls, keyboard and accessibility support, multi-size layouts, genuine installed native-app validation and real protocol tests.
4. Stable distribution: signed artifacts, verified updater, target-OS installation and upgrade/rollback, live provider acceptance, exact-source provenance.

The former mandatory design-reference pixel comparison is retired. Real screenshot provenance, responsive behavior, file-safety protections and functional validation remain mandatory.

See [Stable release policy](release/STABLE_RELEASE_POLICY.md) and [production readiness](status/PRODUCTION_READINESS.md).
