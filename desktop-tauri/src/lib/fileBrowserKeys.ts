import { useMemo } from "react";
import {
  DEFAULT_FILE_BROWSER_KEYBINDINGS,
  type FileBrowserAction,
} from "@ghostftp/file-ui";
import { useBindings } from "@/stores/bindingsStore";
import { effectiveCombo } from "./keybindings";

// The file-browser keys are handled locally inside <FilePane> (they need the
// pane's live selection/anchor), but they're remappable through the same
// bindings layer as every other shortcut: this catalog gives each pane action a
// stable override id + display metadata for the Keyboard settings tab and the
// cheat-sheet, and `useFileBrowserKeyBindings` threads the effective combos back
// into the pane via PaneSettings.keyBindings (Plan 15 Phase 3).

export const FILE_BROWSER_GROUP = "File browser";

export interface FileBrowserKeySpec {
  /** ghostftp.db override id (namespaced so it never collides with a command id). */
  id: string;
  action: FileBrowserAction;
  title: string;
  defaultCombo: string;
}

export const FILE_BROWSER_KEYS: FileBrowserKeySpec[] = [
  { id: "fb.open", action: "open", title: "Open / enter folder", defaultCombo: DEFAULT_FILE_BROWSER_KEYBINDINGS.open },
  { id: "fb.rename", action: "rename", title: "Rename", defaultCombo: DEFAULT_FILE_BROWSER_KEYBINDINGS.rename },
  { id: "fb.delete", action: "delete", title: "Delete selection", defaultCombo: DEFAULT_FILE_BROWSER_KEYBINDINGS.delete },
  { id: "fb.parentDir", action: "parentDir", title: "Up one folder", defaultCombo: DEFAULT_FILE_BROWSER_KEYBINDINGS.parentDir },
  { id: "fb.selectAll", action: "selectAll", title: "Select all", defaultCombo: DEFAULT_FILE_BROWSER_KEYBINDINGS.selectAll },
  { id: "fb.quickInfo", action: "quickInfo", title: "Quick info (properties)", defaultCombo: DEFAULT_FILE_BROWSER_KEYBINDINGS.quickInfo },
  { id: "fb.newFolder", action: "newFolder", title: "New folder", defaultCombo: DEFAULT_FILE_BROWSER_KEYBINDINGS.newFolder },
  { id: "fb.refresh", action: "refresh", title: "Refresh", defaultCombo: DEFAULT_FILE_BROWSER_KEYBINDINGS.refresh },
];

/** Compute the effective per-action combo map for <FilePane>. An unbound action
 *  (`effectiveCombo` → undefined) is passed as "" so the pane's default doesn't
 *  quietly re-apply. */
export function fileBrowserKeyBindings(
  overrides: Record<string, string>
): Partial<Record<FileBrowserAction, string>> {
  const out: Partial<Record<FileBrowserAction, string>> = {};
  for (const spec of FILE_BROWSER_KEYS) {
    out[spec.action] = effectiveCombo(spec.id, spec.defaultCombo, overrides) ?? "";
  }
  return out;
}

export function useFileBrowserKeyBindings(): Partial<
  Record<FileBrowserAction, string>
> {
  const overrides = useBindings((s) => s.overrides);
  return useMemo(() => fileBrowserKeyBindings(overrides), [overrides]);
}
