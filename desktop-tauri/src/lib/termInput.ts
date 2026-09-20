// A tiny module-level registry of live terminal panes, so app-level surfaces
// (the snippets palette, the Snippets panel, the terminal toolbar) can inject
// text into the terminal the user last focused — without threading refs down
// through the component tree. Each mounted pane registers a writer + a focus
// hook and reports focus; the inserter targets the most-recently-focused pane
// that is still alive.
//
// This is intentionally separate from React state: the set of live panes is a
// side-effect of what's mounted, and the "focused" pane is a UI fact that no
// store needs to re-render on.

type Writer = (data: string) => void;

interface PaneReg {
  /** Send raw bytes to this pane's PTY (no newline is added by the caller). */
  write: Writer;
  /** Bring keyboard focus back to this pane's terminal. */
  focus: () => void;
}

const panes = new Map<string, PaneReg>();
let focusedPaneId: string | null = null;

/** Register a live pane. Returns an unregister fn for the pane's cleanup. */
export function registerTerminalPane(id: string, reg: PaneReg): () => void {
  panes.set(id, reg);
  if (focusedPaneId === null) focusedPaneId = id;
  return () => {
    panes.delete(id);
    if (focusedPaneId === id) {
      focusedPaneId = panes.size ? panes.keys().next().value ?? null : null;
    }
  };
}

/** Mark `id` as the focused pane (call on focus / when a pane becomes active). */
export function noteTerminalFocus(id: string): void {
  if (panes.has(id)) focusedPaneId = id;
}

/** Is there a live terminal to receive an insertion right now? */
export function hasFocusedTerminal(): boolean {
  return !!focusedPaneId && panes.has(focusedPaneId);
}

/** Inject `text` into the focused terminal's input line and refocus it. Never
 *  appends a newline — the user reviews/edits, then presses Enter. Returns
 *  false if no live terminal is available. */
export function insertIntoTerminal(text: string): boolean {
  const reg = focusedPaneId ? panes.get(focusedPaneId) : null;
  if (!reg) return false;
  reg.write(text);
  reg.focus();
  return true;
}
