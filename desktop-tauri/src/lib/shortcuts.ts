// Canonical keyboard-combo representation shared by the shortcut hook, the
// command registry, the title-bar menus and the cheat-sheet. "mod" = Ctrl on
// Windows/Linux and Ctrl on Linux so one registry is cross-platform.

const isMac =
  typeof navigator !== "undefined" &&
  (/mac/i.test(navigator.platform) || /mac/i.test(navigator.userAgent));

export const MOD_LABEL = isMac ? "⌘" : "Ctrl";

// Build the canonical combo for a keydown event, e.g. "mod+shift+t".
export function keyCombo(e: KeyboardEvent): string {
  const parts: string[] = [];
  if (e.ctrlKey || e.metaKey) parts.push("mod");
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
