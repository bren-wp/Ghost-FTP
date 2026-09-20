import { WebviewWindow } from "@tauri-apps/api/webviewWindow";
import type { SessionId } from "@/lib/types";

/// A popped-out terminal is a second Tauri window running this same SPA with
/// `?view=terminal` — main.tsx branches on it and renders <TerminalWindow />
/// instead of the full app shell. Window labels must match the `terminal-*`
/// glob in src-tauri/capabilities/default.json or the new webview gets no
/// core permissions (no event listen, no window controls).

export interface TerminalWindowOpts {
  sessionId: SessionId;
  title: string;
  /** Hand off an already-open PTY. Omitted → the new window opens its own. */
  terminalId?: string;
  /** Key for the inline-suggestion history (the profile id) — passed along so
   *  the popped-out shell keeps suggesting from the same server's history. */
  historyKey?: string;
}

/** localStorage key used to pass the serialized scrollback to the new window
 *  (same origin → shared storage; too large to fit in the window URL). */
const POPOUT_BUFFER_PREFIX = "ghostftp-popout-buffer:";

export function popoutBufferKey(terminalId: string) {
  return `${POPOUT_BUFFER_PREFIX}${terminalId}`;
}

/** Remove handoff buffers a crashed/never-loaded popout left behind. Called a
 *  little after main-window startup, when any live popout has long since
 *  consumed its buffer. */
export function sweepStalePopoutBuffers() {
  try {
    for (let i = localStorage.length - 1; i >= 0; i--) {
      const k = localStorage.key(i);
      if (k && k.startsWith(POPOUT_BUFFER_PREFIX)) localStorage.removeItem(k);
    }
  } catch {}
}

export async function openTerminalWindow(
  opts: TerminalWindowOpts
): Promise<WebviewWindow> {
  const qs = new URLSearchParams({
    view: "terminal",
    session: opts.sessionId,
    title: opts.title,
  });
  if (opts.terminalId) qs.set("terminal", opts.terminalId);
  if (opts.historyKey) qs.set("histkey", opts.historyKey);

  const win = new WebviewWindow(`terminal-${crypto.randomUUID().slice(0, 8)}`, {
    url: `index.html?${qs.toString()}`,
    title: `${opts.title} — Ghost FTP`,
    width: 900,
    height: 560,
    minWidth: 480,
    minHeight: 300,
    decorations: false,
  });
  await new Promise<void>((resolve, reject) => {
    void win.once("tauri://created", () => resolve());
    void win.once("tauri://error", (e) =>
      reject(new Error(`Could not open terminal window: ${JSON.stringify(e.payload)}`))
    );
  });
  return win;
}
