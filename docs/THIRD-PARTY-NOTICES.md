# Third-party notices

Ghost FTP **0.0.6** keeps the maintained desktop/core Go module intentionally free of external Go module requirements. The shared desktop engine uses operating-system networking tools for protocol execution rather than bundling an untracked third-party FTP/SSH stack.

The current public release applications are **Windows, Linux and Android**. Chrome, Edge and Firefox helper ZIPs are public companion packages. macOS is an active native development/source frontend and remains outside the public 21-file release until real Developer ID signing and Apple notarization succeed.

## Desktop runtime transport tools

### FTP / FTPS

The shared desktop transport uses an operating-system `curl` executable for FTP and explicit FTPS. Curl is distributed/licensed by the relevant operating-system/vendor package source; Ghost FTP does not silently download a private bundled copy at runtime.

Linux applies the documented trusted root-controlled transport-tool boundary before credential-bearing automation is allowed. Other desktop platforms retain their documented system-tool and transport validation contracts.

### SFTP

The shared desktop SFTP transport uses operating-system OpenSSH `ssh` / `sftp` tooling. Windows normally uses the Windows OpenSSH Client capability and Linux the distribution OpenSSH client package. macOS development consumes the same shared engine rather than adding a second desktop protocol stack.

OpenSSH is distributed/licensed by its operating-system/vendor source. Ghost FTP constrains configuration, trust and credential handoff but does not relicense or impersonate OpenSSH.

## Linux package prerequisites

Debian/Ubuntu packages declare `ca-certificates`, `curl` and `openssh-client`; Fedora declares `ca-certificates`, `curl` and `openssh-clients`. Portable archives do not vendor private copies of those tools or CA stores.

## macOS development dependency boundary

The AppKit development frontend is built from repository source and uses the shared Go engine. AppKit, Security.framework/Keychain and Apple platform tooling are operating-system/developer-platform facilities rather than vendored Go modules.

The development artifact may be ad-hoc signed for validation. Public macOS distribution additionally requires the real Developer ID + notarization path; source/development build success is not production-signing evidence.

## Android public dependency boundary

Android 0.0.6 is a public native application through the protected production-signing path. Its Java/Android SDK and Gradle ecosystem are build/platform dependencies and do not alter the root Go module's zero-external-module contract.

Android uses platform networking and Storage Access Framework capabilities. It exposes FTP and strict explicit FTPS; **SFTP remains hidden** until strict maintained host-key identity support exists. JUnit and CI-only signing identities remain test/build dependencies rather than production telemetry/network services.

## Browser connection helper

`ekstenzije/` contains the official Manifest V3 Chrome, Edge and Firefox companion packages. They use browser-provided extension APIs and one local shared JavaScript runtime; they do not bundle an FTP/SFTP networking stack, connect to transfer servers, or provide a supported browser-to-desktop URI/native-messaging handoff today.

The helper requests no broad host, tab/history, storage, scripting or network permissions. The three deterministic ZIPs are public 0.0.6 companion artifacts while remaining outside the native application protocol runtime.

## GitHub Actions

Build workflows use pinned GitHub Actions revisions for checkout, language/toolchain setup and artifact upload/download. Runner packaging/signing tools are CI dependencies, not Ghost FTP runtime components.

## Signing material

Code-signing certificates, Android publisher keystores, Apple Developer credentials and private keys are not runtime libraries and must never be committed to this repository.

Official public Windows publication requires the protected trusted production Authenticode identity and valid signatures on both public executables:

```text
WINDOWS_AUTHENTICODE=signed
```

There is no supported unsigned continuation. Local/development or ordinary CI Windows artifacts may be unsigned only because they are not official publication artifacts.

Android public publication independently requires the protected production keystore/alias/passwords and exact configured signer-certificate SHA-256 match. Ephemeral CI signing is pipeline smoke evidence only.

macOS production distribution has a separate fail-closed Developer ID + notarization boundary. Ghost FTP does not substitute generated/self-signed production identities or claim notarization from an ad-hoc development build.

## Identity and historical provenance

Historical Git content may describe older release surfaces. Historical provenance is not the active 0.0.6 dependency/release contract.

Current public product identity is **Ghost FTP**. Internal compatibility identifiers containing `GhostFTP` remain only where changing them could break installed identity or upgrade behavior.

See [Dependencies](DEPENDENCIES.md), [Security](SECURITY.md), [Privacy](PRIVACY.md), [Signing](SIGNING.md) and [Platform parity](PLATFORM-PARITY.md).
