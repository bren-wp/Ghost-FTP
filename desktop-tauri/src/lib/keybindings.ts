import { useMemo } from "react";
import { useCommands, type Command } from "./commands";
import { useBindings } from "@/stores/bindingsStore";

// The one resolver every surface reads (Plan 15 Phase 1). Default combos live in
// the command registry; the bindings store holds user overrides. Effective combo
// = override ?? default. Keeping this in a single place is what stops the
// palette, the title-bar menus, the cheat-sheet and the dispatcher from drifting
// apart onto their own private copies of a combo.

/** Resolve one id's effective combo. `""` in the overrides map means the command
 *  is explicitly unbound (no shortcut); an absent id falls back to `def`. */
export function effectiveCombo(
  id: string,
  def: string | undefined,
  overrides: Record<string, string>
): string | undefined {
  if (Object.prototype.hasOwnProperty.call(overrides, id)) {
    return overrides[id] || undefined;
  }
  return def;
}

/** The command registry with each command's `combo` swapped for its effective
 *  binding. Consumers just read `c.combo` and get the resolved value. */
export function useResolvedCommands(): Command[] {
  const commands = useCommands();
  const overrides = useBindings((s) => s.overrides);
  return useMemo(
    () =>
      commands.map((c) => {
        const eff = effectiveCombo(c.id, c.combo, overrides);
        return eff === c.combo ? c : { ...c, combo: eff };
      }),
    [commands, overrides]
  );
}
