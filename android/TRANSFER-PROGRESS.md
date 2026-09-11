# Android transfer progress

Ghost FTP Android reports transfer progress from bytes that actually pass through the active upload/download stream. It does not advance a progress value from a timer and it does not invent a total size.

## Byte ownership

For uploads, `ProgressStreams` wraps the local `InputStream`. The counter advances only when the FTP transfer code actually reads bytes from that stream. The selected local entry size is supplied as the expected total when Android's Storage Access Framework reported a positive size.

For downloads, `ProgressStreams` wraps the SAF destination `OutputStream`. The counter advances only when bytes are actually written toward the staged local document. The MLSD file size is supplied as the expected total only when it is positive.

A zero or unavailable size is treated as unknown. In that case the UI can show transferred bytes and a measured rate, but it does not show a percentage or ETA. If observed transferred bytes exceed a claimed total, `TransferProgress` stops treating that total as trustworthy rather than presenting an impossible percentage.

## Status semantics

Progress text such as `Uploading` or `Downloading` describes the data-stream phase only. Reaching 100% of a known byte total is not the same as a completed Ghost FTP operation.

Upload success still requires the existing staged remote commit contract: the server must confirm data completion and accept the final `RNFR`/`RNTO` commit. Download success still requires the staged SAF document to pass the final conflict check, rename to the requested name, and exact `COLUMN_DISPLAY_NAME` read-back.

The final `Upload completed` and `Download completed and committed` statuses remain owned by those commit paths, not by the progress counter.

## Cancellation and stale callbacks

Every progress sink is bound to the existing transfer generation token and the exact active `FtpSession`. A progress callback is ignored when the transfer is no longer active, its generation was cancelled, the visible session changed, or the connection is no longer valid.

This means a cancelled transfer or reconnect cannot be overwritten by late progress from an older worker. Cancellation continues to use the existing fail-closed transport abort and reconnect requirement.

## Security and privacy

Progress accounting is local-only. `TransferProgress` and `ProgressStreams` open no network connection, emit no telemetry, and add no analytics or crash-reporting dependency. FTP/FTPS transport, certificate/hostname verification, passive-data fail-closed behavior, staging, path validation, and credential handling are unchanged by this feature.
