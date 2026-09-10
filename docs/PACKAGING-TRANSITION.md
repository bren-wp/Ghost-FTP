# Packaging transition note

The current published Ghost FTP release remains **0.0.2** and keeps its already-published immutable asset set. This source change prepares the packaging contract for the next release; it does not redefine or rewrite `ghostftp-v0.0.2`.

## Next Windows release contract

The next release exposes exactly two public Windows executables:

```text
Ghost-FTP-X.Y.Z-Setup.exe
Ghost-FTP-X.Y.Z-Portable.exe
```

The build still produces verified native x64 and x86 application payloads internally. The public bootstrap selects the native payload from Windows system architecture information, performs no runtime download, verifies the selected embedded payload before execution, and retains the existing integrated-uninstall and Authenticode policies.

## Next Linux release contract

Canonical Linux publication is prepared from `linux/BUILD-DISTROS.sh` as twelve distro/portable artifacts:

```text
Ghost-FTP-X.Y.Z-Linux-Debian-amd64.deb
Ghost-FTP-X.Y.Z-Linux-Debian-arm64.deb
Ghost-FTP-X.Y.Z-Linux-Debian-i386.deb
Ghost-FTP-X.Y.Z-Linux-Ubuntu-amd64.deb
Ghost-FTP-X.Y.Z-Linux-Ubuntu-arm64.deb
Ghost-FTP-X.Y.Z-Linux-Ubuntu-i386.deb
Ghost-FTP-X.Y.Z-Linux-Fedora-x86_64.rpm
Ghost-FTP-X.Y.Z-Linux-Fedora-aarch64.rpm
Ghost-FTP-X.Y.Z-Linux-Fedora-i686.rpm
Ghost-FTP-X.Y.Z-Linux-Portable-amd64.tar.gz
Ghost-FTP-X.Y.Z-Linux-Portable-arm64.tar.gz
Ghost-FTP-X.Y.Z-Linux-Portable-i386.tar.gz
```

Each architecture reuses one production executable across its matching Debian, Ubuntu, Fedora and Portable package variants. Native install/remove/GUI lifecycle verification remains Debian 13 amd64, Ubuntu 26.04 LTS amd64 and Fedora 44 x86_64; the other architectures remain protected by build, metadata, extraction and byte-parity checks.

## Next release assembly

```text
WINDOWS_PUBLIC_EXECUTABLES=2
LINUX_DEBIAN_DEB=amd64,arm64,i386
LINUX_UBUNTU_DEB=amd64,arm64,i386
LINUX_FEDORA_RPM=x86_64,aarch64,i686
LINUX_PORTABLE=amd64,arm64,i386
PUBLIC_PLATFORM_ARTIFACTS=14
PUBLIC_RELEASE_FILES=17
```

The 17-file set consists of 14 platform artifacts plus `BUILD-METADATA.txt`, `RELEASE-NOTES.txt` and `SHA256.txt`.

## Version transition

Root `VERSION` intentionally remains **0.0.2** while this packaging implementation is reviewed. After this change is merged and exact post-merge `main` workflows are green, a separate reviewed release-prep change can advance the source candidate to **0.0.3**, update current-release documentation and only then create the corresponding release branch. No existing release or tag is rewritten as part of this packaging change.
