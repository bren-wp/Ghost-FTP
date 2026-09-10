# Packaging transition record

This document records the completed engineering transition from the immutable published `ghostftp-v0.0.2` artifact layout to the 0.0.3 source candidate. It is provenance, not permission to rewrite the historical release.

The **historical 0.0.2 tag/release is not rewritten**. Its architecture-specific Windows and generic Linux asset set remains a fact about that published transaction even after latest-only retention eventually removes superseded public identities.

## Transition goals

The post-0.0.2 packaging work had two explicit public-surface goals:

1. make Windows download choice architecture-neutral without introducing a network bootstrap;
2. make distro-specific Linux packages first-class release artifacts rather than supplemental CI evidence.

The implementation was merged only after exact-head PR gates and exact post-merge `main` gates passed.

## Windows result

0.0.3 exposes exactly two public Windows executables:

```text
Ghost-FTP-0.0.3-Setup.exe
Ghost-FTP-0.0.3-Portable.exe
```

The existing production native builder is preserved as internal staging. It produces and verifies native x64/x86 Setup and Portable payloads, including integrated uninstall ownership and optional Authenticode behavior.

The public bootstrap is an x86-compatible PE that embeds the corresponding native x64/x86 payloads. At runtime it:

- derives native architecture through Windows `GetNativeSystemInfo` rather than architecture environment variables;
- selects only the matching embedded payload;
- stages the payload locally;
- verifies staged payload bytes against the embedded source before execution;
- launches the matching native payload;
- performs no runtime download.

Architecture-specific staging executables must not remain in the public artifact directory.

## Linux result

`linux/BUILD-DISTROS.sh` is the canonical 0.0.3 Linux release builder. It compiles one `ghostftp` executable per Go architecture and packages that same executable into matching distro/archive variants.

Canonical families are:

- Debian DEB: `amd64`, `arm64`, `i386`;
- Ubuntu DEB: `amd64`, `arm64`, `i386`;
- Fedora RPM: `x86_64`, `aarch64`, `i686`;
- Portable tar.gz: `amd64`, `arm64`, `i386`.

`.github/workflows/linux-distro-packages.yml` verifies package metadata and byte parity. `.github/workflows/linux-distro-install.yml` separately proves real install/remove/runtime/GUI lifecycle on Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64. The additional architectures remain build/parity verified without an unsupported native-install claim.

## Release assembly

```text
WINDOWS_PUBLIC_EXECUTABLES=2
WINDOWS_SETUP=UNIVERSAL_X86_X64
WINDOWS_PORTABLE=UNIVERSAL_X86_X64
WINDOWS_NATIVE_PAYLOADS=x64,x86
LINUX_DEBIAN_DEB=amd64,arm64,i386
LINUX_UBUNTU_DEB=amd64,arm64,i386
LINUX_FEDORA_RPM=x86_64,aarch64,i686
LINUX_PORTABLE=amd64,arm64,i386
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
```

The 17-file set consists of 14 platform artifacts plus `BUILD-METADATA.txt`, `RELEASE-NOTES.txt` and `SHA256.txt`.

## Release-lifecycle boundary

The packaging transition itself does not publish a release. The 0.0.3 source candidate must pass its own exact-head PR gates, exact post-merge `main` gates and canonical release-branch identity checks before publication can be authorized. The existing `ghostftp-v0.0.2` identity must never be clobbered to retrofit the new layout.
