# Packaging transition record

The packaging transition prepared after the immutable `ghostftp-v0.0.2` release is now adopted by the **0.0.3 source candidate**. The historical 0.0.2 tag/release is not rewritten.

## Windows result

0.0.3 exposes exactly two public Windows executables:

```text
Ghost-FTP-0.0.3-Setup.exe
Ghost-FTP-0.0.3-Portable.exe
```

Verified native x64/x86 payloads remain internal. The x86-compatible bootstrap selects the native payload from Windows system architecture information, performs no runtime download and verifies staged bytes before execution.

## Linux result

`linux/BUILD-DISTROS.sh` is the canonical 0.0.3 Linux release builder. It publishes Debian/Ubuntu DEBs for `amd64`, `arm64`, `i386`; Fedora RPMs for `x86_64`, `aarch64`, `i686`; and Portable tarballs for `amd64`, `arm64`, `i386`.

## Release assembly

```text
WINDOWS_PUBLIC_EXECUTABLES=2
LINUX_DEBIAN_DEB=amd64,arm64,i386
LINUX_UBUNTU_DEB=amd64,arm64,i386
LINUX_FEDORA_RPM=x86_64,aarch64,i686
LINUX_PORTABLE=amd64,arm64,i386
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
```

The 17-file set consists of 14 platform artifacts plus `BUILD-METADATA.txt`, `RELEASE-NOTES.txt` and `SHA256.txt`. This document is retained as engineering provenance for the 0.0.2 → 0.0.3 packaging transition.
