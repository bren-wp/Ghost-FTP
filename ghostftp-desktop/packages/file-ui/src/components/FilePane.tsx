import { useEffect, useMemo, useState, useCallback, useRef } from "react";
import {
  ArrowUp,
  ArrowLeft,
  ArrowRight,
  RefreshCw,
  FolderPlus,
  Edit3,
  Trash2,
  Copy,
  CopyPlus,
  ShieldCheck,
  ServerOff,
  ExternalLink,
  FileArchive,
  TerminalSquare,
  PieChart,
  GitCompareArrows,
  CopyX,
  Info,
  Search,
  X,
  ChevronUp,
  ChevronDown,
  Pencil,
  Inbox,
  SearchX,
  List,
  Table2,
  AlignJustify,
  LayoutGrid,
  Upload,
  Download,
  Image as ImageIcon,
} from "lucide-react";
import type {
  Capabilities,
  DirEntry,
  PaneDensity,
  PaneViewMode,
  SessionId,
  SortField,
} from "../types";
import { LOCAL_SESSION } from "../types";
import { useFileUi } from "../context";
import { usePathHistory } from "../hooks/usePathHistory";
import { cn } from "../lib/cn";
import { fmtSize, fmtMtime, formatMode } from "../lib/format";
import { isImage, type FileIconSpec } from "../lib/fileIcons";
import { materialIconUrl } from "../lib/materialIcons";
import {
  DEFAULT_FILE_BROWSER_KEYBINDINGS,
  comboFromEvent,
  type FileBrowserAction,
} from "../lib/keys";
import { ContextMenu, type MenuItem } from "./ContextMenu";
import { PromptModal } from "./PromptModal";
import { ConfirmModal } from "./ConfirmModal";
import { PropertiesModal } from "./PropertiesModal";
import { FileListSkeleton } from "./Skeleton";
import { EmptyState } from "./EmptyState";
import { Thumbnail } from "./Thumbnail";

interface Segment {
  label: string;
  path: string;
}

// A drive-letter (C:\ or C:/) or UNC (\\server\share) path is Windows-style
// regardless of whether the session is local — a Ghost FTP Agent on a Windows host
// serves these too, so the separator must be inferred from the path itself,
// never from the session kind.
function isWindowsPath(path: string): boolean {
  return /^[a-zA-Z]:/.test(path) || path.startsWith("\\\\");
}

// Build clickable breadcrumb segments for the address bar. Windows-style
// paths keep drive/UNC prefix + backslashes, everything else is POSIX "/".
// Relative POSIX paths (e.g. "." or "./public_html" — the SFTP default home
// when the profile has no start dir) must stay relative: turning them
// absolute ("/./...") would point the backend at a nonexistent path.
function parseSegments(path: string): Segment[] {
  if (!path || path === ".") return [{ label: path || "/", path: path || "/" }];
  if (path === "/") return [{ label: "/", path: "/" }];
  const parts = path.split(/[/\\]/).filter(Boolean);
  const segs: Segment[] = [];
  if (isWindowsPath(path)) {
    // UNC: \\server\share is the smallest listable root, so it's one crumb.
    const isUnc = path.startsWith("\\\\");
    let acc = isUnc
      ? "\\\\" + parts.slice(0, 2).join("\\")
      : parts[0] + "\\";
    segs.push({ label: isUnc ? acc : parts[0], path: acc });
    for (let i = isUnc ? 2 : 1; i < parts.length; i++) {
      acc = acc.endsWith("\\") ? acc + parts[i] : acc + "\\" + parts[i];
      segs.push({ label: parts[i], path: acc });
    }
  } else if (!path.startsWith("/")) {
    // Relative POSIX path: no root crumb, segments accumulate relatively.
    let acc = "";
    for (const p of parts) {
      acc = acc ? acc + "/" + p : p;
      segs.push({ label: p, path: acc });
    }
  } else {
    segs.push({ label: "/", path: "/" });
    let acc = "";
    for (const p of parts) {
      acc = acc + "/" + p;
      segs.push({ label: p, path: acc });
    }
  }
  return segs;
}

const DRAG_MIME = "application/x-ghostftp";

interface DragPayload {
  paneId: string;
  entries: DirEntry[];
}

interface Props {
  paneId: string;
  title: string;
  sessionId: SessionId | null;
  path: string;
  onPathChange: (path: string) => void;
  onTransfer?: (entries: DirEntry[]) => void;
  /** Optional toolbar Upload action (server view) — opens a native file picker. */
  onUpload?: (e: React.MouseEvent) => void;
  onDrop?: (entries: DirEntry[]) => void;
  transferLabel?: string;
  /** Bump to force a re-listing of the current directory (e.g. after an upload). */
  reloadToken?: number;
}

/** Extract a human message from any thrown value. Transport backends may reject
 *  with a structured `{ kind, message }` object (Ghost FTP's Tauri commands do); those
 *  would otherwise stringify to "[object Object]" in the error banner. */
function errorText(e: unknown): string {
  if (typeof e === "string") return e;
  if (e && typeof e === "object" && typeof (e as { message?: unknown }).message === "string") {
    return (e as { message: string }).message;
  }
  if (e instanceof Error) return e.message;
  return String(e);
}

type ModalState =
  | { type: "rename"; entry: DirEntry }
  | { type: "mkdir" }
  | { type: "chmod"; entry: DirEntry }
  | { type: "delete"; entries: DirEntry[] }
  | { type: "props"; entry: DirEntry }
  | null;

export function FilePane({
  paneId,
  title,
  sessionId,
  path,
  onPathChange,
  onTransfer,
  onUpload,
  onDrop,
  transferLabel = "Transfer",
  reloadToken,
}: Props) {
  // Everything that used to be a direct Tauri call or a zustand read now comes
  // through the injected adapter + settings. The component is otherwise the
  // same presentational/interaction logic it always was.
  const { fs, settings, iconFor } = useFileUi();
  const {
    showHiddenFiles,
    sortField,
    sortDirection,
    setSortField,
    setSortDirection,
    paneViewMode,
    paneDensity,
    setPaneViewMode,
    setPaneDensity,
    editorLabel,
    remoteImagePreviews,
    setRemoteImagePreviews,
    keyBindings,
  } = settings;

  // Effective in-pane key combos: host overrides layered over the built-in
  // defaults, indexed combo→action so the keydown handler is a single lookup.
  // Memoized on the combo values (not object identity) so a host that rebuilds
  // the settings object each render doesn't thrash the listener.
  const comboToAction = useMemo(() => {
    const merged = { ...DEFAULT_FILE_BROWSER_KEYBINDINGS, ...keyBindings };
    const map: Record<string, FileBrowserAction> = {};
    for (const [action, combo] of Object.entries(merged)) {
      if (combo) map[combo] = action as FileBrowserAction;
    }
    return map;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify({ ...DEFAULT_FILE_BROWSER_KEYBINDINGS, ...keyBindings })]);

  // Re-key the thumbnail loader whenever the remote-preview toggle flips, so
  // already-mounted Thumbnails re-run their effect the moment the user clicks
  // the toolbar toggle (off→on fetches; on→off falls back to icons) instead of
  // waiting for the next navigation. Wrapping `fs.thumbnail` gives a fresh
  // function identity per toggle state — the dep the Thumbnail effect watches.
  const loadThumb = useMemo(
    () =>
      fs.thumbnail
        ? (sid: SessionId, entry: DirEntry, signal?: AbortSignal) =>
            fs.thumbnail!(sid, entry, signal)
        : undefined,
    [fs, remoteImagePreviews]
  );

  const [entries, setEntries] = useState<DirEntry[]>([]);
  const [draftPath, setDraftPath] = useState(path);
  const [editingPath, setEditingPath] = useState(false);
  const [filter, setFilter] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [anchor, setAnchor] = useState<string | null>(null);
  const [isDropTarget, setIsDropTarget] = useState(false);
  const [caps, setCaps] = useState<Capabilities | null>(null);
  // Width tier of THIS pane (not the viewport). Two panes share the window, so
  // a narrow window squeezes each pane; metadata columns yield to the filename
  // before it gets crushed. Driven by a ResizeObserver, see below.
  const [paneTier, setPaneTier] = useState<"wide" | "mid" | "narrow">("wide");
  const [menu, setMenu] = useState<{
    x: number;
    y: number;
    items: MenuItem[];
  } | null>(null);
  const [modal, setModal] = useState<ModalState>(null);
  const [paneFocused, setPaneFocused] = useState(false);
  const dragCounter = useRef(0);
  const containerRef = useRef<HTMLDivElement>(null);
  const typeBufRef = useRef("");
  const typeAtRef = useRef(0);

  const history = usePathHistory(path, onPathChange);

  const scrollToIdx = (n: number) => {
    requestAnimationFrame(() => {
      containerRef.current
        ?.querySelector(`[data-idx="${n}"]`)
        ?.scrollIntoView({ block: "nearest" });
    });
  };

  useEffect(() => {
    setDraftPath(path);
    setEditingPath(false);
    setFilter("");
  }, [path]);

  const load = useCallback(
    async (p: string) => {
      if (!sessionId) {
        setEntries([]);
        return;
      }
      setLoading(true);
      setError(null);
      try {
        const list = await fs.listDirectory(sessionId, p);
        setEntries(list);
        setSelected(new Set());
        setAnchor(null);
      } catch (e) {
        setError(errorText(e));
      } finally {
        setLoading(false);
      }
    },
    [sessionId, fs]
  );

  useEffect(() => {
    load(path);
  }, [sessionId, path, load, reloadToken]);

  // Pull capabilities once per session so chmod / mkdir / etc. can be hidden
  // for backends that don't support them (notably S3).
  useEffect(() => {
    if (!sessionId) {
      setCaps(null);
      return;
    }
    let cancelled = false;
    fs.capabilities(sessionId).then(
      (c) => {
        if (!cancelled) setCaps(c);
      },
      () => {
        if (!cancelled) setCaps(null);
      }
    );
    return () => {
      cancelled = true;
    };
  }, [sessionId, fs]);

  // Apply hidden-file rule + in-pane name filter + sort from settings. Memoized
  // so a big directory isn't re-filtered and re-sorted on every unrelated render
  // (selection, hover, pane-resize); only the inputs below trigger recompute.
  const visible = useMemo(() => {
    const q = filter.trim().toLowerCase();
    return entries
      .filter((e) => showHiddenFiles || !e.name.startsWith("."))
      .filter((e) => q === "" || e.name.toLowerCase().includes(q))
      .slice()
      .sort((a, b) => {
        // Directories always come before files regardless of sort.
        if (a.kind === "directory" && b.kind !== "directory") return -1;
        if (a.kind !== "directory" && b.kind === "directory") return 1;
        let cmp = 0;
        if (sortField === "size") {
          cmp = a.size - b.size;
        } else if (sortField === "modified") {
          cmp = (a.modified ?? 0) - (b.modified ?? 0);
        } else {
          cmp = a.name.localeCompare(b.name);
        }
        return sortDirection === "asc" ? cmp : -cmp;
      });
  }, [entries, showHiddenFiles, filter, sortField, sortDirection]);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const anchorItem = () =>
      anchor ? visible.find((v) => v.path === anchor) ?? null : null;
    const handler = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null;
      const typing =
        !!target &&
        (target.tagName === "INPUT" ||
          target.tagName === "TEXTAREA" ||
          target.isContentEditable);
      if (e.key === "Escape") {
        // Filter first, then selection — least-surprising unwind order. Escape
        // is intentionally not remappable (it's the universal cancel key).
        if (filter) {
          setFilter("");
          if (typing) (target as HTMLInputElement).blur();
        } else {
          setSelected(new Set());
          setAnchor(null);
        }
        return;
      }
      if (typing) return; // don't hijack keys while editing text
      // Only run list nav when the pane/list area is focused, not a toolbar btn.
      if (target && target.closest('button, a, [role="button"]')) return;

      // Remappable in-pane actions (F2/Enter/Delete/Backspace/mod+A/Space/
      // mod+shift+N/F5, all overridable by the host). One combo → one action.
      const action = comboToAction[comboFromEvent(e)];
      if (action) {
        switch (action) {
          case "selectAll":
            e.preventDefault();
            setSelected(new Set(visible.map((x) => x.path)));
            return;
          case "delete": {
            const items = visible.filter((v) => selected.has(v.path));
            if (items.length > 0) setModal({ type: "delete", entries: items });
            return;
          }
          case "rename": {
            const item = anchorItem();
            if (item) {
              e.preventDefault();
              setModal({ type: "rename", entry: item });
            }
            return;
          }
          case "quickInfo": {
            const item = anchorItem();
            if (item) {
              e.preventDefault();
              setModal({ type: "props", entry: item });
            }
            return;
          }
          case "newFolder":
            if (sessionId && caps?.hasDirectories !== false) {
              e.preventDefault();
              setModal({ type: "mkdir" });
            }
            return;
          case "refresh":
            if (sessionId) {
              e.preventDefault();
              load(path);
            }
            return;
          case "open": {
            const item = anchorItem();
            if (item) {
              e.preventDefault();
              onRowActivate(item);
            }
            return;
          }
          case "parentDir":
            e.preventDefault();
            goUp();
            return;
        }
      }

      if (e.ctrlKey || e.metaKey || e.altKey) return; // leave app shortcuts alone

      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        e.preventDefault();
        if (visible.length === 0) return;
        const cur = anchor ? visible.findIndex((v) => v.path === anchor) : -1;
        let next = e.key === "ArrowDown" ? cur + 1 : cur - 1;
        next = Math.max(0, Math.min(visible.length - 1, next));
        const targetItem = visible[next];
        if (targetItem) {
          if (e.shiftKey) setSelected((prev) => new Set(prev).add(targetItem.path));
          else setSelected(new Set([targetItem.path]));
          setAnchor(targetItem.path);
          scrollToIdx(next);
        }
        return;
      }
      // Type-ahead: jump to the first item matching the recently-typed run.
      if (e.key.length === 1) {
        const now = Date.now();
        typeBufRef.current =
          now - typeAtRef.current > 800 ? e.key : typeBufRef.current + e.key;
        typeAtRef.current = now;
        const buf = typeBufRef.current.toLowerCase();
        let idx = visible.findIndex((v) => v.name.toLowerCase().startsWith(buf));
        if (idx < 0) idx = visible.findIndex((v) => v.name.toLowerCase().includes(buf));
        if (idx >= 0) {
          setSelected(new Set([visible[idx].path]));
          setAnchor(visible[idx].path);
          scrollToIdx(idx);
        }
      }
    };
    el.addEventListener("keydown", handler);
    return () => el.removeEventListener("keydown", handler);
  }, [visible, selected, filter, anchor, comboToAction, caps, path, sessionId]);

  // Watch this pane's width and pick a column tier. Content-driven, not
  // viewport-driven: only re-renders when crossing a threshold, not per pixel.
  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const ro = new ResizeObserver((entries) => {
      const w = entries[0].contentRect.width;
      const tier = w < 360 ? "narrow" : w < 480 ? "mid" : "wide";
      setPaneTier((prev) => (prev === tier ? prev : tier));
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const goUp = () => {
    const isWin = isWindowsPath(path);
    const isUnc = path.startsWith("\\\\");
    // Relative POSIX path (e.g. "." / "./public_html" — the SFTP default
    // home): going up must stay relative, and the relative root itself has
    // no parent to go to.
    const isRelative = !isWin && !path.startsWith("/");
    const parts = path.split(/[/\\]/).filter(Boolean);
    const rootParts = isUnc ? 2 : 1; // UNC root is \\server\share
    if (parts.length <= rootParts) {
      onPathChange(isWin || isRelative ? path : "/");
      return;
    }
    parts.pop();
    let next: string;
    if (isUnc) next = "\\\\" + parts.join("\\");
    else if (isWin) next = parts.length === 1 ? parts[0] + "\\" : parts.join("\\");
    else if (isRelative) next = parts.join("/");
    else next = "/" + parts.join("/");
    onPathChange(next || (isRelative ? "." : "/"));
  };

  const joinPath = (parent: string, name: string) => {
    const sep = isWindowsPath(parent) ? "\\" : "/";
    if (parent.endsWith(sep) || parent.endsWith("/")) return `${parent}${name}`;
    return `${parent}${sep}${name}`;
  };

  const onRowClick = (entry: DirEntry, e: React.MouseEvent) => {
    containerRef.current?.focus(); // so arrow / type-ahead nav works after a click
    if (e.shiftKey && anchor) {
      const idxA = visible.findIndex((x) => x.path === anchor);
      const idxB = visible.findIndex((x) => x.path === entry.path);
      if (idxA >= 0 && idxB >= 0) {
        const [lo, hi] = idxA < idxB ? [idxA, idxB] : [idxB, idxA];
        const next = new Set<string>();
        for (let i = lo; i <= hi; i++) next.add(visible[i].path);
        setSelected(next);
      }
    } else if (e.ctrlKey || e.metaKey) {
      setSelected((prev) => {
        const next = new Set(prev);
        if (next.has(entry.path)) next.delete(entry.path);
        else next.add(entry.path);
        return next;
      });
      setAnchor(entry.path);
    } else {
      setSelected(new Set([entry.path]));
      setAnchor(entry.path);
    }
  };

  const onRowActivate = (entry: DirEntry) => {
    if (entry.kind === "directory") {
      onPathChange(entry.path);
      return;
    }
    onTransfer?.([entry]);
  };

  const transferSelection = () => {
    const items = visible.filter((e) => selected.has(e.path));
    if (items.length === 0) return;
    onTransfer?.(items);
  };

  // Keep the shared application toolbar honest: publish only the capabilities
  // of the pane the user is actually working in, so impossible actions are
  // disabled instead of producing dead clicks or confusing no-ops.
  useEffect(() => {
    if (!paneFocused) return;
    const selectedItems = visible.filter((entry) => selected.has(entry.path));
    const activeItem = anchor
      ? visible.find((entry) => entry.path === anchor) ?? selectedItems[0]
      : selectedItems[0];
    window.dispatchEvent(new CustomEvent("ghostftp:pane-action-state", {
      detail: {
        paneId,
        selectedCount: selectedItems.length,
        hasActiveItem: Boolean(activeItem),
        hasSession: Boolean(sessionId),
        canCreateDirectory: Boolean(sessionId && caps?.hasDirectories !== false),
      },
    }));
  }, [paneFocused, paneId, visible, selected, anchor, sessionId, caps?.hasDirectories]);

  // Global toolbar bridge. The reference UI has one toolbar above both panes;
  // commands are routed to the focused pane (or to the explicit local/remote
  // target for Upload/Download). This keeps the screenshot-accurate toolbar
  // fully functional without duplicating file-operation logic.
  useEffect(() => {
    const handler = (event: Event) => {
      const custom = event as CustomEvent<{ action?: string; target?: "local" | "remote" }>;
      const action = custom.detail?.action;
      const target = custom.detail?.target;
      if (!action) return;
      if (target && target !== paneId) return;
      const focused = !!containerRef.current?.contains(document.activeElement);
      if (!target && !focused && paneId !== "local") return;

      const selectedItems = visible.filter((entry) => selected.has(entry.path));
      const activeItem = anchor
        ? visible.find((entry) => entry.path === anchor) ?? selectedItems[0]
        : selectedItems[0];

      switch (action) {
        case "refresh":
          if (sessionId) void load(path);
          break;
        case "upload":
        case "download":
          if (selectedItems.length) onTransfer?.(selectedItems);
          break;
        case "newFolder":
          if (sessionId && caps?.hasDirectories !== false) setModal({ type: "mkdir" });
          break;
        case "delete":
          if (selectedItems.length) setModal({ type: "delete", entries: selectedItems });
          break;
        case "rename":
          if (activeItem) setModal({ type: "rename", entry: activeItem });
          break;
        case "properties":
          if (activeItem) setModal({ type: "props", entry: activeItem });
          break;
      }
    };
    window.addEventListener("ghostftp:toolbar-action", handler as EventListener);
    return () => window.removeEventListener("ghostftp:toolbar-action", handler as EventListener);
  }, [paneId, sessionId, path, visible, selected, anchor, caps, load, onTransfer]);

  // --- Drag & drop ---

  const onRowDragStart = (e: React.DragEvent, entry: DirEntry) => {
    const dragSet = selected.has(entry.path)
      ? visible.filter((x) => selected.has(x.path))
      : [entry];
    if (!selected.has(entry.path)) {
      setSelected(new Set([entry.path]));
      setAnchor(entry.path);
    }
    const payload: DragPayload = { paneId, entries: dragSet };
    e.dataTransfer.setData(DRAG_MIME, JSON.stringify(payload));
    e.dataTransfer.effectAllowed = "copy";
  };

  const isOurDrag = (e: React.DragEvent) =>
    Array.from(e.dataTransfer.types).includes(DRAG_MIME);

  const onPaneDragEnter = (e: React.DragEvent) => {
    if (!isOurDrag(e)) return;
    e.preventDefault();
    dragCounter.current++;
    setIsDropTarget(true);
  };
  const onPaneDragOver = (e: React.DragEvent) => {
    if (!isOurDrag(e)) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "copy";
  };
  const onPaneDragLeave = (e: React.DragEvent) => {
    if (!isOurDrag(e)) return;
    dragCounter.current = Math.max(0, dragCounter.current - 1);
    if (dragCounter.current === 0) setIsDropTarget(false);
  };
  const onPaneDrop = (e: React.DragEvent) => {
    if (!isOurDrag(e)) return;
    e.preventDefault();
    dragCounter.current = 0;
    setIsDropTarget(false);
    const raw = e.dataTransfer.getData(DRAG_MIME);
    if (!raw) return;
    let payload: DragPayload;
    try {
      payload = JSON.parse(raw);
    } catch {
      return;
    }
    if (payload.paneId === paneId) return;
    if (!payload.entries.length) return;
    onDrop?.(payload.entries);
  };

  // --- Context menu ---

  const copyPaths = (items: DirEntry[]) => {
    navigator.clipboard.writeText(items.map((e) => e.path).join("\n"));
  };
  const copyNames = (items: DirEntry[]) => {
    navigator.clipboard.writeText(items.map((e) => e.name).join("\n"));
  };

  const openRowMenu = (e: React.MouseEvent, entry: DirEntry) => {
    e.preventDefault();
    e.stopPropagation();
    if (!selected.has(entry.path)) {
      setSelected(new Set([entry.path]));
      setAnchor(entry.path);
    }
    const selectedItems = selected.has(entry.path)
      ? visible.filter((v) => selected.has(v.path))
      : [entry];
    const single = selectedItems.length === 1 ? selectedItems[0] : null;

    // A remote (non-local) session that exposes a shell, used to gate
    // server-side actions (archive, terminal) that local browsing can't do.
    const isRemote = !!sessionId && sessionId !== LOCAL_SESSION;

    const items: MenuItem[] = [];
    if (single) {
      items.push({
        label: single.kind === "directory" ? "Open" : transferLabel,
        onClick: () => onRowActivate(single),
      });
      // Edit-in-place sits right under the primary action so it's easy to find
      // (it used to be last, below chmod). Remote files only — local files open
      // in their app via the OS, and there's nothing to upload back.
      if (single.kind === "file" && isRemote && fs.editFile) {
        items.push({
          label: editorLabel ? `Edit with ${editorLabel}…` : "Edit file…",
          icon: <ExternalLink size={12} />,
          onClick: () => {
            fs.editFile!(sessionId!, single.path).catch((e) =>
              setError(errorText(e))
            );
          },
        });
      }
      // Server-side zip/tar of a folder: one command on the host, then download
      // the single archive — no per-file walk. Remote dirs only (needs a shell).
      if (
        single.kind === "directory" &&
        isRemote &&
        caps?.hasShell &&
        fs.archive
      ) {
        items.push({
          label: "Download as…",
          icon: <FileArchive size={12} />,
          onClick: () => {},
          children: [
            {
              label: "Compressed .tar.gz",
              onClick: () =>
                fs
                  .archive!(sessionId!, single.path, "tar.gz")
                  .catch((e) => setError(errorText(e))),
            },
            {
              label: "Zip archive (.zip)",
              onClick: () =>
                fs
                  .archive!(sessionId!, single.path, "zip")
                  .catch((e) => setError(errorText(e))),
            },
          ],
        });
      }
      // Open an SSH terminal already cd'd into this folder.
      if (
        single.kind === "directory" &&
        isRemote &&
        caps?.hasShell &&
        fs.openTerminal
      ) {
        items.push({
          label: "Open terminal here",
          icon: <TerminalSquare size={12} />,
          onClick: () =>
            fs
              .openTerminal!(sessionId!, single.path)
              .catch((e) => setError(errorText(e))),
        });
      }
      // Analyze disk usage under this folder (treemap + size breakdown). Works
      // on every backend — no shell needed, the scan falls back to a walk.
      if (single.kind === "directory" && fs.analyzeDiskUsage) {
        items.push({
          label: "Analyze disk usage",
          icon: <PieChart size={12} />,
          onClick: () =>
            sessionId &&
            fs
              .analyzeDiskUsage!(sessionId, single.path)
              .catch((e) => setError(errorText(e))),
        });
      }
      // Compare this folder against another tree (Meld-style side-by-side diff).
      // Works on every backend, including remote↔remote.
      if (single.kind === "directory" && fs.compareDirectory) {
        items.push({
          label: "Compare with…",
          icon: <GitCompareArrows size={12} />,
          onClick: () =>
            sessionId &&
            fs
              .compareDirectory!(sessionId, single.path)
              .catch((e) => setError(errorText(e))),
        });
      }
      // Find duplicate files in this folder (the `name_1.ext` copies a
      // rename-on-conflict upload leaves behind, or content-hash duplicates).
      if (single.kind === "directory" && fs.findDuplicates) {
        items.push({
          label: "Find duplicates",
          icon: <CopyX size={12} />,
          onClick: () =>
            sessionId &&
            fs
              .findDuplicates!(sessionId, single.path)
              .catch((e) => setError(errorText(e))),
        });
      }
      // Search this folder by name or content (Spotlight/grep over the backend).
      // Works on every backend; content grep uses a server-side fast path on SSH.
      if (single.kind === "directory" && fs.searchDirectory) {
        items.push({
          label: "Search here…",
          icon: <Search size={12} />,
          onClick: () =>
            sessionId &&
            fs
              .searchDirectory!(sessionId, single.path)
              .catch((e) => setError(errorText(e))),
        });
      }
      // Duplicate needs a server-side copy (cp over SSH) or local fs copy; object
      // stores / FTP can't, so hide it there.
      if (fs.duplicate && (sessionId === LOCAL_SESSION || caps?.hasShell)) {
        items.push({
          label: "Duplicate",
          icon: <CopyPlus size={12} />,
          onClick: () => doDuplicate(single),
        });
      }
      items.push({
        label: "Rename",
        icon: <Edit3 size={12} />,
        onClick: () => setModal({ type: "rename", entry: single }),
        separatorAfter: true,
      });
    } else {
      items.push({
        label: `${transferLabel} ${selectedItems.length} items`,
        onClick: () => onTransfer?.(selectedItems),
        separatorAfter: true,
      });
    }
    items.push({
      label: `Delete${selectedItems.length > 1 ? ` (${selectedItems.length})` : ""}`,
      icon: <Trash2 size={12} />,
      destructive: true,
      onClick: () => setModal({ type: "delete", entries: selectedItems }),
      separatorAfter: true,
    });
    items.push({
      label: `Copy path${selectedItems.length > 1 ? `s (${selectedItems.length})` : ""}`,
      icon: <Copy size={12} />,
      onClick: () => copyPaths(selectedItems),
    });
    items.push({
      label: `Copy name${selectedItems.length > 1 ? `s (${selectedItems.length})` : ""}`,
      onClick: () => copyNames(selectedItems),
      separatorAfter: !!single,
    });
    if (
      single?.kind === "file" &&
      sessionId !== LOCAL_SESSION &&
      caps?.canChmod !== false
    ) {
      items.push({
        label: "Change permissions (chmod)…",
        icon: <ShieldCheck size={12} />,
        onClick: () => setModal({ type: "chmod", entry: single }),
      });
    }
    if (single) {
      items.push({
        label: "Properties",
        icon: <Info size={12} />,
        onClick: () => setModal({ type: "props", entry: single }),
      });
    }
    setMenu({ x: e.clientX, y: e.clientY, items });
  };

  const openPaneMenu = (e: React.MouseEvent) => {
    if (!sessionId) return;
    e.preventDefault();
    const items: MenuItem[] = [];
    if (caps?.hasDirectories !== false) {
      items.push({
        label: "New folder…",
        icon: <FolderPlus size={12} />,
        onClick: () => setModal({ type: "mkdir" }),
        separatorAfter: true,
      });
    }
    items.push({
      label: "Refresh",
      icon: <RefreshCw size={12} />,
      onClick: () => load(path),
    });
    setMenu({ x: e.clientX, y: e.clientY, items });
  };

  // --- File op handlers ---

  const doRename = async (entry: DirEntry, newName: string) => {
    if (!sessionId) return;
    const parentEnd = Math.max(
      entry.path.lastIndexOf("/"),
      entry.path.lastIndexOf("\\")
    );
    const parent = entry.path.slice(0, parentEnd);
    const to = joinPath(parent, newName);
    try {
      await fs.rename(sessionId, entry.path, to);
      await load(path);
    } catch (e) {
      setError(errorText(e));
    }
  };

  const doDelete = async (items: DirEntry[]) => {
    if (!sessionId) return;
    try {
      for (const it of items) {
        await fs.remove(sessionId, it.path, it.kind === "directory");
      }
      await load(path);
    } catch (e) {
      setError(errorText(e));
    }
  };

  const doDuplicate = async (entry: DirEntry) => {
    if (!sessionId || !fs.duplicate) return;
    try {
      await fs.duplicate(sessionId, entry.path, entry.kind);
      await load(path);
    } catch (e) {
      setError(errorText(e));
    }
  };

  const doMkdir = async (name: string) => {
    if (!sessionId) return;
    try {
      await fs.mkdir(sessionId, joinPath(path, name));
      await load(path);
    } catch (e) {
      setError(errorText(e));
    }
  };

  const doChmod = async (entry: DirEntry, modeText: string) => {
    if (!sessionId) return;
    const mode = parseInt(modeText, 8);
    if (Number.isNaN(mode)) {
      setError(`invalid mode "${modeText}" — use octal like 755`);
      return;
    }
    try {
      await fs.chmod(sessionId, entry.path, mode);
      await load(path);
    } catch (e) {
      setError(errorText(e));
    }
  };

  const selectionCount = selected.size;
  const activeIdx = anchor ? visible.findIndex((v) => v.path === anchor) : -1;
  const activeDescId = activeIdx >= 0 ? `${paneId}-row-${activeIdx}` : undefined;
  const segments = parseSegments(path);
  const hasPerms =
    !!sessionId && caps?.canChmod !== false && visible.some((e) => e.mode != null);
  // Columns yield as the pane narrows so the filename never gets crushed:
  // Modified drops below ~360px, Perms shows only when the pane is wide.
  const showModified = paneTier !== "narrow";
  const showPermsCol = hasPerms && paneTier === "wide";
  const cols = [
    "minmax(0,1fr)",
    showModified ? "7.5rem" : null,
    "5.5rem",
    showPermsCol ? "5.5rem" : null,
  ]
    .filter(Boolean)
    .join(" ");
  const selectedBytes = visible
    .filter((e) => selected.has(e.path))
    .reduce((a, e) => a + (e.kind === "file" ? e.size : 0), 0);

  const onSort = (f: SortField) => {
    if (sortField === f) {
      setSortDirection(sortDirection === "asc" ? "desc" : "asc");
    } else {
      setSortField(f);
      setSortDirection("asc");
    }
  };

  const commitPath = () => {
    onPathChange(draftPath.trim() || (sessionId === LOCAL_SESSION ? path : "/"));
    setEditingPath(false);
  };
  const cancelPathEdit = () => {
    setDraftPath(path);
    setEditingPath(false);
  };

  return (
    <div
      ref={containerRef}
      tabIndex={0}
      role="listbox"
      aria-label={`${title} directory listing`}
      aria-multiselectable="true"
      aria-activedescendant={activeDescId}
      onFocus={() => setPaneFocused(true)}
      onBlur={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
          setPaneFocused(false);
        }
      }}
      onDragEnter={onPaneDragEnter}
      onDragOver={onPaneDragOver}
      onDragLeave={onPaneDragLeave}
      onDrop={onPaneDrop}
      onContextMenu={openPaneMenu}
      className={cn(
        "ghost-file-pane flex h-full min-w-0 flex-1 flex-col bg-bg-panel outline-none transition-colors",
        "focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-accent/40",
        isDropTarget && "ring-2 ring-inset ring-accent/60 bg-accent/5"
      )}
    >
      <div className="ghost-pane-header flex min-w-0 items-center gap-1 border-b border-border bg-bg-subtle px-2 py-1.5">
        <span
          title={title}
          className="min-w-0 truncate text-xs font-semibold uppercase tracking-wider text-text-muted"
        >
          {title}
        </span>
        {selectionCount > 0 && (
          <span className="rounded bg-accent/20 px-1.5 py-0.5 text-[10px] font-medium text-accent">
            {selectionCount} selected
          </span>
        )}
        {isDropTarget && (
          <span className="rounded bg-accent-strong px-1.5 py-0.5 text-[10px] font-medium text-white">
            drop to {transferLabel.toLowerCase()}
          </span>
        )}
        <div className="flex-1" />
        {selectionCount > 0 && onTransfer && (
          <button
            onClick={transferSelection}
            className="flex items-center gap-1 rounded bg-accent-strong px-2 py-0.5 text-[11px] font-medium text-white hover:brightness-110"
            title={`${transferLabel} ${selectionCount} item(s)`}
          >
            {transferLabel === "Upload" ? (
              <Upload size={11} />
            ) : (
              <Download size={11} />
            )}
            {transferLabel} {selectionCount}
          </button>
        )}
        {onUpload && (
          <button
            onClick={onUpload}
            disabled={!sessionId}
            className="flex items-center gap-1 rounded border border-accent/40 bg-accent-soft px-2 py-0.5 text-[11px] font-medium text-accent hover:bg-accent/20 disabled:opacity-40"
            title="Upload files or a folder to this directory"
          >
            <Upload size={11} /> Upload
          </button>
        )}
        {fs.analyzeDiskUsage && (
          <button
            onClick={() =>
              sessionId &&
              fs
                .analyzeDiskUsage!(sessionId, path)
                .catch((e) => setError(errorText(e)))
            }
            disabled={!sessionId}
            className="rounded p-1 text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-40"
            title="Analyze disk usage in this folder"
          >
            <PieChart size={13} />
          </button>
        )}
        {fs.searchDirectory && (
          <button
            onClick={() =>
              sessionId &&
              fs
                .searchDirectory!(sessionId, path)
                .catch((e) => setError(errorText(e)))
            }
            disabled={!sessionId}
            className="rounded p-1 text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-40"
            title="Search this folder by name or content"
          >
            <Search size={13} />
          </button>
        )}
        {fs.findDuplicates && (
          <button
            onClick={() =>
              sessionId &&
              fs
                .findDuplicates!(sessionId, path)
                .catch((e) => setError(errorText(e)))
            }
            disabled={!sessionId}
            className="rounded p-1 text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-40"
            title="Find duplicate files in this folder"
          >
            <CopyX size={13} />
          </button>
        )}
        {caps?.hasDirectories !== false && (
          <button
            onClick={() => setModal({ type: "mkdir" })}
            disabled={!sessionId}
            className="rounded p-1 text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-40"
            title="New folder"
          >
            <FolderPlus size={13} />
          </button>
        )}
        <button
          onClick={history.back}
          disabled={!sessionId || !history.canBack}
          className="rounded p-1 text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-30"
          title="Back"
        >
          <ArrowLeft size={13} />
        </button>
        <button
          onClick={history.forward}
          disabled={!sessionId || !history.canForward}
          className="rounded p-1 text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-30"
          title="Forward"
        >
          <ArrowRight size={13} />
        </button>
        <button
          onClick={goUp}
          disabled={!sessionId}
          className="rounded p-1 text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-40"
          title="Up (Backspace)"
        >
          <ArrowUp size={13} />
        </button>
        <button
          onClick={() => load(path)}
          disabled={!sessionId}
          className="rounded p-1 text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-40"
          title="Refresh"
        >
          <RefreshCw size={13} />
        </button>
      </div>

      <div className="ghost-pane-pathbar flex items-center gap-1.5 border-b border-border px-2 py-1">
        {editingPath ? (
          <input
            autoFocus
            value={draftPath}
            onChange={(e) => setDraftPath(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") commitPath();
              else if (e.key === "Escape") cancelPathEdit();
            }}
            onBlur={commitPath}
            disabled={!sessionId}
            className="min-w-0 flex-1 rounded border border-accent bg-bg-subtle px-2 py-1 font-mono text-xs outline-none disabled:opacity-50"
            placeholder="/"
          />
        ) : (
          <div className="flex min-w-0 flex-1 items-center overflow-x-auto whitespace-nowrap [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
            {segments.map((seg, i) => {
              const last = i === segments.length - 1;
              return (
                <span key={seg.path + i} className="flex items-center">
                  {i > 0 && <span className="px-0.5 text-text-dim">/</span>}
                  <button
                    onClick={() => !last && sessionId && onPathChange(seg.path)}
                    disabled={!sessionId || last}
                    className={cn(
                      "max-w-[12rem] truncate rounded px-1 py-0.5 font-mono text-xs",
                      last
                        ? "font-medium text-text"
                        : "text-text-muted hover:bg-bg-hover hover:text-text"
                    )}
                    title={seg.path}
                  >
                    {seg.label}
                  </button>
                </span>
              );
            })}
          </div>
        )}
        <div className="flex shrink-0 items-center gap-0.5">
          <button
            onClick={() => setPaneViewMode("list")}
            className={cn(
              "rounded p-1 hover:bg-bg-hover",
              paneViewMode === "list"
                ? "text-accent"
                : "text-text-dim hover:text-text"
            )}
            title="List view"
          >
            <List size={13} />
          </button>
          <button
            onClick={() => setPaneViewMode("details")}
            className={cn(
              "rounded p-1 hover:bg-bg-hover",
              paneViewMode === "details"
                ? "text-accent"
                : "text-text-dim hover:text-text"
            )}
            title="Details view"
          >
            <Table2 size={13} />
          </button>
          <button
            onClick={() => setPaneViewMode("grid")}
            className={cn(
              "rounded p-1 hover:bg-bg-hover",
              paneViewMode === "grid"
                ? "text-accent"
                : "text-text-dim hover:text-text"
            )}
            title="Grid view"
          >
            <LayoutGrid size={13} />
          </button>
          <button
            onClick={() =>
              setPaneDensity(paneDensity === "compact" ? "comfortable" : "compact")
            }
            className={cn(
              "rounded p-1 hover:bg-bg-hover",
              paneDensity === "compact"
                ? "text-accent"
                : "text-text-dim hover:text-text"
            )}
            title={
              paneDensity === "compact"
                ? "Compact rows — click for comfortable"
                : "Comfortable rows — click for compact"
            }
          >
            <AlignJustify size={13} />
          </button>
          {sessionId &&
            sessionId !== LOCAL_SESSION &&
            setRemoteImagePreviews && (
              <button
                onClick={() => setRemoteImagePreviews(!remoteImagePreviews)}
                className={cn(
                  "rounded p-1 hover:bg-bg-hover",
                  remoteImagePreviews
                    ? "text-accent"
                    : "text-text-dim hover:text-text"
                )}
                title={
                  remoteImagePreviews
                    ? "Image previews on — click to turn off (fetches images to thumbnail them)"
                    : "Image previews off — click to show remote image thumbnails"
                }
              >
                <ImageIcon size={13} />
              </button>
            )}
        </div>
        {!editingPath && (
          <button
            onClick={() => {
              setDraftPath(path);
              setEditingPath(true);
            }}
            disabled={!sessionId}
            className="shrink-0 rounded p-1 text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-40"
            title="Edit path"
          >
            <Pencil size={12} />
          </button>
        )}
        <div className="relative shrink-0">
          <Search
            size={12}
            className="pointer-events-none absolute left-2 top-1/2 -translate-y-1/2 text-text-dim"
          />
          <input
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            disabled={!sessionId}
            placeholder="Filter"
            className="w-32 rounded border border-border bg-bg-subtle py-1 pl-7 pr-6 text-xs outline-none focus:border-accent disabled:opacity-50"
          />
          {filter && (
            <button
              onClick={() => setFilter("")}
              className="absolute right-1.5 top-1/2 -translate-y-1/2 rounded p-0.5 text-text-dim hover:text-text"
              title="Clear filter"
            >
              <X size={11} />
            </button>
          )}
        </div>
      </div>

      {sessionId && paneViewMode === "details" && (
        <div
          className="ghost-pane-columns grid items-center gap-2 border-b border-border bg-bg-subtle px-3 py-1 text-[10px] font-semibold uppercase tracking-wider text-text-muted"
          style={{ gridTemplateColumns: cols }}
        >
          <SortHeader
            label="Name"
            field="name"
            sortField={sortField}
            sortDirection={sortDirection}
            onSort={onSort}
          />
          {showModified && (
            <SortHeader
              label="Modified"
              field="modified"
              sortField={sortField}
              sortDirection={sortDirection}
              onSort={onSort}
            />
          )}
          <SortHeader
            label="Size"
            field="size"
            align="right"
            sortField={sortField}
            sortDirection={sortDirection}
            onSort={onSort}
          />
          {showPermsCol && <span className="text-right">Perms</span>}
        </div>
      )}

      <div role="presentation" className="ghost-pane-list flex-1 overflow-y-auto">
        {!sessionId ? (
          <EmptyState
            icon={<ServerOff size={20} />}
            title="No connection"
            hint="Pick a server in the left rail to connect."
          />
        ) : error ? (
          <div className="px-3 py-2 text-xs text-danger">
            {error}{" "}
            <button
              onClick={() => setError(null)}
              className="ml-1 underline opacity-60 hover:opacity-100"
            >
              dismiss
            </button>
          </div>
        ) : loading ? (
          <FileListSkeleton />
        ) : visible.length === 0 ? (
          filter ? (
            <EmptyState
              icon={<SearchX size={20} />}
              title="No matches"
              hint={`Nothing here matches “${filter}”.`}
              action={
                <button
                  onClick={() => setFilter("")}
                  className="rounded-md border border-border bg-bg-subtle px-2.5 py-1 text-xs text-text-muted hover:text-text"
                >
                  Clear filter
                </button>
              }
            />
          ) : (
            <EmptyState icon={<Inbox size={20} />} title="Empty folder" />
          )
        ) : paneViewMode === "grid" ? (
          <div className="grid [grid-template-columns:repeat(auto-fill,minmax(108px,1fr))] gap-1 p-2">
            {visible.map((entry, i) => (
              <Row
                key={entry.path}
                paneId={paneId}
                entry={entry}
                icon={iconFor(entry)}
                index={i}
                viewMode={paneViewMode}
                density={paneDensity}
                selected={selected.has(entry.path)}
                cols={cols}
                showModified={showModified}
                showPermsCol={showPermsCol}
                sessionId={sessionId}
                loadThumb={loadThumb}
                onClick={(e) => onRowClick(entry, e)}
                onActivate={() => onRowActivate(entry)}
                onDragStart={(e) => onRowDragStart(e, entry)}
                onContextMenu={(e) => openRowMenu(e, entry)}
              />
            ))}
          </div>
        ) : (
          visible.map((entry, i) => (
            <Row
              key={entry.path}
              paneId={paneId}
              entry={entry}
              icon={iconFor(entry)}
              index={i}
              viewMode={paneViewMode}
              density={paneDensity}
              selected={selected.has(entry.path)}
              cols={cols}
              showModified={showModified}
              showPermsCol={showPermsCol}
              sessionId={sessionId}
              loadThumb={loadThumb}
              onClick={(e) => onRowClick(entry, e)}
              onActivate={() => onRowActivate(entry)}
              onDragStart={(e) => onRowDragStart(e, entry)}
              onContextMenu={(e) => openRowMenu(e, entry)}
            />
          ))
        )}
      </div>

      {sessionId && (
        <div className="flex shrink-0 items-center gap-1.5 border-t border-border bg-bg-subtle px-2.5 py-1 text-[10px] text-text-dim">
          <span>
            {visible.length} item{visible.length === 1 ? "" : "s"}
            {filter && entries.length !== visible.length
              ? ` of ${entries.length}`
              : ""}
          </span>
          {selectionCount > 0 && (
            <span className="text-text-muted">
              · {selectionCount} selected
              {selectedBytes > 0 ? ` · ${fmtSize(selectedBytes)}` : ""}
            </span>
          )}
          <div className="flex-1" />
          <span className="max-w-[55%] truncate font-mono" title={path}>
            {path}
          </span>
        </div>
      )}

      {menu && (
        <ContextMenu
          x={menu.x}
          y={menu.y}
          items={menu.items}
          onClose={() => setMenu(null)}
        />
      )}

      {modal?.type === "rename" && (
        <PromptModal
          title={`Rename ${modal.entry.name}`}
          label="New name"
          initialValue={modal.entry.name}
          okLabel="Rename"
          onClose={() => setModal(null)}
          onSubmit={(v) => doRename(modal.entry, v)}
        />
      )}
      {modal?.type === "mkdir" && (
        <PromptModal
          title="New folder"
          label={`In ${path}`}
          initialValue=""
          okLabel="Create"
          onClose={() => setModal(null)}
          onSubmit={(v) => doMkdir(v)}
        />
      )}
      {modal?.type === "chmod" && (
        <PromptModal
          title={`Permissions for ${modal.entry.name}`}
          label="Octal mode (e.g. 755, 644)"
          initialValue={
            modal.entry.mode
              ? (modal.entry.mode & 0o777).toString(8)
              : "644"
          }
          okLabel="Apply"
          onClose={() => setModal(null)}
          onSubmit={(v) => doChmod(modal.entry, v)}
        />
      )}
      {modal?.type === "props" && (
        <PropertiesModal
          entry={modal.entry}
          sessionId={sessionId}
          onApplied={() => void load(path)}
          onOpenContainingFolder={() => {
            const normalized = modal.entry.path.replace(/[\\/]+$/, "");
            const slash = Math.max(normalized.lastIndexOf("/"), normalized.lastIndexOf("\\"));
            const parent = slash > 0 ? normalized.slice(0, slash) : slash === 0 ? "/" : path;
            onPathChange(parent);
            setModal(null);
          }}
          onClose={() => setModal(null)}
        />
      )}
      {modal?.type === "delete" && (
        <ConfirmModal
          title={
            modal.entries.length === 1
              ? `Delete ${modal.entries[0].name}?`
              : `Delete ${modal.entries.length} items?`
          }
          message={
            modal.entries.length === 1
              ? `Path: ${modal.entries[0].path}${
                  modal.entries[0].kind === "directory"
                    ? "\nThis directory and all its contents will be removed."
                    : ""
                }`
              : modal.entries
                  .slice(0, 5)
                  .map((e) => e.path)
                  .join("\n") +
                (modal.entries.length > 5
                  ? `\n…and ${modal.entries.length - 5} more`
                  : "")
          }
          destructive
          confirmLabel="Delete"
          onClose={() => setModal(null)}
          onConfirm={() => doDelete(modal.entries)}
        />
      )}
    </div>
  );
}

function Row({
  paneId,
  entry,
  icon,
  index,
  viewMode,
  density,
  selected,
  cols,
  showModified,
  showPermsCol,
  sessionId,
  loadThumb,
  onClick,
  onActivate,
  onDragStart,
  onContextMenu,
}: {
  paneId: string;
  entry: DirEntry;
  icon: FileIconSpec;
  index: number;
  viewMode: PaneViewMode;
  density: PaneDensity;
  selected: boolean;
  cols: string;
  showModified: boolean;
  showPermsCol: boolean;
  sessionId: SessionId | null;
  loadThumb?: (
    sessionId: SessionId,
    entry: DirEntry,
    signal?: AbortSignal
  ) => Promise<Blob | null>;
  onClick: (e: React.MouseEvent) => void;
  onActivate: () => void;
  onDragStart: (e: React.DragEvent) => void;
  onContextMenu: (e: React.MouseEvent) => void;
}) {
  const { Icon, className: iconColor } = icon;
  const pad = density === "compact" ? "py-0.5 text-xs" : "py-1 text-sm";
  const sel = selected
    ? "bg-accent/15 hover:bg-accent/20"
    : "hover:bg-bg-hover";
  const title =
    entry.kind === "file"
      ? "Double-click to transfer • Right-click for more"
      : "Double-click to open • Right-click for more";
  // Prefer a Material Icon Theme SVG for known file types; fall back to the
  // lucide glyph for folders, symlinks, and unmatched files.
  const matUrl = entry.kind === "file" ? materialIconUrl(entry.name) : undefined;
  const glyph = (size: number) =>
    matUrl ? (
      <img
        src={matUrl}
        width={size}
        height={size}
        alt=""
        draggable={false}
        className="shrink-0 select-none"
      />
    ) : (
      <Icon size={size} className={cn("shrink-0", iconColor)} />
    );
  const iconEl = glyph(13);
  // Show a real image preview when the adapter can supply bytes. The adapter
  // gates the (expensive, network) remote kind on the `remoteImagePreviews`
  // setting — when it's off it returns null fast and the icon fallback shows,
  // so mounting Thumbnail here costs nothing extra.
  const canThumb = !!loadThumb && !!sessionId && isImage(entry);
  // Small inline thumbnail for the list / details rows (grid uses its own,
  // larger one below). Reuses the same lazy, cached pipeline.
  const rowThumb =
    canThumb && sessionId ? (
      <Thumbnail
        sessionId={sessionId}
        entry={entry}
        load={loadThumb!}
        size={18}
        fallback={iconEl}
        className="shrink-0"
      />
    ) : (
      iconEl
    );

  if (viewMode === "grid") {
    const bigIcon = glyph(30);
    return (
      <div
        data-idx={index}
        role="option"
        id={`${paneId}-row-${index}`}
        aria-selected={selected}
        onClick={onClick}
        onDoubleClick={onActivate}
        onContextMenu={onContextMenu}
        draggable
        onDragStart={onDragStart}
        title={title}
        className={cn(
          "ghost-file-row ghost-file-row-grid flex cursor-default select-none flex-col items-center gap-1.5 rounded-lg border p-2.5 text-center",
          "[content-visibility:auto] [contain-intrinsic-size:auto_92px]",
          selected
            ? "border-accent/40 bg-accent/15"
            : "border-transparent hover:border-border-subtle hover:bg-bg-hover"
        )}
      >
        {canThumb && sessionId ? (
          <Thumbnail
            sessionId={sessionId}
            entry={entry}
            load={loadThumb!}
            size={56}
            fallback={bigIcon}
          />
        ) : (
          bigIcon
        )}
        <span className="line-clamp-2 w-full break-words text-xs leading-tight">
          {entry.name}
        </span>
      </div>
    );
  }

  if (viewMode === "list") {
    return (
      <div
        data-idx={index}
        role="option"
        id={`${paneId}-row-${index}`}
        aria-selected={selected}
        onClick={onClick}
        onDoubleClick={onActivate}
        onContextMenu={onContextMenu}
        draggable
        onDragStart={onDragStart}
        title={title}
        className={cn(
          "ghost-file-row ghost-file-row-list flex cursor-default select-none items-center gap-2 border-b border-border-subtle px-3",
          "[content-visibility:auto] [contain-intrinsic-size:auto_28px]",
          pad,
          sel
        )}
      >
        {rowThumb}
        <span className="min-w-0 flex-1 truncate">{entry.name}</span>
        <span className="shrink-0 text-xs tabular-nums text-text-dim">
          {entry.kind === "file" ? fmtSize(entry.size) : ""}
        </span>
      </div>
    );
  }

  return (
    <div
      data-idx={index}
      role="option"
      id={`${paneId}-row-${index}`}
      aria-selected={selected}
      onClick={onClick}
      onDoubleClick={onActivate}
      onContextMenu={onContextMenu}
      draggable
      onDragStart={onDragStart}
      title={title}
      style={{ gridTemplateColumns: cols }}
      className={cn(
        "ghost-file-row ghost-file-row-details grid cursor-default select-none items-center gap-2 border-b border-border-subtle px-3",
        "[content-visibility:auto] [contain-intrinsic-size:auto_28px]",
        pad,
        sel
      )}
    >
      <span className="flex min-w-0 items-center gap-2">
        {rowThumb}
        <span className="truncate">{entry.name}</span>
      </span>
      {showModified && (
        <span className="truncate text-xs text-text-dim">
          {fmtMtime(entry.modified)}
        </span>
      )}
      <span className="text-right text-xs tabular-nums text-text-dim">
        {entry.kind === "file" ? fmtSize(entry.size) : ""}
      </span>
      {showPermsCol && (
        <span
          className="text-right font-mono text-[10px] text-text-dim"
          title={entry.mode != null ? (entry.mode & 0o777).toString(8) : ""}
        >
          {formatMode(entry.mode)}
        </span>
      )}
    </div>
  );
}

function SortHeader({
  label,
  field,
  align = "left",
  sortField,
  sortDirection,
  onSort,
}: {
  label: string;
  field: SortField;
  align?: "left" | "right";
  sortField: SortField;
  sortDirection: "asc" | "desc";
  onSort: (f: SortField) => void;
}) {
  const active = sortField === field;
  return (
    <button
      onClick={() => onSort(field)}
      className={cn(
        "flex items-center gap-1 uppercase tracking-wider hover:text-text",
        align === "right" ? "justify-end" : "justify-start text-left",
        active ? "text-text" : "text-text-muted"
      )}
    >
      {label}
      {active &&
        (sortDirection === "asc" ? (
          <ChevronUp size={11} />
        ) : (
          <ChevronDown size={11} />
        ))}
    </button>
  );
}
