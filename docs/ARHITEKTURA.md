# Architecture

ByFTP is organized so protocol/security logic is independent from presentation.

## Main layers

- `cmd/byftp`: product entry point and platform startup.
- `internal/api`: application facade used by desktop/terminal frontends.
- `internal/remote`: FTP/FTPS/SFTP implementations, connection lifecycle and remote operations.
- `internal/transfer`: queue, workers, retries, progress and transfer state.
- `internal/config`: settings/profile persistence and transaction-safe state writes.
- `internal/security`: path, secret, deletion and filesystem safety primitives.
- `internal/desktop`: Windows native UI and Linux/macOS terminal frontend.
- `internal/i18n`: canonical English catalog plus all supported runtime locales.
- `internal/platform`: small OS-specific integration primitives.

## Windows UI modules

The Windows frontend is deliberately split by responsibility: connection lifecycle, profiles, navigation, file mutations, transfer selection, transfer queue state, localization, responsive layout, button rendering and low-level Win32 definitions. This prevents one large UI file from mixing credentials, filesystem mutations and drawing code.

The interface follows a dual-pane file-transfer model: local filesystem on the left, remote server on the right, explicit upload/download actions between them and a persistent transfer queue below. Saved profiles and protocol-specific options are available without overwhelming the main file workflow.

## Protocol implementation

FTP/FTPS uses curl as a child process with controlled configuration input. SFTP uses OpenSSH. Child execution is bounded and lifecycle-managed. Remote and local commit logic revalidates filesystem/server state before final replacement where possible.

## State ownership

Connection generation tokens prevent stale asynchronous callbacks from mutating a newer UI session. Transfer-manager state is authoritative for pause/resume. Profile credentials are identity-bound. These constraints should remain true even when UI code is refactored.