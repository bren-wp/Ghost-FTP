# Ghost FTP Remote Desktop

Ghost FTP includes an **Advanced → Remote Desktop** launcher for users who manage the same infrastructure with file transfer and RDP.

## Security boundary

Ghost FTP does not implement the RDP protocol and does not proxy Remote Desktop traffic. It validates only a server target in `host[:port]` form and delegates the session to the operating system or an installed RDP client.

**RDP credentials are not stored by Ghost FTP.** Passwords, tokens, saved RDP credentials and certificate-bypass flags are not added to command lines, profile storage, logs or Android intents. Authentication remains inside the selected RDP client.

The default port is **3389**. A custom port must be in the range `1..65535`. Targets containing control characters, whitespace, URL paths, user-info syntax or other ambiguous separators are rejected before launch.

RDP is intentionally separate from Ghost FTP's FTP/FTPS/SFTP protocol selector. An RDP target is not an FTP profile and does not inherit FTP passwords, SFTP private keys or host-key trust state.

## Windows

Windows opens the platform Remote Desktop client directly:

```text
mstsc.exe /v:host:port
```

Ghost FTP starts `mstsc.exe` without a shell. No `/password`, command-shell wrapper or credential argument is generated.

## Linux

Linux looks for **FreeRDP** in this order:

```text
xfreerdp3
xfreerdp
```

The launcher supplies only the validated `/v:host:port` target. Ghost FTP does not add `/p:`, `/cert:ignore`, `/cert:` or shell-evaluated arguments. If FreeRDP is not installed, Ghost FTP reports that requirement instead of silently falling back to an insecure command.

## Android

Android delegates a validated target through a standard view intent:

```text
rdp://host:port
```

Ghost FTP first checks whether an installed application has registered support for the RDP URI. If no compatible RDP client exists, the app reports that state and does not open a web fallback or transmit the target to a Ghost FTP service.

No username or password is embedded in the RDP URI and no RDP credential is added as an intent extra.

## Relationship to file-transfer security

Remote Desktop launch support does not change Ghost FTP's existing transfer trust model:

- FTPS certificate and hostname verification remain enabled.
- SFTP on desktop continues to require strict host-key verification/pinning.
- Android SFTP remains intentionally unavailable until equivalent host-key verification is implemented.
- Local path/root confinement and Android SAF capability boundaries are unchanged.
- Ghost FTP still includes no telemetry, advertising, hidden proxy or mandatory cloud backend.

## Support expectations

Ghost FTP owns target validation and the safe hand-off to the platform client. Session rendering, RDP authentication, server-side policy, RDP certificates and client-specific options are owned by `mstsc.exe`, FreeRDP or the Android RDP application selected by the operating system.
