import { create } from "zustand";
import { ipc, onDiskScanEvent } from "@/lib/ipc";
import { toast } from "./toastStore";
import { toastError } from "@/lib/errors";
import type {
  DuNode,
  ScanSnapshot,
  ScanState,
  ScanStrategy,
  DiskScanProgress,
  SessionId,
} from "@/lib/types";

// How the treemap colours its cells.
export type ColorMode = "type" | "depth";

interface DiskScanStoreState {
  /** Whether the Disk Usage explorer overlay is open. */
  open: boolean;
  scanId: string | null;
  sessionId: SessionId | null;
  /** The directory the scan was rooted at. */
  root: string;

  state: ScanState | null;
  strategy: ScanStrategy;
  dirsScanned: number;
  filesFound: number;
  totalBytes: number;
  error: string | null;
  /** Why the fast path fell back to the walk, when it did. */
  note: string | null;
  /** Aggregated size tree (present once the scan is done). */
  tree: DuNode | null;

  /** Drill-down path: segment names from the tree root to the shown node. */
  crumbs: string[];
  colorMode: ColorMode;

  /** Detach the event listener for the current scan. */
  unlisten: (() => void) | null;

  /** Open the explorer and start a scan of `path` on `sessionId`. */
  openFor: (sessionId: SessionId, path: string) => Promise<void>;
  /** Re-run the scan on the same root. */
  rescan: () => Promise<void>;
  /** Ask the backend to stop the running scan. */
  cancel: () => Promise<void>;
  /** Close the overlay and forget the backend scan. */
  close: () => void;

  /** Delete a node on the backend, then prune it from the tree in place
   *  (subtracting its size from every ancestor) so the map updates without a
   *  full rescan. */
  removeNode: (node: DuNode) => Promise<void>;

  /** Drill into a directory node (by walking to it from the tree root). */
  drillTo: (node: DuNode) => void;
  /** Jump the breadcrumb to depth `k` (−1 = the root). */
  navigateTo: (k: number) => void;

  setColorMode: (m: ColorMode) => void;
}

/** Copy a terminal snapshot's fields into the store. */
function fromSnapshot(s: ScanSnapshot) {
  return {
    state: s.state,
    strategy: s.strategy,
    dirsScanned: s.dirsScanned,
    filesFound: s.filesFound,
    totalBytes: s.totalBytes,
    error: s.error ?? null,
    note: s.note ?? null,
    tree: s.tree ?? null,
  };
}

/** Rebuild `node` without the descendant at `targetPath`, subtracting its size
 *  from every ancestor along the way. */
function pruneNode(node: DuNode, targetPath: string): DuNode {
  if (!node.children) return node;
  let removed = 0;
  const children: DuNode[] = [];
  for (const c of node.children) {
    if (c.path === targetPath) {
      removed += c.size;
      continue;
    }
    if (
      targetPath.startsWith(c.path + "/") ||
      targetPath.startsWith(c.path + "\\")
    ) {
      const before = c.size;
      const pruned = pruneNode(c, targetPath);
      removed += before - pruned.size;
      children.push(pruned);
    } else {
      children.push(c);
    }
  }
  return { ...node, size: node.size - removed, children };
}

/** Resolve the currently-shown node by walking `crumbs` from the tree root. */
export function resolveNode(tree: DuNode | null, crumbs: string[]): DuNode | null {
  if (!tree) return null;
  let n = tree;
  for (const seg of crumbs) {
    const next = n.children?.find((c) => c.name === seg);
    if (!next) break;
    n = next;
  }
  return n;
}

export const useDiskScan = create<DiskScanStoreState>((set, get) => ({
  open: false,
  scanId: null,
  sessionId: null,
  root: "",
  state: null,
  strategy: "generic",
  dirsScanned: 0,
  filesFound: 0,
  totalBytes: 0,
  error: null,
  note: null,
  tree: null,
  crumbs: [],
  colorMode: "type",
  unlisten: null,

  openFor: async (sessionId, path) => {
    const prev = get();
    // Tear down any previous scan first.
    if (prev.scanId) ipc.diskScanForget(prev.scanId).catch(() => {});
    prev.unlisten?.();

    const unlisten = await onDiskScanEvent((kind, payload) => {
      // Only react to events for the scan we're currently showing.
      if (!payload || payload.id !== get().scanId) return;
      if (kind === "progress") {
        const p = payload as DiskScanProgress;
        set({
          dirsScanned: p.dirsScanned,
          filesFound: p.filesFound,
          totalBytes: p.totalBytes,
          strategy: p.strategy,
        });
      } else {
        set(fromSnapshot(payload as ScanSnapshot));
      }
    });

    set({
      open: true,
      scanId: null,
      sessionId,
      root: path,
      state: "scanning",
      strategy: "generic",
      dirsScanned: 0,
      filesFound: 0,
      totalBytes: 0,
      error: null,
      note: null,
      tree: null,
      crumbs: [],
      unlisten,
    });

    try {
      const scanId = await ipc.diskScanStart(sessionId, path);
      set({ scanId });
      // Reconcile: the scan may have finished before `scanId` was set, so a
      // terminal event could have been ignored above.
      const snap = await ipc.diskScanTree(scanId);
      if (snap.state !== "scanning") set(fromSnapshot(snap));
      else
        set({
          dirsScanned: snap.dirsScanned,
          filesFound: snap.filesFound,
          totalBytes: snap.totalBytes,
          strategy: snap.strategy,
        });
    } catch (e) {
      set({ state: "error", error: String(e) });
      toast.error("Couldn't start disk usage scan", String(e));
    }
  },

  rescan: async () => {
    const { sessionId, root } = get();
    if (sessionId != null) await get().openFor(sessionId, root);
  },

  cancel: async () => {
    const { scanId } = get();
    if (scanId) await ipc.diskScanCancel(scanId).catch(() => {});
  },

  close: () => {
    const { scanId, unlisten } = get();
    if (scanId) ipc.diskScanForget(scanId).catch(() => {});
    unlisten?.();
    set({
      open: false,
      scanId: null,
      sessionId: null,
      root: "",
      state: null,
      tree: null,
      error: null,
      note: null,
      crumbs: [],
      unlisten: null,
    });
  },

  removeNode: async (node) => {
    const { sessionId, tree, root } = get();
    if (!sessionId || !tree) return;
    // Never let the scan root itself be deleted from here.
    if (node.path === tree.path || node.path === root) return;
    try {
      await ipc.deletePath(sessionId, node.path, node.kind === "directory");
    } catch (e) {
      // `deletePath` rejects with a structured {kind, message} error (Plan 12
      // Phase 3) — toast keyed off the kind.
      toastError(e, "Delete failed");
      return;
    }
    set({ tree: pruneNode(tree, node.path) });
    toast.success("Deleted", node.name);
  },

  drillTo: (node) => {
    const { tree } = get();
    if (!tree || node.kind !== "directory") return;
    const rootPath = tree.path;
    const rel = node.path.startsWith(rootPath)
      ? node.path.slice(rootPath.length)
      : node.path;
    const crumbs = rel.split(/[/\\]/).filter(Boolean);
    set({ crumbs });
  },

  navigateTo: (k) => {
    set((s) => ({ crumbs: k < 0 ? [] : s.crumbs.slice(0, k + 1) }));
  },

  setColorMode: (m) => set({ colorMode: m }),
}));
