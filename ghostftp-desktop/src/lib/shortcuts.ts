// Canonical keyboard-combo representation shared by all shortcut surfaces.
// "mod" = Command on macOS, Control on Windows/Linux. Distinguish macOS
// Control (used by SSH terminals) from Command, and Windows/Super from Ctrl.
// Combining secondary modifiers must not accidentally trigger primary commands.

const isMac =
  typeof navigator !== "undefined" &&
  (/mac/i.test(navigator.platform) || /mac/i.test(navigator.userAgent));

export const MOD_LABEL = isMac ? "⌘" : "Ctrl";

// Build the canonical combo for a keydown event, e.g. "mod+shift+t".
export function keyCombo(e: KeyboardEvent): string {
  const parts: string[] = [];
  if (isMac) {
    if (e.metaKey) parts.push("mod");
    if (e.ctrlKey) parts.push("ctrl");
  } else {
    if (e.ctrlKey) parts.push("mod");
    if (e.metaKey) parts.push("meta");
  }
  if (e.altKey) parts.push("alt");
  if (e.shiftKey) parts.push("shift");
  let k = e.key.toLowerCase();
  if (k === " ") k = "space";
  parts.push(k);
  return parts.join("+");
}

// Human-readable form for menus / cheat-sheet, e.g. "Ctrl+Shift+T".
export function formatCombo(combo: string): string {
  return combo
    .split("+")
    .map((p) => {
      if (p === "mod") return MOD_LABEL;
      if (p === "alt") return isMac ? "⌥" : "Alt";
      if (p === "shift") return isMac ? "⇧" : "Shift";
      if (p === "space") return "Space";
      return p.length === 1
        ? p.toUpperCase()
        : p.charAt(0).toUpperCase() + p.slice(1);
    })
    .join(isMac ? " " : "+");
}
