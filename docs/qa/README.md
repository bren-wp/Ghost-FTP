# Ghost FTP — Windows/Linux quality assurance

**Supported product:** Ghost FTP desktop for Windows and Linux. Native screenshots must always be genuine captures, not mockups presented as running applications.

The previous mandatory pixel-comparison checklist has been retired. Current UI quality is verified through functional interaction, accessible keyboard and screen-reader behavior, correct text/layout scaling, responsive windows and regression testing. There is **no required exact screenshot-match threshold or fixed screen count** for release qualification.

## Required validation

- [Protocol E2E](PROTOCOL_E2E.md): authorized FTP, explicit FTPS and SFTP roundtrips, host identity, failure/retry/cancellation and file integrity.
- [Transfer reliability](TRANSFERS.md): queue, recovery, incomplete downloads and overwrite safeguards.
- [Interactive actions](CLICK.md): actual file operations, destructive confirmation, error handling and focus.
- [Responsive layouts](RESPONSIVE.md): multi-size and display scaling without clipped controls.
- [Window and keyboard behavior](TITLEBAR.md) and [installed package acceptance](INSTALLER.md).
- [Desktop production readiness](../status/PRODUCTION_READINESS.md) and [stable release policy](../release/STABLE_RELEASE_POLICY.md).

Retain historical visual records only as provenance; they are **not** required release gates or proof of stable readiness. Actual OS installation, upgrades, keyring behavior and network workflows require independent verification.
