import { useEffect, useRef, useState } from "react";
import {
  Plus,
  X,
  TerminalSquare,
  PictureInPicture2,
  Braces,
  Settings2,
  SplitSquareHorizontal,
  SplitSquareVertical,
  Maximize2,
} from "lucide-react";
import type { SessionId } from "@/lib/types";
import {
  acquirePane,
  getPane,
  type PaneState,
} from "@/lib/terminalRegistry";
import { noteTerminalFocus } from "@/lib/termInput";
import { useSettings, TERMINAL_THEMES } from "@/stores/settingsStore";
import { useConnections } from "@/stores/connectionsStore";
import { useSnippets } from "@/stores/snippetsStore";
import {
  useTerminals,
  type TerminalTab,
  type PaneNode,
  type SplitDir,
} from "@/stores/terminalsStore";
import { openTerminalWindow, popoutBufferKey } from "@/lib/popout";
import { toast } from "@/stores/toastStore";
import { ContextMenu, type MenuItem } from "./ContextMenu";
import { cn } from "@/lib/cn";

/// Tabbed terminal dock. Each tab owns a LAYOUT TREE of panes (Plan 11): a leaf
/// is one shell (its own PTY channel on the shared SSH connection), a split
/// arranges two subtrees. Every tab across all sessions stays mounted; only the
/// active tab's tree is visible. xterm instances live in the terminal registry
/// (outside React) so scrollback survives tab switches, dock toggles, splits,
/// popouts, and HMR — components only attach/detach the cached node.
export function TerminalDock({
  sessionId,
  visible,
}: {
  sessionId: SessionId | null;
  visible: boolean;
}) {
  const tabs = useTerminals((s) => s.tabs);
  const activeId = useTerminals((s) => s.activeId);
  const openTab = useTerminals((s) => s.openTab);
  const closeTab = useTerminals((s) => s.closeTab);
  const setActive = useTerminals((s) => s.setActive);
  const renameTab = useTerminals((s) => s.renameTab);
  const splitActivePane = useTerminals((s) => s.splitActivePane);
  const toggleZoom = useTerminals((s) => s.toggleZoom);
  const snippets = useSnippets((s) => s.snippets);
  const requestInsert = useSnippets((s) => s.requestInsert);
  const openSnippets = useSnippets((s) => s.openPanel);
  const [menu, setMenu] = useState<{ x: number; y: number; items: MenuItem[] } | null>(
    null
  );

  // Move the active pane's shell into its own window: hand the live PTY over
  // (scrollback serialized through localStorage) and drop it here.
  const popOut = async (tab: TerminalTab) => {
    const entry = getPane(tab.activePaneId);
    const terminalId = entry?.getTerminalId() ?? null;
    if (entry && terminalId) {
      try {
        const buffer = entry.serialize();
        if (buffer) localStorage.setItem(popoutBufferKey(terminalId), buffer);
      } catch {}
      entry.setHandedOff(true);
    }
    try {
      await openTerminalWindow({
        sessionId: tab.sessionId,
        title: tab.title,
        terminalId: terminalId ?? undefined,
        historyKey: useConnections
          .getState()
          .sessions.find((x) => x.sessionId === tab.sessionId)?.profileId,
      });
      // Dispose won't close the handed-off PTY; the new window owns it now.
      useTerminals.getState().closePane(tab.id, tab.activePaneId);
    } catch (e) {
      if (entry && terminalId) {
        entry.setHandedOff(false);
        localStorage.removeItem(popoutBufferKey(terminalId));
      }
      toast.error("Pop out failed", String(e));
    }
  };

  const tabMenuItems = (tab: TerminalTab): MenuItem[] => [
    {
      label: "Split right",
      icon: <SplitSquareHorizontal size={12} />,
      onClick: () => splitActivePane(tab.id, "row"),
    },
    {
      label: "Split down",
      icon: <SplitSquareVertical size={12} />,
      onClick: () => splitActivePane(tab.id, "col"),
      separatorAfter: true,
    },
    {
      label: "Pop out active pane",
      icon: <PictureInPicture2 size={12} />,
      // No PTY id yet (shell still opening) → nothing to hand off.
      disabled: !getPane(tab.activePaneId)?.getTerminalId(),
      onClick: () => void popOut(tab),
      separatorAfter: true,
    },
    {
      label: "Close",
      icon: <X size={12} />,
      destructive: true,
      onClick: () => closeTab(tab.id),
    },
  ];

  const sessionTabs = sessionId
    ? tabs.filter((t) => t.sessionId === sessionId)
    : [];

  const openSplitMenu = (e: React.MouseEvent) => {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    if (!activeId) return;
    setMenu({
      x: r.left,
      y: r.bottom + 4,
      items: [
        {
          label: "Split right",
          icon: <SplitSquareHorizontal size={12} />,
          onClick: () => splitActivePane(activeId, "row"),
        },
        {
          label: "Split down",
          icon: <SplitSquareVertical size={12} />,
          onClick: () => splitActivePane(activeId, "col"),
          separatorAfter: true,
        },
        {
          label: "Zoom active pane",
          icon: <Maximize2 size={12} />,
          onClick: () => toggleZoom(activeId),
        },
      ],
    });
  };

  const openSnippetMenu = (e: React.MouseEvent) => {
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const items: MenuItem[] = [];
    if (snippets.length === 0) {
      items.push({ label: "No snippets yet", disabled: true, onClick: () => {} });
    } else {
      snippets.slice(0, 12).forEach((s, i, arr) => {
        items.push({
          label: s.name || "(unnamed)",
          icon: <Braces size={12} />,
          onClick: () => requestInsert(s),
          separatorAfter: i === arr.length - 1,
        });
      });
    }
    items.push({
      label: "Manage snippets…",
      icon: <Settings2 size={12} />,
      onClick: () => openSnippets(),
    });
    setMenu({ x: r.left, y: r.bottom + 4, items });
  };

  // Open a first shell when the dock is shown for a session that has none.
  useEffect(() => {
    if (visible && sessionId && sessionTabs.length === 0) {
      openTab(sessionId);
    }
  }, [visible, sessionId, sessionTabs.length, openTab]);

  // While shown, keep the active tab pointed at one in the focused session.
  useEffect(() => {
    if (!visible || !sessionId) return;
    if (!activeId || !sessionTabs.some((t) => t.id === activeId)) {
      const first = sessionTabs[0];
      if (first) setActive(first.id);
    }
  }, [visible, sessionId, sessionTabs, activeId, setActive]);

  return (
    <div className="flex h-full flex-col">
      <div className="flex h-8 shrink-0 items-center gap-1 border-b border-border bg-bg-subtle px-2">
        <div className="flex min-w-0 flex-1 items-center gap-1 overflow-x-auto">
          {sessionTabs.map((tab) => (
            <TabChip
              key={tab.id}
              tab={tab}
              active={tab.id === activeId}
              onClick={() => setActive(tab.id)}
              onClose={() => closeTab(tab.id)}
              onRename={(title) => renameTab(tab.id, title)}
              onContextMenu={(e) => {
                e.preventDefault();
                setMenu({ x: e.clientX, y: e.clientY, items: tabMenuItems(tab) });
              }}
            />
          ))}
          {sessionId && (
            <button
              onClick={() => openTab(sessionId)}
              title="New shell"
              className="ml-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-text-muted hover:bg-bg-hover hover:text-text"
            >
              <Plus size={12} />
            </button>
          )}
        </div>
        {sessionId && sessionTabs.length > 0 && (
          <button
            onClick={openSplitMenu}
            title="Split terminal"
            className="flex h-6 shrink-0 items-center justify-center rounded-md px-1.5 text-text-muted hover:bg-bg-hover hover:text-text"
          >
            <SplitSquareHorizontal size={13} />
          </button>
        )}
        {sessionId && (
          <button
            onClick={openSnippetMenu}
            title="Insert snippet"
            className="flex h-6 shrink-0 items-center justify-center rounded-md px-1.5 text-text-muted hover:bg-bg-hover hover:text-text"
          >
            <Braces size={13} />
          </button>
        )}
      </div>
      <div className="relative flex-1 overflow-hidden">
        {/* Every tab stays mounted; only the focused session's active tab is
            shown. Zoom collapses the view to one pane (others stay alive in the
            registry). This is what keeps background shells alive. */}
        {tabs.map((tab) => {
          const isVisible = visible && tab.id === activeId;
          const node: PaneNode = tab.zoomedPaneId
            ? { kind: "leaf", paneId: tab.zoomedPaneId }
            : tab.root;
          return (
            <div
              key={tab.id}
              className={cn(
                "absolute inset-0",
                isVisible ? "" : "invisible pointer-events-none"
              )}
              aria-hidden={!isVisible}
            >
              <PaneView tab={tab} node={node} visible={isVisible} />
            </div>
          );
        })}
      </div>
      {menu && (
        <ContextMenu
          x={menu.x}
          y={menu.y}
          items={menu.items}
          onClose={() => setMenu(null)}
        />
      )}
    </div>
  );
}

function TabChip({
  tab,
  active,
  onClick,
  onClose,
  onRename,
  onContextMenu,
}: {
  tab: TerminalTab;
  active: boolean;
  onClick: () => void;
  onClose: () => void;
  onRename: (title: string) => void;
  onContextMenu: (e: React.MouseEvent) => void;
}) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(tab.title);

  if (editing) {
    return (
      <div
        className={cn(
          "flex h-6 shrink-0 items-center gap-1 rounded-md px-1.5 ring-1 ring-inset",
          active ? "bg-bg-panel ring-accent/40" : "bg-bg-panel ring-border"
        )}
      >
        <TerminalSquare size={11} className="text-text-muted" />
        <input
          autoFocus
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onBlur={() => {
            onRename(draft.trim() || tab.title);
            setEditing(false);
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              onRename(draft.trim() || tab.title);
              setEditing(false);
            } else if (e.key === "Escape") {
              setDraft(tab.title);
              setEditing(false);
            }
          }}
          className="w-20 bg-transparent text-[11px] outline-none"
        />
      </div>
    );
  }

  return (
    <button
      onClick={onClick}
      onDoubleClick={() => {
        setDraft(tab.title);
        setEditing(true);
      }}
      onContextMenu={onContextMenu}
      title={`${tab.title} — double-click to rename`}
      className={cn(
        "group/tab flex h-6 shrink-0 items-center gap-1.5 rounded-md px-2 text-[11px] transition-colors",
        active
          ? "bg-bg-panel text-text ring-1 ring-inset ring-accent/30"
          : "text-text-muted hover:bg-bg-hover hover:text-text"
      )}
    >
      <TerminalSquare size={11} className={cn("shrink-0", active && "text-accent")} />
      <span className="max-w-[18rem] truncate">{tab.title}</span>
      <span
        role="button"
        tabIndex={-1}
        onClick={(e) => {
          e.stopPropagation();
          onClose();
        }}
        className="ml-0.5 flex h-4 w-4 items-center justify-center rounded text-text-dim opacity-0 hover:bg-bg-hover hover:text-text group-hover/tab:opacity-100"
      >
        <X size={10} />
      </span>
    </button>
  );
}

// ---- Layout tree rendering ----

function PaneView({
  tab,
  node,
  visible,
}: {
  tab: TerminalTab;
  node: PaneNode;
  visible: boolean;
}) {
  if (node.kind === "leaf") {
    return <TerminalLeaf tab={tab} paneId={node.paneId} visible={visible} />;
  }
  return <SplitView tab={tab} node={node} visible={visible} />;
}

function SplitView({
  tab,
  node,
  visible,
}: {
  tab: TerminalTab;
  node: Extract<PaneNode, { kind: "split" }>;
  visible: boolean;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const setSplitRatio = useTerminals((s) => s.setSplitRatio);
  const onDrag = (clientX: number, clientY: number) => {
    const el = ref.current;
    if (!el) return;
    const r = el.getBoundingClientRect();
    const ratio =
      node.dir === "row"
        ? (clientX - r.left) / r.width
        : (clientY - r.top) / r.height;
    setSplitRatio(tab.id, node.id, ratio);
  };
  return (
    <div
      ref={ref}
      className={cn(
        "flex h-full w-full",
        node.dir === "row" ? "flex-row" : "flex-col"
      )}
    >
      <div
        className="relative min-h-0 min-w-0 overflow-hidden"
        style={{ flex: `0 0 ${node.ratio * 100}%` }}
      >
        <PaneView tab={tab} node={node.a} visible={visible} />
      </div>
      <SplitHandle dir={node.dir} onDrag={onDrag} />
      <div className="relative min-h-0 min-w-0 flex-1 overflow-hidden">
        <PaneView tab={tab} node={node.b} visible={visible} />
      </div>
    </div>
  );
}

function SplitHandle({
  dir,
  onDrag,
}: {
  dir: SplitDir;
  onDrag: (clientX: number, clientY: number) => void;
}) {
  const start = (e: React.MouseEvent) => {
    e.preventDefault();
    const move = (ev: MouseEvent) => onDrag(ev.clientX, ev.clientY);
    const up = () => {
      window.removeEventListener("mousemove", move);
      window.removeEventListener("mouseup", up);
      document.body.style.userSelect = "";
      document.body.style.cursor = "";
    };
    document.body.style.userSelect = "none";
    document.body.style.cursor = dir === "row" ? "col-resize" : "row-resize";
    window.addEventListener("mousemove", move);
    window.addEventListener("mouseup", up);
  };
  return (
    <div
      onMouseDown={start}
      className={cn(
        "z-20 shrink-0 bg-border transition-colors hover:bg-accent/60",
        dir === "row" ? "w-1 cursor-col-resize" : "h-1 cursor-row-resize"
      )}
    />
  );
}

function TerminalLeaf({
  tab,
  paneId,
  visible,
}: {
  tab: TerminalTab;
  paneId: string;
  visible: boolean;
}) {
  const hostRef = useRef<HTMLDivElement>(null);
  const [state, setState] = useState<PaneState>({
    status: "opening",
    error: null,
    exitCode: null,
  });

  const fontSize = useSettings((s) => s.terminalFontSize);
  const fontFamily = useSettings((s) => s.terminalFontFamily);
  const themeKey = useSettings((s) => s.terminalTheme);

  const setPaneActive = useTerminals((s) => s.setPaneActive);
  const closePane = useTerminals((s) => s.closePane);
  const isActive = useTerminals(
    (s) => s.tabs.find((t) => t.id === tab.id)?.activePaneId === paneId
  );
  const isSplit = useTerminals((s) => {
    const t = s.tabs.find((t) => t.id === tab.id);
    return t ? t.root.kind === "split" : false;
  });

  // Acquire + attach the registry instance; detach (NEVER dispose — the store
  // owns disposal) on unmount. Capturing `host` targets detach at this leaf so a
  // re-parent doesn't yank the node out of the leaf that adopted it.
  useEffect(() => {
    const host = hostRef.current;
    const initialCommand =
      tab.initialPaneId === paneId ? tab.initialCommand : undefined;
    const entry = acquirePane(paneId, { sessionId: tab.sessionId, initialCommand });
    if (host) entry.attach(host);
    const unsub = entry.subscribe(setState);
    return () => {
      if (host) entry.detach(host);
      unsub();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [paneId]);

  // Refit whenever the host resizes (splits, ratio drags, zoom, dock show).
  useEffect(() => {
    const el = hostRef.current;
    if (!el) return;
    const ro = new ResizeObserver(() => getPane(paneId)?.refit());
    ro.observe(el);
    return () => ro.disconnect();
  }, [paneId]);

  // Fit + focus the active pane on becoming visible.
  useEffect(() => {
    if (!visible) return;
    const raf = requestAnimationFrame(() => {
      const entry = getPane(paneId);
      if (!entry) return;
      entry.refit();
      if (isActive) {
        entry.term.focus();
        noteTerminalFocus(paneId);
      }
    });
    return () => cancelAnimationFrame(raf);
  }, [visible, isActive, paneId]);

  // Live-apply font/theme without recreating the term.
  useEffect(() => {
    const entry = getPane(paneId);
    if (!entry) return;
    entry.term.options.fontSize = fontSize;
    entry.term.options.fontFamily = fontFamily;
    entry.term.options.theme = TERMINAL_THEMES[themeKey];
    if (visible) entry.refit();
  }, [fontSize, fontFamily, themeKey, visible, paneId]);

  const theme = TERMINAL_THEMES[themeKey];

  return (
    <div
      className={cn(
        "group absolute inset-0 flex flex-col",
        visible ? "" : "pointer-events-none invisible",
        isSplit && isActive && "z-10 ring-1 ring-inset ring-accent/50"
      )}
      aria-hidden={!visible}
    >
      <div
        ref={hostRef}
        onMouseDown={() => {
          getPane(paneId)?.term.focus();
          noteTerminalFocus(paneId);
          setPaneActive(tab.id, paneId);
        }}
        className="h-full w-full flex-1 overflow-hidden"
        style={{ background: theme.background }}
      />
      {isSplit && (
        <button
          onClick={() => closePane(tab.id, paneId)}
          title="Close pane"
          className="absolute right-1.5 top-1.5 z-20 rounded p-0.5 text-text-dim opacity-0 hover:bg-bg-hover hover:text-danger group-hover:opacity-100"
        >
          <X size={12} />
        </button>
      )}
      {state.status === "opening" && (
        <div className="pointer-events-none absolute left-2 top-2 rounded bg-bg-panel/70 px-1.5 py-0.5 text-[10px] text-text-dim">
          opening…
        </div>
      )}
      {state.error && (
        <div className="border-t border-border bg-danger-soft px-3 py-1 text-xs text-danger">
          {state.error}
        </div>
      )}
    </div>
  );
}

// Backwards-compatible re-export so any old import keeps working.
export const Terminal = TerminalDock;
