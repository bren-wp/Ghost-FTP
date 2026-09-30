import type { AppDialog } from "@/stores/layoutStore";

const WORKSPACE_DIALOGS = new Set<AppDialog>([
  "settings",
  "siteManager",
  "transferCenter",
  "sync",
  "help",
  "updates",
  "about",
]);

export function resolveWorkspaceDialog(
  dialog: AppDialog | null,
  returnDialog: AppDialog | null
): AppDialog | null {
  if (dialog && WORKSPACE_DIALOGS.has(dialog)) return dialog;
  if (returnDialog && WORKSPACE_DIALOGS.has(returnDialog)) return returnDialog;
  return null;
}

export function workspaceLabel(
  dialog: AppDialog | null,
  returnDialog: AppDialog | null
): "Files" | "Sites" | "Transfers" | "Sync & Backup" | "Settings" | "Help & About" {
  const active = resolveWorkspaceDialog(dialog, returnDialog);
  if (active === "siteManager") return "Sites";
  if (active === "transferCenter") return "Transfers";
  if (active === "sync") return "Sync & Backup";
  if (active === "settings") return "Settings";
  if (active === "about" || active === "help" || active === "updates") return "Help & About";
  return "Files";
}
