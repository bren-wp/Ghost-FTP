# ByFTP build & verification scripts

This directory contains the release tooling used by ByFTP's production pipeline.

The scripts are intentionally part of the repository so a release can be audited from source before publication.

Key checks include:

- privacy/network-policy audit
- installer payload creation and SHA-256 verification
- Windows PE resource/version metadata injection
- release PE mitigation verification
- reproducible release integrity checks

The production pipeline is designed around ByFTP's core constraints: native Windows operation, no telemetry, no external runtime API dependency and no hidden credential upload path.

For the full release sequence use `BUILD-WINDOWS.ps1` from the repository root and review `RELEASE-CHECKLIST.md` before publishing binaries.
