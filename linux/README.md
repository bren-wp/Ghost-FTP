# Ghost FTP for Linux

Ghost FTP Linux packages are built by `linux/BUILD.sh` for `amd64`, `arm64` and `i386`.

## Build

```bash
go telemetry off
bash linux/BUILD.sh
```

The script creates:

```text
dist/Ghost-FTP-X.Y.Z-Linux-amd64.deb
dist/Ghost-FTP-X.Y.Z-Linux-arm64.deb
dist/Ghost-FTP-X.Y.Z-Linux-i386.deb
```

The GitHub Release combines these verified packages into one public download:

```text
Ghost-FTP-X.Y.Z-Linux-multiarch.zip
```

## Installed identity

- Debian package: `ghost-ftp`
- executable: `/usr/bin/ghostftp`
- desktop name: **Ghost FTP**
- desktop entry: `ghost-ftp.desktop`

Runtime dependencies declared by the package are `ca-certificates`, `curl` and `openssh-client`.

Production build scripts require Go telemetry to be disabled and use controlled Go dependency settings from CI.
