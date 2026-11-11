// Windows/Linux shortcuts: Control is primary. Super never aliases Control.
export const MOD_LABEL = "Ctrl";

export function keyCombo(event: KeyboardEvent): string {
  const parts: string[] = [];
  if (event.ctrlKey) parts.push("mod");
  if (event.metaKey) parts.push("meta");
  if (event.altKey) parts.push("alt");
  if (event.shiftKey) parts.push("shift");
  const key = event.key.toLowerCase() === " " ? "space" : event.key.toLowerCase();
  return [...parts, key].join("+");
}

export function formatCombo(combo: string): string {
  return combo.split("+").map(part => {
    if (part === "mod") return MOD_LABEL;
    if (part === "space") return "Space";
    return part.length === 1 ? part.toUpperCase() : part.charAt(0).toUpperCase() + part.slice(1);
  }).join("+");
}
