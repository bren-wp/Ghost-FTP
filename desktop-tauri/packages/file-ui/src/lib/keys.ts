// File-browser keyboard actions and their built-in default combos. The package
// stays transport- and app-agnostic: a host can remap these by passing
// `PaneSettings.keyBindings`, but with nothing supplied the pane still works
// with these sensible defaults. Ghost FTP threads its effective (user-overridable)
// bindings in from the shared bindings layer (Plan 15).

export type FileBrowserAction =
  | "rename"
  | "open"
  | "delete"
  | "parentDir"
  | "selectAll"
  | "quickInfo"
  | "newFolder"
  | "refresh";

/** Canonical default combos, in the same "mod+shift+key" shape the rest of Ghost FTP
 *  uses ("mod" = Ctrl on Windows/Linux, Ctrl on Linux). */
export const DEFAULT_FILE_BROWSER_KEYBINDINGS: Record<FileBrowserAction, string> = {
  rename: "f2",
  open: "enter",
  delete: "delete",
  parentDir: "backspace",
  selectAll: "mod+a",
  quickInfo: "space",
  newFolder: "mod+shift+n",
  refresh: "f5",
};

/** Build the canonical combo string for a keydown event, mirroring the app's
 *  `keyCombo` (src/lib/shortcuts.ts) so the two never disagree on a spelling.
 *  Kept local to the package so it carries no app dependency. */
export function comboFromEvent(e: KeyboardEvent): string {
  const parts: string[] = [];
  if (e.ctrlKey || e.metaKey) parts.push("mod");
  if (e.altKey) parts.push("alt");
  if (e.shiftKey) parts.push("shift");
  let k = e.key.toLowerCase();
  if (k === " ") k = "space";
  parts.push(k);
  return parts.join("+");
}
