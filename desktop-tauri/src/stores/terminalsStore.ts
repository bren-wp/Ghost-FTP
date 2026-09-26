import { create } from "zustand";
import type { SessionId } from "@/lib/types";
import { disposePane } from "@/lib/terminalRegistry";

// A terminal tab owns a LAYOUT TREE of panes (Plan 11 Phase 2). A fresh tab is
// a single leaf; splitting a leaf replaces it with a split node holding the
// original leaf plus a new one. Every leaf's `paneId` keys a live xterm in the
// terminal registry — the tree only describes geometry, never the instance.
// Multiple tabs may belong to the same SSH session; each pane opens its own PTY
// channel on that one pooled connection (no re-auth).

/** One split direction: `row` = side-by-side (vertical divider, Cmd+D),
 *  `col` = stacked (horizontal divider, Cmd+Shift+D). */
export type SplitDir = "row" | "col";

export type PaneNode =
  | { kind: "leaf"; paneId: string }
  | {
      kind: "split";
      id: string;
      dir: SplitDir;
      /** Fraction (0..1) of the space the first child gets. */
      ratio: number;
      a: PaneNode;
      b: PaneNode;
    };

export interface TerminalTab {
  id: string; // client-side id, used as React key
  sessionId: SessionId;
  title: string; // editable; defaults to `Shell N`
  /** The pane layout for this tab. */
  root: PaneNode;
  /** The focused leaf within this tab (target for splits / close-pane). */
  activePaneId: string;
  /** When set, only this pane is shown full-size (others stay mounted). */
  zoomedPaneId: string | null;
  /** The one pane seeded with `initialCommand` (the tab's original leaf). */
  initialPaneId: string;
  /** Sent to that pane's PTY once, right after it opens (e.g. a `cd`). */
  initialCommand?: string;
}

interface TerminalsState {
  tabs: TerminalTab[];
  activeId: string | null;

  openTab: (sessionId: SessionId, initialCommand?: string) => TerminalTab;
  closeTab: (id: string) => void;
  setActive: (id: string) => void;
  renameTab: (id: string, title: string) => void;
  /** Drop all tabs for a session (called on disconnect). */
  dropSessionTabs: (sessionId: SessionId) => void;

  // ---- Split panes ----
  /** Split the given tab's active pane; the new pane becomes active. */
  splitActivePane: (tabId: string, dir: SplitDir) => void;
  /** Close one pane; collapses its split, or closes the tab if it was last. */
  closePane: (tabId: string, paneId: string) => void;
  /** Focus a pane within a tab. */
  setPaneActive: (tabId: string, paneId: string) => void;
  /** Toggle "zoom" on the active pane (full-size, siblings hidden). */
  toggleZoom: (tabId: string) => void;
  /** Drag-resize: set a split node's ratio. */
  setSplitRatio: (tabId: string, splitId: string, ratio: number) => void;
}

function genId() {
  return crypto.randomUUID();
}

// ---- Pure tree helpers ----

function leafIds(node: PaneNode): string[] {
  return node.kind === "leaf"
    ? [node.paneId]
    : [...leafIds(node.a), ...leafIds(node.b)];
}

function firstLeaf(node: PaneNode): string {
  return node.kind === "leaf" ? node.paneId : firstLeaf(node.a);
}

function replaceLeaf(node: PaneNode, paneId: string, repl: PaneNode): PaneNode {
  if (node.kind === "leaf") return node.paneId === paneId ? repl : node;
  const a = replaceLeaf(node.a, paneId, repl);
  const b = replaceLeaf(node.b, paneId, repl);
  return a === node.a && b === node.b ? node : { ...node, a, b };
}

/** Remove a leaf, collapsing its parent split to the surviving sibling. Returns
 *  null only when the removed leaf WAS the whole tree. */
function removeLeaf(node: PaneNode, paneId: string): PaneNode | null {
  if (node.kind === "leaf") return node.paneId === paneId ? null : node;
  const a = removeLeaf(node.a, paneId);
  const b = removeLeaf(node.b, paneId);
  if (a === null) return b;
  if (b === null) return a;
  return a === node.a && b === node.b ? node : { ...node, a, b };
}

function setRatio(node: PaneNode, splitId: string, ratio: number): PaneNode {
  if (node.kind === "leaf") return node;
  if (node.id === splitId) return { ...node, ratio };
  const a = setRatio(node.a, splitId, ratio);
  const b = setRatio(node.b, splitId, ratio);
  return a === node.a && b === node.b ? node : { ...node, a, b };
}

export const useTerminals = create<TerminalsState>((set, get) => ({
  tabs: [],
  activeId: null,

  openTab: (sessionId, initialCommand) => {
    const sessionCount = get().tabs.filter((t) => t.sessionId === sessionId).length;
    const paneId = genId();
    const tab: TerminalTab = {
      id: genId(),
      sessionId,
      title: `Shell ${sessionCount + 1}`,
      root: { kind: "leaf", paneId },
      activePaneId: paneId,
      zoomedPaneId: null,
      initialPaneId: paneId,
      initialCommand,
    };
    set((s) => ({ tabs: [...s.tabs, tab], activeId: tab.id }));
    return tab;
  },

  closeTab: (id) => {
    const s = get();
    const tab = s.tabs.find((t) => t.id === id);
    if (!tab) return;
    // Store-driven disposal: tear down every pane the tab owned.
    for (const paneId of leafIds(tab.root)) disposePane(paneId);
    const tabs = s.tabs.filter((t) => t.id !== id);
    let activeId = s.activeId;
    if (s.activeId === id) {
      const sibling =
        tabs.filter((t) => t.sessionId === tab.sessionId).pop() ??
        tabs[tabs.length - 1] ??
        null;
      activeId = sibling ? sibling.id : null;
    }
    set({ tabs, activeId });
  },

  setActive: (id) => set({ activeId: id }),

  renameTab: (id, title) =>
    set((s) => ({
      tabs: s.tabs.map((t) => (t.id === id ? { ...t, title } : t)),
    })),

  dropSessionTabs: (sessionId) => {
    const s = get();
    for (const t of s.tabs) {
      if (t.sessionId === sessionId)
        for (const paneId of leafIds(t.root)) disposePane(paneId);
    }
    const tabs = s.tabs.filter((t) => t.sessionId !== sessionId);
    const activeStillThere = tabs.some((t) => t.id === s.activeId);
    set({ tabs, activeId: activeStillThere ? s.activeId : tabs[0]?.id ?? null });
  },

  splitActivePane: (tabId, dir) =>
    set((s) => ({
      tabs: s.tabs.map((t) => {
        if (t.id !== tabId) return t;
        const newPane = genId();
        const split: PaneNode = {
          kind: "split",
          id: genId(),
          dir,
          ratio: 0.5,
          a: { kind: "leaf", paneId: t.activePaneId },
          b: { kind: "leaf", paneId: newPane },
        };
        return {
          ...t,
          root: replaceLeaf(t.root, t.activePaneId, split),
          activePaneId: newPane,
          // A split and a zoom are mutually exclusive — splitting un-zooms.
          zoomedPaneId: null,
        };
      }),
    })),

  closePane: (tabId, paneId) => {
    const tab = get().tabs.find((t) => t.id === tabId);
    if (!tab) return;
    // Last pane in the tab → close the whole tab (handles disposal + active).
    if (leafIds(tab.root).length <= 1) {
      get().closeTab(tabId);
      return;
    }
    disposePane(paneId);
    set((s) => ({
      tabs: s.tabs.map((t) => {
        if (t.id !== tabId) return t;
        const root = removeLeaf(t.root, paneId) ?? t.root;
        const remaining = leafIds(root);
        const activePaneId = remaining.includes(t.activePaneId)
          ? t.activePaneId
          : firstLeaf(root);
        return {
          ...t,
          root,
          activePaneId,
          zoomedPaneId:
            t.zoomedPaneId && remaining.includes(t.zoomedPaneId)
              ? t.zoomedPaneId
              : null,
        };
      }),
    }));
  },

  setPaneActive: (tabId, paneId) =>
    set((s) => ({
      tabs: s.tabs.map((t) =>
        t.id === tabId ? { ...t, activePaneId: paneId } : t
      ),
    })),

  toggleZoom: (tabId) =>
    set((s) => ({
      tabs: s.tabs.map((t) =>
        t.id === tabId
          ? { ...t, zoomedPaneId: t.zoomedPaneId ? null : t.activePaneId }
          : t
      ),
    })),

  setSplitRatio: (tabId, splitId, ratio) =>
    set((s) => ({
      tabs: s.tabs.map((t) =>
        t.id === tabId
          ? { ...t, root: setRatio(t.root, splitId, clamp(ratio, 0.1, 0.9)) }
          : t
      ),
    })),
}));

function clamp(v: number, lo: number, hi: number) {
  return Math.max(lo, Math.min(hi, v));
}

// Exposed for tests / the split renderer.
export const _tree = { leafIds, firstLeaf, replaceLeaf, removeLeaf, setRatio };
