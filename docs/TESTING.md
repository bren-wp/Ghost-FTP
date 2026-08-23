# Testing and quality gates

The repository treats tests and audits as release requirements.

Core checks include `go test ./...`, the Go race detector, `go vet`, Windows x64/x86 builds, Linux package builds, the macOS Universal package, deterministic brand-asset verification, localization catalog checks, version consistency, documentation links, security/privacy audits and release-pipeline validation.

Installer changes require regression coverage for payload validation, transaction/rollback behavior and settings persistence. Localization changes must preserve English fallback and compatible formatting placeholders.
