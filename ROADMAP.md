# ByFTP Roadmap

ByFTP is developed as a focused, native and privacy-first Windows FTP / FTPS / SFTP client by Brendigo.

This roadmap is directional. Features, priorities and release dates may change as stability, security and compatibility findings take precedence.

## Current baseline — 2.12.x

- Native Win32 dark desktop interface
- FTP, FTPS Explicit, FTPS Implicit and SFTP
- Password and SFTP private-key authentication
- Windows DPAPI-protected saved profiles
- SFTP host-key fingerprint verification and pinning
- Local/server two-panel file manager
- Multi-select and whole-directory transfers
- Batch file operations and remote permissions
- Transfer queue with 1–8 parallel jobs
- Transient-only retry and Skip Existing mode
- Transactional staging and rollback protections
- Symlink, junction, reparse-point and path traversal defenses
- No telemetry, analytics, advertising SDKs or cloud account requirement

## Planned quality work

### Transfer experience

- More precise live byte progress
- Transfer speed and ETA based on real measured throughput
- Better large-queue filtering and history presentation
- Additional conflict handling options for existing destination files
- More detailed but user-friendly transfer diagnostics

### File manager

- Drag-and-drop workflows where they can be implemented without weakening path protections
- Improved keyboard navigation and contextual actions
- Optional directory comparison tools
- Safer synchronization-oriented workflows
- Additional Site Manager/profile ergonomics

### Connection management

- Faster reconnect workflows after network interruption
- Improved capability detection and presentation
- Better server-specific compatibility handling without external lookup services
- Multiple-session/tab research while preserving resource isolation

### Windows integration

- Windows 10/11 runtime validation matrix
- Additional accessibility and high-DPI validation
- Signed public binaries using the real Brendigo Authenticode identity
- ARM64 evaluation after the x64 release path is fully mature

## Non-goals

ByFTP is not planned to become a cloud control plane, browser dashboard or telemetry-driven product. The project intends to preserve these constraints:

- no browser/localhost application shell
- no analytics or behavioral telemetry
- no advertising SDKs
- no mandatory Brendigo account
- no automatic upload of connection/profile data
- no hidden third-party API dependency for normal FTP/FTPS/SFTP use

## Licensing

This roadmap does not grant rights to implement, redistribute, rebrand or publish derivative versions of ByFTP. The repository is governed by the proprietary [LICENSE](LICENSE).
