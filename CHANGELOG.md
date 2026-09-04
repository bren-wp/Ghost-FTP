# GhostFTP changelog

## 1.1.0 — 2026-09-04

- Rebuilt the Windows client as a C#-only, dependency-free desktop application.
- Added Windows 11 Fluent/Mica visual treatment and premium dual-pane file management UX.
- Added local Demo mode with realistic folders and transfer operations without network traffic.
- Added FTP, explicit FTPS and implicit FTPS support using the .NET networking stack directly.
- Added upload/download queues, cancellation, recursive folders, rename, delete, new folder and refresh operations.
- Added per-transfer FTP/FTPS sessions so cancelled transfers cannot corrupt the browser control connection.
- Added strict TLS 1.2/1.3 validation with no certificate-bypass option and offline revocation cache checks.
- Added CR/LF command-injection guards, path canonicalization, PASV host hardening, traversal limits and reply-size limits.
- Added safe partial downloads and temporary remote uploads before atomic rename into the destination.
- Added NTFS reparse-point protection for recursive uploads/deletes.
- Added DPAPI-protected optional saved passwords and atomic profile/settings writes.
- Added a C# per-user installer, Start Menu/Desktop integration and Windows uninstall registration.
- Added x64/ARM64 portable + setup release builds with SHA-256 checksums.
- Added CI self-tests and source audits that reject NuGet PackageReference dependencies and known telemetry/tracking SDKs.
- Synchronized product metadata: author Brendigo, ghostftp.com, brendigo.com and version 1.1.0.
