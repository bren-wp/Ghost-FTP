import {
  LOCAL_SESSION,
  isImage,
  imageMime,
  type DirEntry,
  type FileSystemAdapter,
  type SessionId,
} from "@ghostftp/file-ui";
import { ipc } from "./ipc";
import { useEditor } from "@/stores/editorStore";
import { useTerminals } from "@/stores/terminalsStore";
import { useLayout } from "@/stores/layoutStore";
import { useDiskScan } from "@/stores/diskScanStore";
import { useDiff } from "@/stores/diffStore";
import { useDedupe } from "@/stores/dedupeStore";
import { useSearch } from "@/stores/searchStore";
import { useSettings } from "@/stores/settingsStore";

// POSIX single-quote a path so a `cd` survives spaces / shell metacharacters.
function shQuote(p: string): string {
  return `'${p.replace(/'/g, `'\\''`)}'`;
}

// ---- Remote image previews (Plan 13 Phase 1) ----------------------------------
// Raster formats the Rust `image` decoder handles without system libs. SVG/AVIF
// are intentionally excluded from *remote* previews (SVG stays a local-only
// passthrough); a non-match falls back to the type icon.
const REMOTE_PREVIEW_EXTS = new Set([
  "png", "jpg", "jpeg", "gif", "webp", "bmp", "ico", "tif", "tiff",
]);
// Mirror of the backend's file-size guard — skip before dispatching a request.
const MAX_REMOTE_PREVIEW_BYTES = 25 * 1024 * 1024;
// Concurrent remote preview fetches per connection. The Thumbnail's viewport
// observer already bounds requests to on-screen rows; this queue keeps a whole
// screenful scrolling in at once from firing dozens of IPC calls simultaneously,
// and — crucially — lets a row that scrolls back out cancel *before* dispatch.
const PER_SESSION_LIMIT = 4;

interface Gate {
  active: number;
  queue: Array<() => void>;
}
const previewGates = new Map<SessionId, Gate>();

/** Acquire a per-session slot, resolving to a release fn. Rejects if `signal`
 *  aborts while still queued — so the request is never dispatched. */
function acquirePreviewSlot(
  sessionId: SessionId,
  signal?: AbortSignal
): Promise<() => void> {
  let gate = previewGates.get(sessionId);
  if (!gate) {
    gate = { active: 0, queue: [] };
    previewGates.set(sessionId, gate);
  }
  const g = gate;
  return new Promise((resolve, reject) => {
    const start = () => {
      g.active++;
      let released = false;
      resolve(() => {
        if (released) return;
        released = true;
        g.active--;
        g.queue.shift()?.();
      });
    };
    if (signal?.aborted) {
      reject(new DOMException("aborted", "AbortError"));
      return;
    }
    if (g.active < PER_SESSION_LIMIT) {
      start();
      return;
    }
    const runQueued = () => {
      signal?.removeEventListener("abort", onAbort);
      start();
    };
    const onAbort = () => {
      const i = g.queue.indexOf(runQueued);
      if (i >= 0) g.queue.splice(i, 1);
      reject(new DOMException("aborted", "AbortError"));
    };
    g.queue.push(runQueued);
    signal?.addEventListener("abort", onAbort, { once: true });
  });
}

/** Fetch a remote thumbnail through the backend cache/decode pipeline, gated by
 *  the setting, the extension/size guards, the viewport signal, and the
 *  per-connection concurrency limiter. Any miss → null → the pane shows an icon. */
async function remoteThumbnail(
  sessionId: SessionId,
  entry: DirEntry,
  signal?: AbortSignal
): Promise<Blob | null> {
  if (useSettings.getState().remoteImagePreviews !== "on") return null;
  const ext = entry.name.split(".").pop()?.toLowerCase() ?? "";
  if (!REMOTE_PREVIEW_EXTS.has(ext)) return null;
  if (entry.size > MAX_REMOTE_PREVIEW_BYTES) return null;
  if (signal?.aborted) return null;

  let release: (() => void) | null = null;
  try {
    release = await acquirePreviewSlot(sessionId, signal);
    if (signal?.aborted) return null;
    // Change token → cache key: prefer the backend's ETag, else size+mtime.
    const sig = entry.etag ?? `${entry.size}:${entry.modified ?? 0}`;
    const b64 = await ipc.previewThumbnail(sessionId, entry.path, entry.size, sig);
    const bytes = Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));
    return new Blob([bytes], {
      type: imageMime(entry.name) ?? "application/octet-stream",
    });
  } catch {
    return null;
  } finally {
    release?.();
  }
}

// The concrete @ghostftp/file-ui adapter for Ghost FTP: it maps the package's
// transport-agnostic operations onto Ghost FTP's Tauri command surface. This is the
// ONE place the open-source file UI meets Ghost FTP's Rust backend — swap this object
// and the same components would drive any other backend (HTTP, mock, …).
export const tauriFileSystem: FileSystemAdapter = {
  listDirectory: (sessionId, path) => ipc.listDirectory(sessionId, path),
  capabilities: (sessionId) => ipc.capabilities(sessionId),
  rename: (sessionId, from, to) => ipc.renamePath(sessionId, from, to),
  remove: (sessionId, path, recursive) =>
    ipc.deletePath(sessionId, path, recursive),
  mkdir: (sessionId, path) => ipc.createDirectory(sessionId, path),
  chmod: (sessionId, path, mode) => ipc.chmodPath(sessionId, path, mode),
  chmodRecursive: (sessionId, path, mode) => ipc.chmodPathRecursive(sessionId, path, mode),
  checksum: (sessionId, path) => ipc.checksumPath(sessionId, path),
  // Edit-in-place runs through the editor store, which installs the
  // save/error listeners and tracks the live-edit pill in the status bar.
  editFile: (sessionId, path) =>
    useEditor.getState().startEditing(sessionId, path),

  // Copy a file/folder next to the original. The Rust side resolves a free
  // name and does the copy server-side (cp over SSH) or on the local disk.
  duplicate: (sessionId, path) => ipc.duplicatePath(sessionId, path),

  // Server-side archive then download. Returns once the transfer is queued; the
  // transfers panel tracks the actual byte download, and the backend removes the
  // temp archive on the server when it finishes.
  archive: async (sessionId, path, format) => {
    await ipc.startArchiveDownload(sessionId, path, format);
  },

  // Reveal the terminal dock and open a fresh shell already cd'd into `path`.
  openTerminal: async (sessionId, path) => {
    useLayout.getState().setTerminalOpen(true);
    useTerminals.getState().openTab(sessionId, `cd ${shQuote(path)}\n`);
  },

  // Open the Disk Usage explorer and kick off a scan of `path`. The store owns
  // the scan lifecycle (progress events, cancel, drill-down); this just triggers it.
  analyzeDiskUsage: async (sessionId, path) => {
    await useDiskScan.getState().openFor(sessionId, path);
  },

  // Open the Directory Diff view with this folder as side A. The user picks side
  // B (a connection + path) in the overlay, then runs the comparison; the diff
  // store owns that lifecycle.
  compareDirectory: async (sessionId, path) => {
    useDiff.getState().openFor(sessionId, path);
  },

  // Open the Fleet Search panel rooted at this folder. The user types a name or
  // content query and runs it; the search store owns the streaming lifecycle.
  searchDirectory: async (sessionId, path) => {
    useSearch.getState().openFor(sessionId, path);
  },

  // Open the Find Duplicates panel rooted at this folder. The user scans
  // (name or content-hash mode), reviews the groups, and deletes the checked
  // copies; the dedupe store owns that lifecycle.
  findDuplicates: async (sessionId, path) => {
    useDedupe.getState().openFor(sessionId, path);
  },

  // Image previews for the grid/list views. Local files read straight off disk
  // (`read_file_preview`, always on). Remote files go through the budgeted,
  // cached `preview_thumbnail` pipeline — opt-in via the `remoteImagePreviews`
  // setting, viewport- and concurrency-gated so scrolling never fires unbounded
  // requests (Plan 13 Phase 1). Any failure → null → the pane shows the icon.
  thumbnail: async (sessionId, entry, signal) => {
    if (!isImage(entry)) return null;
    if (sessionId === LOCAL_SESSION) {
      try {
        const b64 = await ipc.readFilePreview(sessionId, entry.path);
        const bytes = Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));
        const type = imageMime(entry.name) ?? "application/octet-stream";
        return new Blob([bytes], { type });
      } catch {
        return null;
      }
    }
    return remoteThumbnail(sessionId, entry, signal);
  },
};
