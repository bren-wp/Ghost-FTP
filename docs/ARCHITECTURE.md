# Architecture

GhostFTP 1.1 is intentionally dependency-light and Windows-first.

## Projects

- `GhostFTP.Core`: FTP/FTPS protocol engine, listing parsers, transfer queue, demo session and profile persistence abstractions.
- `GhostFTP.App`: Windows desktop UI written programmatically in C# using the WPF framework included with .NET Desktop. No XAML is used.
- `GhostFTP.Setup`: per-user C# installer/uninstaller. No MSI, WiX, NSIS or Inno Setup dependency.
- `GhostFTP.SelfTest`: dependency-free executable self-tests used by CI.

## Dependency policy

There are no `PackageReference` entries and no third-party runtime libraries. The self-contained release bundles the Microsoft .NET Desktop runtime needed to execute GhostFTP.

The application itself never calls GitHub or package feeds. NuGet/GitHub infrastructure may be used by the build environment to obtain Microsoft SDK/runtime packs, but these are build-time concerns and are not network dependencies of the installed application.

## Network boundary

Only `FtpSession` owns FTP/FTPS sockets. Demo mode uses `DemoFtpSession` and does not open sockets. No analytics, telemetry or update service exists in the application architecture.
