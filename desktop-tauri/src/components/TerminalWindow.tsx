import { useEffect, useRef, useState } from "react";
import { Terminal as XTerm } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { Minus, Square, Copy as CopySquares, X, TerminalSquare } from "lucide-react";
import { ipc, onTerminalData, onTerminalExit } from "@/lib/ipc";
import { attachSuggestions } from "@/lib/termSuggest";
import { popoutBufferKey } from "@/lib/popout";
import { useSettings, TERMINAL_THEMES } from "@/stores/settingsStore";
import { cn } from "@/lib/cn";

/// Standalone terminal window — the whole UI of a `?view=terminal` webview
/// (see main.tsx). Either attaches to a PTY handed off by the main window
/// (`?terminal=<id>`, scrollback passed via localStorage) or opens a fresh
/// shell on the given session. Closing the window closes the PTY.
export function TerminalWindow() {
  const params = new URLSearchParams(window.location.search);
  const sessionId = params.get("session") ?? "";
  const title = params.get("title") || "Terminal";
  const handoffId = params.get("terminal");
  const histKey = params.get("histkey");

  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<XTerm | null>(null);
  const terminalIdRef = useRef<string | null>(null);
  const [status, setStatus] = useState<"opening" | "ready" | "exited">(
    handoffId ? "ready" : "opening"
  );
  const [error, setError] = useState<string | null>(null);

  const themeKey = useSettings((s) => s.terminalTheme);
  const theme = TERMINAL_THEMES[themeKey];

  useEffect(() => {
    if (!containerRef.current) return;

    const {
      terminalFontSize,
      terminalFontFamily,
      terminalTheme,
      terminalScrollback,
    } = useSettings.getState();
    const term = new XTerm({
      fontFamily: terminalFontFamily,
      fontSize: terminalFontSize,
      scrollback: terminalScrollback,
      theme: TERMINAL_THEMES[terminalTheme],
      cursorBlink: true,
      allowProposedApi: true,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.loadAddon(new WebLinksAddon());
    term.open(containerRef.current);
    fit.fit();
    termRef.current = term;

    let unlistenData: (() => void) | null = null;
    let unlistenExit: (() => void) | null = null;
    let disposed = false;

    const dataDisposable = term.onData((data) => {
      const id = terminalIdRef.current;
      if (id) ipc.terminalWrite(id, data).catch(() => {});
    });
    const resizeDisposable = term.onResize(({ cols, rows }) => {
      const id = terminalIdRef.current;
      if (id) ipc.terminalResize(id, cols, rows).catch(() => {});
    });
    const selectionDisposable = term.onSelectionChange(() => {
      if (!useSettings.getState().terminalCopyOnSelect) return;
      const text = term.getSelection();
      if (text) navigator.clipboard.writeText(text).catch(() => {});
    });

    const suggest = attachSuggestions(term, {
      historyKey: histKey || sessionId || "default",
      send: (data) => {
        const id = terminalIdRef.current;
        if (id) ipc.terminalWrite(id, data).catch(() => {});
      },
    });

    const onWindowResize = () => {
      try {
        fit.fit();
      } catch {}
    };
    window.addEventListener("resize", onWindowResize);

    // The webview can be torn down without React unmounting (OS window close),
    // so the PTY close also rides on beforeunload — fire-and-forget is fine.
    const onBeforeUnload = () => {
      const id = terminalIdRef.current;
      if (id) ipc.closeTerminal(id).catch(() => {});
      terminalIdRef.current = null;
    };
    window.addEventListener("beforeunload", onBeforeUnload);

    (async () => {
      try {
        unlistenData = await onTerminalData((e) => {
          if (e.terminalId === terminalIdRef.current) term.write(e.data);
        });
        unlistenExit = await onTerminalExit((e) => {
          if (e.terminalId === terminalIdRef.current) {
            setStatus("exited");
            term.writeln(
              `\r\n\x1b[33m[session ended${
                e.code !== null ? ` (exit ${e.code})` : ""
              }]\x1b[0m`
            );
          }
        });
        if (handoffId) {
          // Adopt the PTY the main window handed us. Replay its scrollback
          // first so the terminal doesn't come up blank, then nudge the PTY
          // with our size so full-screen apps repaint.
          const key = popoutBufferKey(handoffId);
          const buffer = localStorage.getItem(key);
          localStorage.removeItem(key);
          if (buffer) term.write(buffer);
          terminalIdRef.current = handoffId;
          ipc.terminalResize(handoffId, term.cols, term.rows).catch(() => {});
          setStatus("ready");
        } else {
          if (!sessionId) throw new Error("No session for this terminal window");
          const id = await ipc.openTerminal(sessionId, term.cols, term.rows);
          terminalIdRef.current = id;
          if (disposed) {
            ipc.closeTerminal(id).catch(() => {});
            return;
          }
          setStatus("ready");
        }
        term.focus();
      } catch (e) {
        setError(String(e));
        setStatus("exited");
      }
    })();

    return () => {
      disposed = true;
      window.removeEventListener("resize", onWindowResize);
      window.removeEventListener("beforeunload", onBeforeUnload);
      suggest.dispose();
      dataDisposable.dispose();
      resizeDisposable.dispose();
      selectionDisposable.dispose();
      if (unlistenData) unlistenData();
      if (unlistenExit) unlistenExit();
      const id = terminalIdRef.current;
      if (id) ipc.closeTerminal(id).catch(() => {});
      term.dispose();
      termRef.current = null;
      terminalIdRef.current = null;
    };
    // Mount once — the window's params never change during its lifetime.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="flex h-screen w-screen flex-col bg-bg">
      <PopoutTitleBar title={title} exited={status === "exited"} />
      <div className="relative flex-1 overflow-hidden">
        <div
          ref={containerRef}
          onMouseDown={() => termRef.current?.focus()}
          className="h-full w-full"
          style={{ background: theme.background }}
        />
        {status === "opening" && (
          <div className="pointer-events-none absolute right-2 top-2 rounded bg-bg-panel/70 px-1.5 py-0.5 text-[10px] text-text-dim">
            opening…
          </div>
        )}
        {error && (
          <div className="absolute inset-x-0 bottom-0 border-t border-border bg-danger-soft px-3 py-1 text-xs text-danger">
            {error}
          </div>
        )}
      </div>
    </div>
  );
}

function PopoutTitleBar({ title, exited }: { title: string; exited: boolean }) {
  const [maximized, setMaximized] = useState(false);

  useEffect(() => {
    const win = getCurrentWindow();
    win.isMaximized().then(setMaximized).catch(() => {});
    const unlisten = win.onResized(() => {
      win.isMaximized().then(setMaximized).catch(() => {});
    });
    return () => {
      unlisten.then((u) => u()).catch(() => {});
    };
  }, []);

  const win = getCurrentWindow();
  return (
    <div
      data-tauri-drag-region
      className="flex h-8 shrink-0 select-none items-center gap-2 border-b border-border bg-bg-panel pl-2.5"
    >
      <TerminalSquare size={13} className="shrink-0 text-accent" />
      <span
        data-tauri-drag-region
        className="min-w-0 truncate text-[12px] font-medium tracking-tight"
      >
        {title}
      </span>
      {exited && (
        <span className="rounded-full bg-bg-hover px-1.5 py-0.5 text-[10px] text-text-dim">
          ended
        </span>
      )}
      <div className="flex-1" data-tauri-drag-region />
      <div className="flex h-8 items-stretch">
        <PopoutWindowButton onClick={() => void win.minimize()} title="Minimize">
          <Minus size={12} />
        </PopoutWindowButton>
        <PopoutWindowButton
          onClick={() => void win.toggleMaximize()}
          title={maximized ? "Restore" : "Maximize"}
        >
          {maximized ? <CopySquares size={11} /> : <Square size={11} />}
        </PopoutWindowButton>
        <PopoutWindowButton onClick={() => void win.close()} title="Close" danger>
          <X size={13} />
        </PopoutWindowButton>
      </div>
    </div>
  );
}

function PopoutWindowButton({
  onClick,
  title,
  danger,
  children,
}: {
  onClick: () => void;
  title: string;
  danger?: boolean;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      title={title}
      className={cn(
        "flex h-8 w-11 items-center justify-center text-text-muted transition-colors",
        danger
          ? "hover:bg-danger hover:text-white"
          : "hover:bg-bg-hover hover:text-text"
      )}
    >
      {children}
    </button>
  );
}
