import { createPortal } from "react-dom";
import { useEffect, useMemo } from "react";
import {
  Search,
  X,
  Ban,
  Loader2,
  AlertCircle,
  Eye,
  Copy,
  FileText,
  Regex,
  CaseSensitive,
  Play,
  RefreshCw,
  FolderSearch,
} from "lucide-react";
import { useSearch } from "@/stores/searchStore";
import { useConnections } from "@/stores/connectionsStore";
import { useLayout } from "@/stores/layoutStore";
import { useEditor } from "@/stores/editorStore";
import { toast } from "@/stores/toastStore";
import { LOCAL_SESSION } from "@ghostftp/file-ui";
import type { SearchHit, SessionId } from "@/lib/types";
import { fmtSize } from "@/lib/format";
import { cn } from "@/lib/cn";

/** Parent directory of a POSIX/Windows path (its own root at the top). */
function parentDir(p: string): string {
  if (p === "/") return "/";
  const driveRoot = p.match(/^[a-zA-Z]:[\\/]/)?.[0];
  const i = Math.max(p.lastIndexOf("/"), p.lastIndexOf("\\"));
  if (driveRoot && i === 2) return driveRoot;
  if (i <= 0) return p.startsWith("/") ? "/" : p;
  return p.slice(0, i);
}

const STRATEGY_LABEL: Record<string, string> = {
  generic: "walk",
  shell: "server-side grep",
  objectFlat: "bucket listing",
};

/** Mount point: renders the overlay only while a search is open. */
export function FleetSearchHost() {
  const open = useSearch((s) => s.open);
  if (!open) return null;
  return <FleetSearch />;
}

function FleetSearch() {
  const sessionId = useSearch((s) => s.sessionId);
  const root = useSearch((s) => s.root);
  const pattern = useSearch((s) => s.pattern);
  const kind = useSearch((s) => s.kind);
  const regex = useSearch((s) => s.regex);
  const caseSensitive = useSearch((s) => s.caseSensitive);
  const include = useSearch((s) => s.include);
  const exclude = useSearch((s) => s.exclude);
  const contentRemote = useSearch((s) => s.contentRemote);
  const state = useSearch((s) => s.state);
  const strategy = useSearch((s) => s.strategy);
  const filesScanned = useSearch((s) => s.filesScanned);
  const hitCount = useSearch((s) => s.hitCount);
  const truncated = useSearch((s) => s.truncated);
  const note = useSearch((s) => s.note);
  const error = useSearch((s) => s.error);
  const hits = useSearch((s) => s.hits);
  const setPattern = useSearch((s) => s.setPattern);
  const setKind = useSearch((s) => s.setKind);
  const setRegex = useSearch((s) => s.setRegex);
  const setCaseSensitive = useSearch((s) => s.setCaseSensitive);
  const setInclude = useSearch((s) => s.setInclude);
  const setExclude = useSearch((s) => s.setExclude);
  const setContentRemote = useSearch((s) => s.setContentRemote);
  const run = useSearch((s) => s.run);
  const cancel = useSearch((s) => s.cancel);
  const close = useSearch((s) => s.close);

  const sessions = useConnections((s) => s.sessions);
  const profiles = useConnections((s) => s.profiles);
  const requestReveal = useLayout((s) => s.requestReveal);

  const nameFor = useMemo(() => {
    const nameOfProfile = (profileId: string) =>
      profiles.find((p) => p.id === profileId)?.name ?? "connection";
    const byId = new Map<SessionId, string>([[LOCAL_SESSION, "Local filesystem"]]);
    for (const s of sessions) byId.set(s.sessionId, nameOfProfile(s.profileId));
    return (id: SessionId) => byId.get(id) ?? id;
  }, [sessions, profiles]);

  // Esc closes (matches every other overlay).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [close]);

  const searching = state === "searching";
  const isContent = kind === "content";
  const canRun = pattern.trim() !== "" && !searching;

  const reveal = (path: string, isDir: boolean) => {
    requestReveal(sessionId, isDir ? path : parentDir(path));
    close();
  };
  const openFile = (path: string) => {
    useEditor.getState().startEditing(sessionId, path).catch((e) => toast.error("Couldn't open file", String(e)));
    close();
  };
  const copyPath = (path: string) => {
    navigator.clipboard.writeText(path);
    toast.info("Path copied", path);
  };

  const done = state === "done";
  const empty = done && hits.length === 0;

  return createPortal(
    <div className="anim-modal fixed inset-0 z-modal flex flex-col bg-bg">
      {/* Header */}
      <div className="flex shrink-0 items-center gap-2 border-b border-border bg-bg-panel px-3 py-2">
        <Search size={15} className="shrink-0 text-accent" />
        <span className="shrink-0 text-sm font-semibold">Fleet Search</span>
        <span
          className="shrink-0 max-w-[20rem] truncate rounded bg-bg-subtle px-2 py-1 font-mono text-xs"
          title={`${nameFor(sessionId)}:${root}`}
        >
          <span className="text-text-dim">{nameFor(sessionId)}</span>
          <span className="text-text-dim">:</span>
          {root}
        </span>

        {/* Name / Content toggle */}
        <div className="flex shrink-0 overflow-hidden rounded-md border border-border text-[11px]">
          <button
            onClick={() => setKind("name")}
            disabled={searching}
            className={cn(
              "flex items-center gap-1 px-2 py-1 disabled:opacity-40",
              !isContent ? "bg-accent/10 text-accent" : "text-text-muted hover:bg-bg-hover"
            )}
            title="Match file/folder names"
          >
            <FolderSearch size={12} /> Name
          </button>
          <button
            onClick={() => setKind("content")}
            disabled={searching}
            className={cn(
              "flex items-center gap-1 border-l border-border px-2 py-1 disabled:opacity-40",
              isContent ? "bg-accent/10 text-accent" : "text-text-muted hover:bg-bg-hover"
            )}
            title="Grep inside file contents"
          >
            <FileText size={12} /> Content
          </button>
        </div>

        <input
          value={pattern}
          onChange={(e) => setPattern(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && canRun) void run();
          }}
          autoFocus
          spellCheck={false}
          placeholder={isContent ? "grep pattern…" : "name (glob or substring)…"}
          className="min-w-0 flex-1 rounded border border-border bg-bg-subtle px-2 py-1 font-mono text-xs text-text focus:border-accent focus:outline-none"
        />

        {/* Case toggle (both kinds) + regex (content only) */}
        <button
          onClick={() => setCaseSensitive(!caseSensitive)}
          disabled={searching}
          className={cn(
            "flex shrink-0 items-center rounded-md border px-1.5 py-1 disabled:opacity-40",
            caseSensitive
              ? "border-accent bg-accent/10 text-accent"
              : "border-border text-text-muted hover:bg-bg-hover"
          )}
          title="Case-sensitive"
        >
          <CaseSensitive size={13} />
        </button>
        {isContent && (
          <button
            onClick={() => setRegex(!regex)}
            disabled={searching}
            className={cn(
              "flex shrink-0 items-center rounded-md border px-1.5 py-1 disabled:opacity-40",
              regex
                ? "border-accent bg-accent/10 text-accent"
                : "border-border text-text-muted hover:bg-bg-hover"
            )}
            title="Treat the pattern as a regular expression"
          >
            <Regex size={13} />
          </button>
        )}

        {searching ? (
          <button
            onClick={() => cancel()}
            className="flex shrink-0 items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text"
            title="Stop the search"
          >
            <Ban size={12} /> Cancel
          </button>
        ) : (
          <button
            onClick={() => run()}
            disabled={!canRun}
            className="flex shrink-0 items-center gap-1 rounded-md border border-border px-2 py-1 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-40"
            title="Run the search (Enter)"
          >
            {done ? <RefreshCw size={12} /> : <Play size={12} />}
            {done ? "Rerun" : "Search"}
          </button>
        )}
        <button
          onClick={() => close()}
          className="shrink-0 rounded-md p-1 text-text-muted hover:bg-bg-hover hover:text-danger"
          title="Close (Esc)"
        >
          <X size={16} />
        </button>
      </div>

      {/* Filters row: include / exclude globs + content-remote opt-in */}
      <div className="flex shrink-0 items-center gap-2 border-b border-border bg-bg-subtle px-3 py-1.5 text-xs">
        <span className="shrink-0 text-[11px] text-text-dim">Files:</span>
        <input
          value={include}
          onChange={(e) => setInclude(e.target.value)}
          disabled={searching}
          spellCheck={false}
          placeholder="include e.g. *.rs, *.toml"
          className="min-w-0 flex-1 rounded border border-border bg-bg px-2 py-1 font-mono text-[11px] text-text focus:border-accent focus:outline-none disabled:opacity-40"
        />
        <input
          value={exclude}
          onChange={(e) => setExclude(e.target.value)}
          disabled={searching}
          spellCheck={false}
          placeholder="exclude e.g. *.min.js"
          className="min-w-0 flex-1 rounded border border-border bg-bg px-2 py-1 font-mono text-[11px] text-text focus:border-accent focus:outline-none disabled:opacity-40"
        />
        {isContent && (
          <button
            onClick={() => setContentRemote(!contentRemote)}
            disabled={searching}
            className={cn(
              "shrink-0 rounded-md border px-2 py-1 text-[11px] disabled:opacity-40",
              contentRemote
                ? "border-accent bg-accent/10 text-accent"
                : "border-border text-text-muted hover:bg-bg-hover"
            )}
            title="Allow content search to DOWNLOAD files on backends with no server-side grep (object stores, FTP, WebDAV, cloud)"
          >
            Allow remote download
          </button>
        )}
      </div>

      {/* Progress / summary bar */}
      {(searching || done) && (
        <div className="flex shrink-0 items-center gap-2 border-b border-border bg-bg-panel px-3 py-1.5 text-xs text-text-muted">
          {searching ? (
            <Loader2 size={13} className="animate-spin text-accent" />
          ) : (
            <Search size={13} className="text-text-dim" />
          )}
          <span className="tabular-nums">
            {hitCount.toLocaleString()} {hitCount === 1 ? "match" : "matches"}
            {searching ? "…" : ""}
          </span>
          {!searching && filesScanned > 0 && (
            <span className="text-text-dim">· {filesScanned.toLocaleString()} files read</span>
          )}
          <span className="rounded-full bg-bg-subtle px-2 py-0.5 text-[10px] text-text-dim">
            {STRATEGY_LABEL[strategy] ?? strategy}
          </span>
          {truncated && (
            <span className="text-amber-400">· capped at {hits.length.toLocaleString()} — narrow the query</span>
          )}
          {note && <span className="text-text-dim">· {note}</span>}
        </div>
      )}

      {/* Body */}
      {error ? (
        <div className="flex flex-1 items-center justify-center">
          <div className="flex max-w-md flex-col items-center gap-2 text-center">
            <AlertCircle size={22} className="text-danger" />
            <div className="text-sm font-medium">Search failed</div>
            <div className="text-xs text-text-dim">{error}</div>
          </div>
        </div>
      ) : state === null ? (
        <div className="flex flex-1 items-center justify-center">
          <div className="flex max-w-sm flex-col items-center gap-2 text-center text-text-dim">
            <Search size={26} />
            <div className="text-sm">
              Type a {isContent ? "grep pattern" : "name"} and press Enter.
            </div>
          </div>
        </div>
      ) : empty ? (
        <div className="flex flex-1 items-center justify-center">
          <div className="text-sm text-text-dim">No matches.</div>
        </div>
      ) : isContent ? (
        <ContentHits hits={hits} onReveal={reveal} onOpen={openFile} onCopyPath={copyPath} />
      ) : (
        <NameHits hits={hits} onReveal={reveal} onCopyPath={copyPath} />
      )}
    </div>,
    document.body
  );
}

const MAX_ROWS = 2000;

function NameHits({
  hits,
  onReveal,
  onCopyPath,
}: {
  hits: SearchHit[];
  onReveal: (path: string, isDir: boolean) => void;
  onCopyPath: (path: string) => void;
}) {
  const shown = hits.slice(0, MAX_ROWS);
  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      {shown.map((h) => (
        <div
          key={h.path}
          className="group flex items-center gap-2 border-b border-border-subtle px-3 py-1.5 text-xs hover:bg-bg-hover"
          title={h.path}
        >
          {h.isDir ? (
            <FolderSearch size={13} className="shrink-0 text-sky-400" />
          ) : (
            <FileText size={13} className="shrink-0 text-text-dim" />
          )}
          <span className="min-w-0 flex-1 truncate font-mono">
            {h.relative}
            {h.isDir && "/"}
          </span>
          {!h.isDir && (
            <span className="w-20 shrink-0 text-right tabular-nums text-text-dim">
              {fmtSize(h.size)}
            </span>
          )}
          <div className="flex w-12 shrink-0 items-center justify-end gap-1 opacity-0 group-hover:opacity-100">
            <button
              onClick={() => onReveal(h.path, h.isDir)}
              className="rounded p-0.5 text-text-muted hover:bg-bg-hover hover:text-text"
              title="Reveal in browser"
            >
              <Eye size={12} />
            </button>
            <button
              onClick={() => onCopyPath(h.path)}
              className="rounded p-0.5 text-text-muted hover:bg-bg-hover hover:text-text"
              title="Copy path"
            >
              <Copy size={12} />
            </button>
          </div>
        </div>
      ))}
      {hits.length > MAX_ROWS && (
        <div className="px-3 py-2 text-center text-[10px] text-text-dim">
          +{(hits.length - MAX_ROWS).toLocaleString()} more not shown
        </div>
      )}
    </div>
  );
}

function ContentHits({
  hits,
  onReveal,
  onOpen,
  onCopyPath,
}: {
  hits: SearchHit[];
  onReveal: (path: string, isDir: boolean) => void;
  onOpen: (path: string) => void;
  onCopyPath: (path: string) => void;
}) {
  // Group consecutive hits by file, preserving discovery order.
  const groups = useMemo(() => {
    const shown = hits.slice(0, MAX_ROWS);
    const out: { relative: string; path: string; lines: SearchHit[] }[] = [];
    const index = new Map<string, number>();
    for (const h of shown) {
      let i = index.get(h.path);
      if (i === undefined) {
        i = out.length;
        index.set(h.path, i);
        out.push({ relative: h.relative, path: h.path, lines: [] });
      }
      out[i].lines.push(h);
    }
    return out;
  }, [hits]);

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      {groups.map((g) => (
        <div key={g.path} className="border-b border-border-subtle">
          <div className="group sticky top-0 z-sticky flex items-center gap-2 bg-bg-subtle px-3 py-1 text-xs">
            <FileText size={12} className="shrink-0 text-accent" />
            <span className="min-w-0 flex-1 truncate font-mono font-medium" title={g.path}>
              {g.relative}
            </span>
            <span className="shrink-0 text-[10px] text-text-dim">{g.lines.length}</span>
            <div className="flex shrink-0 items-center gap-1 opacity-0 group-hover:opacity-100">
              <button
                onClick={() => onOpen(g.path)}
                className="rounded p-0.5 text-text-muted hover:bg-bg-hover hover:text-text"
                title="Open in editor"
              >
                <FileText size={12} />
              </button>
              <button
                onClick={() => onReveal(g.path, false)}
                className="rounded p-0.5 text-text-muted hover:bg-bg-hover hover:text-text"
                title="Reveal in browser"
              >
                <Eye size={12} />
              </button>
              <button
                onClick={() => onCopyPath(g.path)}
                className="rounded p-0.5 text-text-muted hover:bg-bg-hover hover:text-text"
                title="Copy path"
              >
                <Copy size={12} />
              </button>
            </div>
          </div>
          {g.lines.map((h, i) => (
            <button
              key={`${h.line}-${i}`}
              onClick={() => onOpen(h.path)}
              className="flex w-full items-baseline gap-3 px-3 py-0.5 text-left text-xs hover:bg-bg-hover"
              title="Open in editor"
            >
              <span className="w-12 shrink-0 text-right font-mono tabular-nums text-text-dim">
                {h.line}
              </span>
              <span className="min-w-0 flex-1 truncate whitespace-pre font-mono text-text-muted">
                {h.preview}
              </span>
            </button>
          ))}
        </div>
      ))}
      {hits.length > MAX_ROWS && (
        <div className="px-3 py-2 text-center text-[10px] text-text-dim">
          +{(hits.length - MAX_ROWS).toLocaleString()} more lines not shown
        </div>
      )}
    </div>
  );
}
