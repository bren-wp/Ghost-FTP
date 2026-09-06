#!/usr/bin/env python3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def patch(path: str, old: str, new: str, count: int = 1) -> None:
    p = ROOT / path
    text = p.read_text(encoding="utf-8")
    actual = text.count(old)
    if actual != count:
        raise SystemExit(f"{path}: expected {count} occurrence(s), found {actual}: {old[:120]!r}")
    p.write_text(text.replace(old, new, count), encoding="utf-8")


# Windows Setup: keep legacy Uninstall.exe cleanup, but retain/recreate the
# Windows Installed Apps registry entry so the installed GhostFTP.exe can own
# the --uninstall maintenance flow without shipping a second executable.
patch(
    "cmd/installer/main.go",
    '''\n\tif err := platform.DeleteRegistryKey(uninstallKey); err != nil {\n\t\twarnings = append(warnings, "The legacy Windows uninstall registry entry could not be removed.")\n\t}\n''',
    "\n",
)
patch(
    "cmd/installer/main.go",
    '''\tlegacyCleanupWarning := cleanupLegacyUninstaller(dir)\n\n\tlanguageWarning := ""\n''',
    '''\tlegacyCleanupWarning := cleanupLegacyUninstaller(dir)\n\n\tuninstallWarning := ""\n\tif err := registerIntegratedUninstall(appPath, version); err != nil {\n\t\tuninstallWarning = "\\n\\nWindows Installed Apps registration could not be completed. Re-run Setup to restore uninstall support."\n\t}\n\n\tlanguageWarning := ""\n''',
)
patch(
    "cmd/installer/main.go",
    '''\t\tsetupCopy.ReadyBody+legacyCleanupWarning+languageWarning+shortcutWarning+"\\n\\n"+setupCopy.LaunchQuestion,\n''',
    '''\t\tsetupCopy.ReadyBody+legacyCleanupWarning+uninstallWarning+languageWarning+shortcutWarning+"\\n\\n"+setupCopy.LaunchQuestion,\n''',
)

# Linux graphical frontend: route the high-frequency visible shell through the
# same 24-language catalog already used by Windows and the terminal fallback.
replacements = {
    '"FTP / FTPS / SFTP  |  private by design"': 'u.tr("app.subtitle")',
    'badge := "OFFLINE"': 'badge := u.tr("badge.disconnected")',
    'badge = "CONNECTED"': 'badge = u.tr("badge.connected")',
    'badge = "WORKING"': 'badge = u.tr("status.running")',
    'u.drawButton(u.layout.settings, "Settings", !u.busy, false)': 'u.drawButton(u.layout.settings, u.tr("common.settings"), !u.busy, false)',
    'u.x.text(premiumOuterGap, 84, "QUICK CONNECT", premiumTheme.Muted, premiumTheme.Window)': 'u.x.text(premiumOuterGap, 84, strings.ToUpper(u.tr("profile.quick")), premiumTheme.Muted, premiumTheme.Window)',
    'u.drawField(linuxFieldProtocol, "Protocol")': 'u.drawField(linuxFieldProtocol, u.tr("terminal.protocol"))',
    'u.drawField(linuxFieldHost, "Server")': 'u.drawField(linuxFieldHost, u.tr("terminal.server"))',
    'u.drawField(linuxFieldPort, "Port")': 'u.drawField(linuxFieldPort, u.tr("terminal.port"))',
    'u.drawField(linuxFieldUser, "Username")': 'u.drawField(linuxFieldUser, u.tr("terminal.username"))',
    'u.drawField(linuxFieldPassword, "Password")': 'u.drawField(linuxFieldPassword, u.tr("terminal.password"))',
    'u.drawButton(u.layout.connect, "Connect", !u.connected && !u.busy, true)': 'u.drawButton(u.layout.connect, u.tr("common.connect"), !u.connected && !u.busy, true)',
    'profileLabel := "Profiles"': 'profileLabel := u.tr("profile.quick")',
    'u.drawButton(u.layout.saveProfile, "Save profile", u.host != "" && !u.busy, false)': 'u.drawButton(u.layout.saveProfile, u.tr("profile.save"), u.host != "" && !u.busy, false)',
    'u.drawButton(u.layout.removeProfile, "Remove", u.selectedProfileID != "" && !u.busy, false)': 'u.drawButton(u.layout.removeProfile, u.tr("profile.delete"), u.selectedProfileID != "" && !u.busy, false)',
    'u.drawField(linuxFieldKey, "SFTP private key path")': 'u.drawField(linuxFieldKey, u.tr("cue.private_key"))',
    'u.drawField(linuxFieldPassphrase, "Key passphrase")': 'u.drawField(linuxFieldPassphrase, u.tr("cue.passphrase"))',
    'u.drawButton(u.layout.disconnect, "Disconnect", u.connected && !u.busy, false)': 'u.drawButton(u.layout.disconnect, u.tr("common.disconnect"), u.connected && !u.busy, false)',
    'u.x.text(r.left+8, r.top+17, "NAME", premiumTheme.Muted, premiumTheme.List)': 'u.x.text(r.left+8, r.top+17, strings.ToUpper(u.tr("column.name")), premiumTheme.Muted, premiumTheme.List)',
    'u.x.text(r.right-112, r.top+17, "SIZE", premiumTheme.Muted, premiumTheme.List)': 'u.x.text(r.right-112, r.top+17, strings.ToUpper(u.tr("column.size")), premiumTheme.Muted, premiumTheme.List)',
    'u.x.text(premiumOuterGap, leftPanel.top+20, "LOCAL COMPUTER", premiumTheme.Muted, premiumTheme.Panel)': 'u.x.text(premiumOuterGap, leftPanel.top+20, u.tr("section.local"), premiumTheme.Muted, premiumTheme.Panel)',
    'u.x.text(u.layout.remotePath.left, leftPanel.top+20, "SERVER", premiumTheme.Muted, premiumTheme.Panel)': 'u.x.text(u.layout.remotePath.left, leftPanel.top+20, u.tr("section.remote"), premiumTheme.Muted, premiumTheme.Panel)',
    'u.drawField(linuxFieldLocalPath, "Local path")': 'u.drawField(linuxFieldLocalPath, u.tr("column.local"))',
    'u.drawButton(u.layout.localUp, "Up", !u.busy, false)': 'u.drawButton(u.layout.localUp, u.tr("common.up"), !u.busy, false)',
    'u.drawButton(u.layout.localRefresh, "Refresh", !u.busy, false)': 'u.drawButton(u.layout.localRefresh, u.tr("common.refresh"), !u.busy, false)',
    'u.drawField(linuxFieldRemotePath, "Remote path")': 'u.drawField(linuxFieldRemotePath, u.tr("column.remote"))',
    'u.drawButton(u.layout.remoteUp, "Up", u.connected && !u.busy, false)': 'u.drawButton(u.layout.remoteUp, u.tr("common.up"), u.connected && !u.busy, false)',
    'u.drawButton(u.layout.remoteRefresh, "Refresh", u.connected && !u.busy, false)': 'u.drawButton(u.layout.remoteRefresh, u.tr("common.refresh"), u.connected && !u.busy, false)',
    'u.drawButton(u.layout.localNew, "New folder", !u.busy, false)': 'u.drawButton(u.layout.localNew, u.tr("common.new_folder"), !u.busy, false)',
    'u.drawButton(u.layout.localRename, "Rename", u.selectedLocal >= 0 && !u.busy, false)': 'u.drawButton(u.layout.localRename, u.tr("common.rename"), u.selectedLocal >= 0 && !u.busy, false)',
    'u.drawButton(u.layout.localDelete, "Delete", u.selectedLocal >= 0 && !u.busy, false)': 'u.drawButton(u.layout.localDelete, u.tr("common.delete"), u.selectedLocal >= 0 && !u.busy, false)',
    'u.drawButton(u.layout.remoteNew, "New folder", u.connected && !u.busy, false)': 'u.drawButton(u.layout.remoteNew, u.tr("common.new_folder"), u.connected && !u.busy, false)',
    'u.drawButton(u.layout.remoteRename, "Rename", u.connected && u.selectedRemote >= 0 && !u.busy, false)': 'u.drawButton(u.layout.remoteRename, u.tr("common.rename"), u.connected && u.selectedRemote >= 0 && !u.busy, false)',
    'u.drawButton(u.layout.remoteDelete, "Delete", u.connected && u.selectedRemote >= 0 && !u.busy, false)': 'u.drawButton(u.layout.remoteDelete, u.tr("common.delete"), u.connected && u.selectedRemote >= 0 && !u.busy, false)',
    'u.drawButton(u.layout.remoteChmod, "Permissions", u.connected && u.selectedRemote >= 0 && !u.busy, false)': 'u.drawButton(u.layout.remoteChmod, u.tr("common.permissions"), u.connected && u.selectedRemote >= 0 && !u.busy, false)',
    'u.drawButton(u.layout.upload, "Upload ->", u.connected && u.selectedLocal >= 0 && !u.busy, true)': 'u.drawButton(u.layout.upload, u.tr("transfer.upload")+" ->", u.connected && u.selectedLocal >= 0 && !u.busy, true)',
    'u.drawButton(u.layout.download, "<- Download", u.connected && u.selectedRemote >= 0 && !u.busy, true)': 'u.drawButton(u.layout.download, "<- "+u.tr("transfer.download"), u.connected && u.selectedRemote >= 0 && !u.busy, true)',
    'u.x.text(premiumOuterGap, u.layout.pause.top+19, "TRANSFERS", premiumTheme.Muted, premiumTheme.Window)': 'u.x.text(premiumOuterGap, u.layout.pause.top+19, u.tr("section.transfers"), premiumTheme.Muted, premiumTheme.Window)',
    'u.drawButton(u.layout.pause, "Pause", !u.queuePaused, false)': 'u.drawButton(u.layout.pause, u.tr("transfer.pause"), !u.queuePaused, false)',
    'u.drawButton(u.layout.resume, "Resume", u.queuePaused, false)': 'u.drawButton(u.layout.resume, u.tr("transfer.resume"), u.queuePaused, false)',
    'u.drawButton(u.layout.cancelJob, "Cancel", u.selectedTransfer >= 0, false)': 'u.drawButton(u.layout.cancelJob, u.tr("common.cancel"), u.selectedTransfer >= 0, false)',
    'u.drawButton(u.layout.retryJob, "Retry", u.selectedTransfer >= 0, false)': 'u.drawButton(u.layout.retryJob, u.tr("transfer.retry"), u.selectedTransfer >= 0, false)',
    'u.drawButton(u.layout.clearQueue, "Clear done", len(u.transferJobs) > 0, false)': 'u.drawButton(u.layout.clearQueue, u.tr("transfer.clear"), len(u.transferJobs) > 0, false)',
    'u.x.text(u.layout.queue.left+8, u.layout.queue.top+17, "DIRECTION   LOCAL / SERVER", premiumTheme.Muted, premiumTheme.List)': 'u.x.text(u.layout.queue.left+8, u.layout.queue.top+17, strings.ToUpper(u.tr("column.direction")+"   "+u.tr("column.local")+" / "+u.tr("column.remote")), premiumTheme.Muted, premiumTheme.List)',
    'u.setStatus("Connecting securely...")': 'u.setStatus(u.tr("connection.connecting", u.host))',
    'u.setStatus("Disconnecting and cancelling active transfers...")': 'u.setStatus(u.tr("disconnect.progress"))',
    'u.setStatus("Transfer queued.")': 'u.setStatus(u.tr("status.queued"))',
    'u.setStatus("Transfer queue paused.")': 'u.setStatus(u.tr("transfer.pause"))',
    'u.setStatus("Transfer queue resumed.")': 'u.setStatus(u.tr("transfer.resume"))',
    'u.setStatus("Disconnected.")': 'u.setStatus(u.tr("disconnect.done"))',
}
for old, new in replacements.items():
    patch("internal/desktop/gui_linux.go", old, new)

# Remove the obsolete Web-source wording from the shared desktop theme source.
patch(
    "internal/desktop/theme.go",
    "// These are the canonical Ghost FTP product tokens. They intentionally mirror\n// GhostFTP WEB/assets/css/app.css so native Windows, native Linux and the Web\n// companion remain one visual system without importing a GUI framework.\n",
    "// Canonical Ghost FTP desktop tokens shared by native Windows and Linux.\n// The desktop product owns this palette directly; no Web/PWA runtime or GUI\n// framework is required to render it.\n",
)

print("GHOST_FTP_020_DESKTOP_POLISH=APPLIED")
