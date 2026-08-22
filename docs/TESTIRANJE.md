# Testing and quality gates

A release is not considered production-ready because it builds once. ByFTP uses layered automated gates for functional behavior, races, security invariants, privacy policy, documentation and packaging.

## Go checks

The standard gate runs:

```text
go test ./...
go test -race ./...
go vet ./...
```

Platform jobs additionally compile/build Windows x64 and x86, Linux amd64/arm64/i386 packages and the macOS Universal package.

## Real FTP integration

The remote test suite includes an authenticated loopback FTP server and drives the production curl-backed client through real TCP control and passive data channels. It verifies login, wrong-password rejection, listing, upload, rename, byte-identical download, delete and cleanup.

This is a real protocol integration test, not a fake curl stdout fixture. It remains local so CI does not require public FTP credentials.

## Source-level security regressions

Python audits enforce invariants that are easy to accidentally bypass during refactoring: process lifecycle configuration, release immutability, documentation links, privacy/network policy, localization completeness, installer transaction rules and critical remote-transfer protections.

## Localization

English is canonical. Every supported locale must contain the same keys and compatible formatting placeholders. Secondary locales are tested against excessive English fallback. Windows action text is split into structured locale modules to make missing workflow translations fail visibly during tests.

## Release rule

Never merge because most jobs are green. The exact PR head intended for merge must pass the complete required gate.