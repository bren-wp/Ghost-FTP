# Architecture

ByFTP is a Go application split into small packages with typed boundaries.

`cmd/byftp` starts the client, `cmd/installer` and `cmd/uninstaller` own the Windows lifecycle, `internal/api` exposes the application engine, `internal/desktop` contains native/terminal presentation, `internal/remote` owns protocol sessions, `internal/transfer` owns queue state, `internal/config` owns durable settings and profiles, `internal/security` centralizes hardening, `internal/i18n` owns localization and `internal/platform` isolates operating-system primitives.

The client does not use a hidden localhost web service or generic remote command bridge. Network access is limited to endpoints selected by the user and operating-system tooling required by the chosen protocol.
