# ByFTP for Linux

`linux/` is the canonical Linux application packaging surface. The runtime itself remains in the shared Go desktop core under `cmd/` and `internal/` so Linux, Windows and macOS do not carry duplicated protocol, transfer or security implementations.

## Contents

- `BUILD.sh` — production Linux build and DEB packaging entry point.
- `byftp.desktop` — desktop-entry metadata installed under `/usr/share/applications`.
- `debian/control.in` — DEB metadata template populated from the root `VERSION` and target architecture.

## Release architectures

The production build generates:

- `ByFTP-<version>-Linux-amd64.deb`
- `ByFTP-<version>-Linux-arm64.deb`
- `ByFTP-<version>-Linux-i386.deb`

All three packages are built from the same canonical source and root `VERSION`.

## Build

```bash
go telemetry off
bash linux/BUILD.sh
```

The reviewed 1.2.3 production baseline is Go 1.27.0 plus `dpkg-deb`, with Go telemetry disabled. The build is dependency-locked (`GOPROXY=off`, `GOSUMDB=off`, `GOTOOLCHAIN=local`) and uses the shared `build/icon.png` brand asset.

CI and the production release workflow invoke `linux/BUILD.sh` directly. There is no second Linux production-build implementation or compatibility wrapper under `scripts/`.
