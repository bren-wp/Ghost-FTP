// Credential-free Ghost FTP CLI recipes for saved folder-sync pairs.
// Reject unsafe or unsupported semantics rather than silently dropping them.
export type SyncScriptFormat = "powershell" | "cmd" | "dotnet-watcher";
export interface ScriptPair {
  name: string;
  profileId: string;
  localRoot: string;
  remoteRoot: string;
  direction: "localToRemote" | "remoteToLocal";
  strategy: "additive" | "mirror";
  mode: "mirror" | "onDemand";
  exclude: string[];
}
function safeText(value: string, field: string): string {
  if (!value || value.length > 4096 || /[\x00-\x1f\x7f]/.test(value)) {
    throw new Error(field + " contains an empty, overlong or control-character value.");
  }
  return value;
}
function psLiteral(value: string): string {
  return "'" + value.replace(/'/g, "''") + "'";
}
function cmdLiteral(value: string): string {
  // CMD expands %VAR% and delayed !VAR! and interprets metacharacters.
  // Refuse ambiguous input instead of trying fragile escaping.
  if (/[%!^"<>&|]/.test(value)) {
    throw new Error("CMD cannot safely represent this path. Use PowerShell.");
  }
  return '"' + value + '"';
}
export function generateSyncScript(pair: ScriptPair, format: SyncScriptFormat): string {
  if (pair.mode === "onDemand") {
    throw new Error("On-demand placeholders require the Ghost FTP application.");
  }
  if (pair.strategy !== "additive") {
    throw new Error("Mirror may delete files; use the in-app preview and confirmation.");
  }
  if (pair.exclude.length > 0) {
    throw new Error("CLI cannot apply this pair's exclude patterns; use the in-app watcher.");
  }
  const profile = safeText(pair.profileId, "Profile ID");
  if (!/^[A-Za-z0-9_-]{1,128}$/.test(profile)) {
    throw new Error("Saved Profile ID is not safe to use in a script.");
  }
  const local = safeText(pair.localRoot, "Local path");
  const remoteRoot = safeText(pair.remoteRoot, "Remote path");
  if (!remoteRoot.startsWith("/") || remoteRoot.includes("\\")) {
    throw new Error("CLI requires an absolute POSIX-style remote directory.");
  }
  const remote = profile + ":" + remoteRoot;
  const direction = pair.direction === "localToRemote" ? "push" : "pull";
  if (format === "cmd") {
    return [
      "@echo off",
      "setlocal DisableDelayedExpansion",
      "rem Ghost FTP CLI — additive one-way sync. No credentials stored in this script.",
      "rem Requires ghostftp-cli.exe in PATH and a saved site profile.",
      "ghostftp-cli.exe sync " + cmdLiteral(local) + " " + cmdLiteral(remote) + " --direction " + direction,
      "if errorlevel 1 (",
      "  >&2 echo Ghost FTP sync failed. Nothing was mirror-deleted.",
      "  exit /b 1",
      ")",
      "exit /b 0",
      "",
    ].join("\r\n");
  }
  const common = [
    "# Ghost FTP — additive sync from an existing saved profile, without secrets.",
    "# Requires ghostftp-cli in PATH. This script never embeds credentials.",
    "$ErrorActionPreference = 'Stop'",
    "$localRoot = " + psLiteral(local),
    "$remoteRoot = " + psLiteral(remote),
    "$direction = " + psLiteral(direction),
    "$exe = (Get-Command 'ghostftp-cli' -ErrorAction Stop).Source",
    "function Invoke-GhostSync {",
    "  & $exe sync $localRoot $remoteRoot --direction $direction",
    "  if ($LASTEXITCODE -ne 0) { throw 'Ghost FTP CLI sync failed.' }",
    "}",
  ];
  if (format === "powershell") {
    return [...common, "Invoke-GhostSync", ""].join("\r\n");
  }
  if (format !== "dotnet-watcher") throw new Error("Unsupported script format.");
  if (direction !== "push") {
    throw new Error("The .NET watcher supports local-to-remote uploads only.");
  }
  return [
    ...common,
    "# .NET FileSystemWatcher: debounce changes; 60-second fallback rescan.",
    "$watcher = [System.IO.FileSystemWatcher]::new($localRoot)",
    "$watcher.IncludeSubdirectories = $true",
    "$watcher.NotifyFilter = [System.IO.NotifyFilters]'FileName, DirectoryName, LastWrite, Size'",
    "$prefix = 'GhostFTP-' + [Guid]::NewGuid().ToString('N')",
    "$names = @('Created', 'Changed', 'Renamed', 'Deleted', 'Error')",
    "try {",
    "  foreach ($name in $names) {",
    "    Register-ObjectEvent -InputObject $watcher -EventName $name -SourceIdentifier ($prefix + '-' + $name) | Out-Null",
    "  }",
    "  $watcher.EnableRaisingEvents = $true",
    "  Invoke-GhostSync",
    "  while ($true) {",
    "    $event = Wait-Event -Timeout 60",
    "    if ($null -ne $event) {",
    "      Start-Sleep -Milliseconds 800",
    "      Get-Event | Where-Object { $_.SourceIdentifier.StartsWith($prefix) } | Remove-Event",
    "    }",
    "    # Event-driven or periodic: additive only, never remote-delete.",
    "    Invoke-GhostSync",
    "  }",
    "} finally {",
    "  $watcher.EnableRaisingEvents = $false",
    "  foreach ($name in $names) { Unregister-Event -SourceIdentifier ($prefix + '-' + $name) -ErrorAction SilentlyContinue }",
    "  $watcher.Dispose()",
    "}",
    "",
  ].join("\r\n");
}
