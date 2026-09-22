import type { Terminal as XTerm, IDecoration, IMarker } from "@xterm/xterm";
import { useSettings } from "@/stores/settingsStore";

/// Fish/VS Code-style inline history suggestions for the terminal: as you type
/// a command, the most recent history entry with that prefix is shown as dim
/// "ghost text" after the cursor; → accepts it (sends the rest to the PTY).
///
/// The terminal is a raw PTY pipe — the shell does the line editing, we only
/// see keystrokes going out and bytes coming back. So the current command line
/// is reconstructed heuristically from the outgoing keystrokes, and anything
/// that would desync that model (arrow keys, tab completion, Ctrl+R, escape
/// sequences) simply disables suggestions until the next prompt (Enter/Ctrl+C).
/// Wrong-but-plausible ghosts are worse than none, so every ambiguity resolves
/// to "hide".

/** MRU command history, most recent first, persisted per profile so it
 *  survives reconnects to the same server. */
const HISTORY_PREFIX = "ghostftp.term-history.v1:";
const HISTORY_CAP = 300;
const MAX_CMD_LEN = 300;
/** Suggest only once a couple of chars are down — 1-char prefixes are noise. */
const MIN_PREFIX = 2;

// Shared across panes so two shells on the same server see each other's
// commands immediately (write-through to localStorage).
const historyCache = new Map<string, string[]>();

function loadHistory(key: string): string[] {
  let list = historyCache.get(key);
  if (!list) {
    try {
      const raw = localStorage.getItem(HISTORY_PREFIX + key);
      list = raw ? (JSON.parse(raw) as string[]) : [];
    } catch {
      list = [];
    }
    historyCache.set(key, list);
  }
  return list;
}

function recordCommand(key: string, cmd: string) {
  const list = loadHistory(key).filter((c) => c !== cmd);
  list.unshift(cmd);
  if (list.length > HISTORY_CAP) list.length = HISTORY_CAP;
  historyCache.set(key, list);
  try {
    localStorage.setItem(HISTORY_PREFIX + key, JSON.stringify(list));
  } catch (error) {
    console.warn("Couldn't persist terminal command history", error);
  }
}

function lookupHistory(key: string, prefix: string): string | null {
  for (const cmd of loadHistory(key)) {
    if (cmd.length > prefix.length && cmd.startsWith(prefix)) return cmd;
  }
  return null;
}

export interface SuggestHandle {
  dispose: () => void;
}

/** Wire inline suggestions onto a terminal. `send` writes raw input to the
 *  PTY (used when → accepts a suggestion). Call after `term.open()`; the
 *  handle must be disposed before the terminal is. */
export function attachSuggestions(
  term: XTerm,
  opts: {
    historyKey: string;
    send: (data: string) => void;
    /** App-level chord? Return true to make xterm ignore the key (so it never
     *  reaches the shell) and let it bubble to the app shortcut handler. */
    swallowKey?: (ev: KeyboardEvent) => boolean;
  }
): SuggestHandle {
  const { historyKey, send, swallowKey } = opts;

  /** Best guess of the shell's current input line (cursor assumed at end). */
  let line = "";
  /** True once the line state can't be trusted (tab completion, cursor moves,
   *  Ctrl+R…). Cleared on the next line reset (Enter / Ctrl+C). */
  let dirty = false;
  /** Full history entry currently offered, or null. */
  let suggestion: string | null = null;

  let marker: IMarker | undefined;
  let deco: IDecoration | undefined;
  let raf = 0;
  let disposed = false;

  const enabled = () => useSettings.getState().terminalSuggestions;

  const clearGhost = () => {
    deco?.dispose();
    marker?.dispose();
    deco = undefined;
    marker = undefined;
  };

  // Ghost text is an xterm decoration anchored at the cursor cell, so it
  // scrolls with the buffer and never touches the PTY stream. Re-rendered on
  // every parsed write because the cursor only moves once the shell's echo
  // arrives — positioning at keystroke time would lag one cell behind.
  const renderGhost = () => {
    clearGhost();
    if (!suggestion || disposed) return;
    // Full-screen apps (vim, htop…) run on the alternate buffer — never
    // overlay ghost text there.
    if (term.buffer.active.type === "alternate") return;
    const text = suggestion.slice(line.length);
    const cursorX = term.buffer.active.cursorX;
    const avail = term.cols - cursorX;
    if (!text || avail <= 0) return;
    const clipped = text.length > avail ? text.slice(0, avail - 1) + "…" : text;
    marker = term.registerMarker(0);
    if (!marker || marker.line < 0) return;
    deco = term.registerDecoration({
      marker,
      x: cursorX,
      width: clipped.length,
      layer: "top",
    });
    deco?.onRender((el) => {
      el.textContent = clipped;
      el.style.color = term.options.theme?.foreground ?? "#ffffff";
      el.style.opacity = "0.4";
      el.style.whiteSpace = "pre";
      el.style.overflow = "hidden";
      el.style.pointerEvents = "none";
      el.style.fontFamily = term.options.fontFamily ?? "monospace";
      el.style.fontSize = `${term.options.fontSize ?? 13}px`;
      el.style.lineHeight = el.style.height;
    });
  };

  const scheduleRender = () => {
    cancelAnimationFrame(raf);
    raf = requestAnimationFrame(renderGhost);
  };

  const updateSuggestion = () => {
    suggestion =
      !dirty && enabled() && line.length >= MIN_PREFIX
        ? lookupHistory(historyKey, line)
        : null;
    scheduleRender();
  };

  // On Enter, only record the line if the shell actually echoed it back —
  // checked a beat later against the buffer row the prompt was on. This is
  // what keeps passwords (no echo) and desynced junk out of the history.
  const commit = () => {
    const cmd = line;
    line = "";
    const wasDirty = dirty;
    dirty = false;
    if (wasDirty || !enabled()) return;
    const trimmed = cmd.trim();
    if (
      trimmed.length < MIN_PREFIX ||
      trimmed.length > MAX_CMD_LEN ||
      cmd.startsWith(" ") // bash HISTCONTROL convention: leading space = off the record
    )
      return;
    const at = term.registerMarker(0);
    if (!at) return;
    setTimeout(() => {
      try {
        if (at.line >= 0 && logicalLineAt(term, at.line).includes(trimmed)) {
          recordCommand(historyKey, trimmed);
        }
      } finally {
        at.dispose();
      }
    }, 250);
  };

  const dataDisposable = term.onData((data) => {
    for (let i = 0; i < data.length; i++) {
      const ch = data[i];
      if (ch === "\x1b") {
        // Escape sequence (arrows, home/end, alt-keys, bracketed paste) —
        // the shell is editing in ways we can't see. Give up on this line.
        dirty = true;
        break;
      } else if (ch === "\r" || ch === "\n") {
        commit();
      } else if (ch === "\x7f" || ch === "\b") {
        line = line.slice(0, -1);
      } else if (ch === "\x03") {
        // Ctrl+C abandons the line; a fresh prompt follows.
        line = "";
        dirty = false;
      } else if (ch === "\x15") {
        line = ""; // Ctrl+U kills the line
      } else if (ch === "\x17") {
        line = line.replace(/\s+$/, "").replace(/\S+$/, ""); // Ctrl+W kills a word
      } else if (ch < " ") {
        dirty = true; // tab completion, Ctrl+R, Ctrl+A… — can't track those
      } else {
        line += ch;
      }
    }
    updateSuggestion();
  });

  const writeDisposable = term.onWriteParsed(scheduleRender);

  term.attachCustomKeyEventHandler((ev) => {
    // App-level chords (terminal splits/zoom/close) must not leak into the
    // shell: tell xterm to ignore them so they bubble to the app handler.
    if (swallowKey?.(ev)) return false;
    if (
      ev.type === "keydown" &&
      ev.key === "ArrowRight" &&
      !ev.ctrlKey &&
      !ev.altKey &&
      !ev.metaKey &&
      !ev.shiftKey &&
      deco && // only intercept while a ghost is actually on screen
      suggestion
    ) {
      const suffix = suggestion.slice(line.length);
      if (suffix) {
        send(suffix);
        line += suffix;
        updateSuggestion();
        return false; // swallow the arrow — the shell gets the text instead
      }
    }
    return true;
  });

  return {
    dispose: () => {
      disposed = true;
      cancelAnimationFrame(raf);
      dataDisposable.dispose();
      writeDisposable.dispose();
      clearGhost();
      term.attachCustomKeyEventHandler(() => true);
    },
  };
}

/** The full logical (unwrapped) line of text containing buffer row `row`. */
function logicalLineAt(term: XTerm, row: number): string {
  const buf = term.buffer.active;
  let start = row;
  while (start > 0 && buf.getLine(start)?.isWrapped) start--;
  let end = row;
  while (end + 1 < buf.length && buf.getLine(end + 1)?.isWrapped) end++;
  let text = "";
  for (let i = start; i <= end; i++) {
    text += buf.getLine(i)?.translateToString(true) ?? "";
  }
  return text;
}
