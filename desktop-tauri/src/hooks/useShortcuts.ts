import { useEffect, useRef } from "react";
import { useResolvedCommands } from "@/lib/keybindings";
import { useLayout } from "@/stores/layoutStore";
import { keyCombo } from "@/lib/shortcuts";

/** True when the keystroke is being typed into an editable surface — a form
 *  control, a contenteditable, or xterm's hidden helper textarea. Bare-key
 *  shortcuts must never fire here (they'd eat the user's typing); modifier
 *  combos still do, so Ctrl+T etc. keep working from a focused input. */
function isEditableTarget(el: EventTarget | null): boolean {
  const t = el as HTMLElement | null;
  if (!t || !t.tagName) return false;
  const tag = t.tagName;
  if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return true;
  if (t.isContentEditable) return true;
  // xterm's textarea is a TEXTAREA (caught above), but guard the container too.
  if (typeof t.closest === "function" && t.closest(".xterm")) return true;
  return false;
}

// One global keydown listener dispatching the (override-resolved) command
// registry. It fires modifier combos everywhere and bare-key combos only when
// focus isn't in an editable surface (Plan 15 Phase 3). File-browser keys are
// deliberately absent here — <FilePane> owns those locally so they can't collide
// with anything global.
export function useShortcuts() {
  const commands = useResolvedCommands();
  const togglePalette = useLayout((s) => s.togglePalette);

  // Keep the latest commands without re-binding the listener every render.
  const ref = useRef(commands);
  ref.current = commands;

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const combo = keyCombo(e);
      const hasMod = e.ctrlKey || e.metaKey;
      // The palette toggle is a fixed binding (not a registry command).
      if (combo === "mod+k") {
        e.preventDefault();
        togglePalette();
        return;
      }
      // Guard the user's typing: bare-key combos don't fire from an input/
      // textarea/terminal. Modifier combos stay global.
      if (!hasMod && isEditableTarget(e.target)) return;
      const cmd = ref.current.find(
        (c) => c.combo === combo && c.enabled !== false
      );
      if (cmd) {
        e.preventDefault();
        cmd.run();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [togglePalette]);
}
