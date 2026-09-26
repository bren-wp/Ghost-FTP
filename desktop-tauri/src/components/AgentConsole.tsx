import { useEffect, useMemo, useRef, useState } from "react";
import {
  X,
  Trash2,
  TerminalSquare,
  FileText,
  Download,
  Upload,
  Search,
  Ban,
  CircleAlert,
  CheckCircle2,
  Loader2,
  Radio,
  ChevronRight,
  ChevronDown,
  ChevronsDownUp,
  Filter,
  FileDown,
} from "lucide-react";
import { useBridge } from "@/stores/bridgeStore";
import { useConnections } from "@/stores/connectionsStore";
import { useLayout } from "@/stores/layoutStore";
import { ipc } from "@/lib/ipc";
import { toast } from "@/stores/toastStore";
import { cn } from "@/lib/cn";
import { relTime } from "@/lib/format";
import type { AgentConsoleEntry } from "@/lib/types";

// Build a plain-text transcript of the console for export — readable and
// paste-friendly for a bug report or feature request.
function buildExportText(
  entries: AgentConsoleEntry[],
  meta: Map<string, { name: string; color?: string }>
): string {
  const out: string[] = [
    "Ghost FTP Agent Console export",
    `Generated: ${new Date().toISOString()}`,
    `Entries: ${entries.length}`,
    "",
  ];
  for (const e of entries) {
    const who = meta.get(e.sessionId)?.name || e.sessionName || e.sessionId;
    const stat =
      e.status === "running"
        ? "running"
        : e.kind === "denied"
          ? "denied"
          : e.kind === "error" || e.ok === false
            ? "FAILED"
            : "ok";
    out.push(
      `[${new Date(e.at).toISOString()}] ${e.kind.toUpperCase()} · ${who} · ${stat}`
    );
    out.push(`  ${e.command}`);
    const body = e.output.replace(/\s+$/, "");
    if (body.length > 0) {
      out.push("  ----- output -----");
      for (const line of body.split("\n")) out.push(`  ${line}`);
      out.push("  ------------------");
    }
    out.push("");
  }
  return out.join("\n");
}

// A live, read-only view of everything the agent does through the bridge,
// docked at the bottom of the window so it stays visible while you work. Exec
// output streams in real time; every other op shows as a feed line. Entries
// collapse so a chatty agent doesn't bury the feed, and a per-session filter
// isolates one server when several are bridged at once.
export function AgentConsoleDock() {
  const entries = useBridge((s) => s.console);
  const clear = useBridge((s) => s.clearConsole);
  const setConsoleOpen = useLayout((s) => s.setConsoleOpen);
  const sessions = useConnections((s) => s.sessions);
  const profiles = useConnections((s) => s.profiles);

  const [filter, setFilter] = useState<string | "all">("all");
  // Per-entry open overrides; running entries default open, done default closed.
  const [overrides, setOverrides] = useState<Record<string, boolean>>({});

  const scrollRef = useRef<HTMLDivElement>(null);
  const stick = useRef(true);

  // Label + tag color for a session id, resolved through the connection list.
  const sessionMeta = useMemo(() => {
    const map = new Map<string, { name: string; color?: string }>();
    for (const s of sessions) {
      const p = profiles.find((x) => x.id === s.profileId);
      if (p) map.set(s.sessionId, { name: p.name, color: p.color });
    }
    return map;
  }, [sessions, profiles]);

  // Sessions that actually appear in the feed, for the filter menu.
  const feedSessions = useMemo(() => {
    const seen = new Map<string, string>();
    for (const e of entries) {
      if (!seen.has(e.sessionId)) {
        seen.set(
          e.sessionId,
          sessionMeta.get(e.sessionId)?.name || e.sessionName || e.sessionId
        );
      }
    }
    return [...seen.entries()].map(([id, name]) => ({ id, name }));
  }, [entries, sessionMeta]);

  const shown = filter === "all" ? entries : entries.filter((e) => e.sessionId === filter);
  const running = entries.filter((e) => e.status === "running").length;

  // Keep the view pinned to the latest line unless the user scrolled up.
  useEffect(() => {
    const el = scrollRef.current;
    if (el && stick.current) el.scrollTop = el.scrollHeight;
  }, [shown]);

  const onScroll = () => {
    const el = scrollRef.current;
    if (!el) return;
    stick.current = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
  };

  const collapseAll = () => {
    const next: Record<string, boolean> = {};
    for (const e of entries) next[e.id] = false;
    setOverrides(next);
  };

  // Export what's currently shown (so the session filter doubles as a scope
  // selector) to a timestamped text file in the user's Downloads folder.
  const exportLogs = async () => {
    if (shown.length === 0) return;
    const d = new Date();
    const p = (n: number) => String(n).padStart(2, "0");
    const name = `ghostftp-agent-console-${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(
      d.getDate()
    )}_${p(d.getHours())}-${p(d.getMinutes())}-${p(d.getSeconds())}.txt`;
    try {
      const path = await ipc.exportAgentLog(
        buildExportText(shown, sessionMeta),
        name
      );
      toast.success("Console exported", path);
    } catch (e) {
      toast.error("Couldn't export console", String(e));
    }
  };

  // Clear the active filter if its session disappears from the feed.
  useEffect(() => {
    if (filter !== "all" && !feedSessions.some((s) => s.id === filter)) {
      setFilter("all");
    }
  }, [filter, feedSessions]);

  return (
    <div className="flex h-full flex-col bg-bg-panel">
      <div className="flex h-8 shrink-0 items-center gap-2 border-b border-border bg-bg-subtle px-2.5">
        <Radio size={13} className="text-accent" />
        <span className="text-[12px] font-semibold tracking-tight">Agent console</span>
        {running > 0 && (
          <span className="flex items-center gap-1 rounded-full bg-accent/15 px-1.5 py-0.5 text-[10px] font-medium text-accent">
            <Loader2 size={10} className="animate-spin" /> {running} running
          </span>
        )}
        <div className="flex-1" />
        {feedSessions.length > 1 && (
          <SessionFilter
            value={filter}
            sessions={feedSessions}
            meta={sessionMeta}
            onChange={setFilter}
          />
        )}
        <ToolButton onClick={collapseAll} disabled={shown.length === 0} title="Collapse all output">
          <ChevronsDownUp size={12} />
        </ToolButton>
        <ToolButton
          onClick={exportLogs}
          disabled={shown.length === 0}
          title="Export console to a file (Downloads)"
        >
          <FileDown size={12} />
        </ToolButton>
        <ToolButton onClick={clear} disabled={entries.length === 0} title="Clear console">
          <Trash2 size={12} />
        </ToolButton>
        <ToolButton onClick={() => setConsoleOpen(false)} title="Close console">
          <X size={14} />
        </ToolButton>
      </div>

      <div
        ref={scrollRef}
        onScroll={onScroll}
        className="min-w-0 flex-1 space-y-1.5 overflow-y-auto px-3 py-2.5"
      >
        {shown.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 text-center text-xs text-text-dim">
            <TerminalSquare size={20} className="opacity-50" />
            <div className="max-w-sm">
              {entries.length === 0
                ? "Nothing yet. When the agent runs commands or transfers files through the bridge, they stream here live."
                : "No activity for this server. Switch the filter to see other sessions."}
            </div>
          </div>
        ) : (
          shown.map((e) => (
            <ConsoleEntry
              key={e.id}
              entry={e}
              tag={sessionMeta.get(e.sessionId)}
              open={overrides[e.id] ?? e.status === "running"}
              onToggle={() =>
                setOverrides((o) => ({
                  ...o,
                  [e.id]: !(o[e.id] ?? e.status === "running"),
                }))
              }
            />
          ))
        )}
      </div>
    </div>
  );
}

function SessionFilter({
  value,
  sessions,
  meta,
  onChange,
}: {
  value: string | "all";
  sessions: { id: string; name: string }[];
  meta: Map<string, { name: string; color?: string }>;
  onChange: (v: string | "all") => void;
}) {
  const [open, setOpen] = useState(false);
  const wrapRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    const onDown = (e: MouseEvent) => {
      if (!wrapRef.current?.contains(e.target as Node)) setOpen(false);
    };
    window.addEventListener("keydown", onKey);
    window.addEventListener("mousedown", onDown);
    return () => {
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("mousedown", onDown);
    };
  }, [open]);

  const label = value === "all" ? "All servers" : meta.get(value)?.name || value;

  return (
    <div className="relative" ref={wrapRef}>
      <button
        onClick={() => setOpen((v) => !v)}
        className={cn(
          "flex h-6 items-center gap-1 rounded-md border border-border px-1.5 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text",
          open && "bg-bg-hover text-text"
        )}
      >
        <Filter size={11} />
        <span className="max-w-[9rem] truncate">{label}</span>
        <ChevronDown size={11} className="text-text-dim" />
      </button>
      {open && (
        <div className="anim-modal absolute right-0 top-7 z-menu w-52 overflow-hidden rounded-md border border-border bg-bg-panel shadow-elev-3">
          <FilterRow active={value === "all"} onClick={() => { onChange("all"); setOpen(false); }}>
            All servers
          </FilterRow>
          {sessions.map((s) => (
            <FilterRow
              key={s.id}
              active={value === s.id}
              color={meta.get(s.id)?.color}
              onClick={() => { onChange(s.id); setOpen(false); }}
            >
              {s.name}
            </FilterRow>
          ))}
        </div>
      )}
    </div>
  );
}

function FilterRow({
  active,
  color,
  onClick,
  children,
}: {
  active: boolean;
  color?: string;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "flex w-full items-center gap-2 px-3 py-1.5 text-left text-[12px] hover:bg-bg-hover",
        active ? "text-text" : "text-text-muted"
      )}
    >
      {color !== undefined ? (
        <span
          className="h-2 w-2 shrink-0 rounded-full"
          style={{ background: color || "rgb(var(--accent))" }}
        />
      ) : (
        <span className="h-2 w-2 shrink-0" />
      )}
      <span className="min-w-0 flex-1 truncate">{children}</span>
      {active && <CheckCircle2 size={12} className="shrink-0 text-accent" />}
    </button>
  );
}

function ToolButton({
  children,
  onClick,
  disabled,
  title,
}: {
  children: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
  title: string;
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      title={title}
      className="flex h-6 w-6 items-center justify-center rounded-md text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-40 disabled:hover:bg-transparent"
    >
      {children}
    </button>
  );
}

function KindIcon({ entry }: { entry: AgentConsoleEntry }) {
  if (entry.status === "running")
    return <Loader2 size={12} className="shrink-0 animate-spin text-accent" />;
  if (entry.kind === "denied")
    return <Ban size={12} className="shrink-0 text-text-dim" />;
  if (entry.kind === "error" || entry.ok === false)
    return <CircleAlert size={12} className="shrink-0 text-danger" />;
  switch (entry.kind) {
    case "exec":
      return <TerminalSquare size={12} className="shrink-0 text-accent" />;
    case "read":
      return <FileText size={12} className="shrink-0 text-text-muted" />;
    case "search":
      return <Search size={12} className="shrink-0 text-text-muted" />;
    case "download":
      return <Download size={12} className="shrink-0 text-text-muted" />;
    case "upload":
      return <Upload size={12} className="shrink-0 text-text-muted" />;
    default:
      return <CheckCircle2 size={12} className="shrink-0 text-success" />;
  }
}

function ConsoleEntry({
  entry,
  tag,
  open,
  onToggle,
}: {
  entry: AgentConsoleEntry;
  tag?: { name: string; color?: string };
  open: boolean;
  onToggle: () => void;
}) {
  const failed = entry.kind === "error" || entry.ok === false;
  const hasOutput = entry.output.length > 0;
  const lineCount = hasOutput ? entry.output.split("\n").length : 0;

  return (
    <div className="overflow-hidden rounded-md border border-border-subtle bg-bg-subtle/40">
      <button
        onClick={hasOutput ? onToggle : undefined}
        className={cn(
          "flex w-full items-center gap-2 px-2.5 py-1.5 text-left",
          hasOutput && "hover:bg-bg-hover/60"
        )}
      >
        {hasOutput ? (
          open ? (
            <ChevronDown size={12} className="shrink-0 text-text-dim" />
          ) : (
            <ChevronRight size={12} className="shrink-0 text-text-dim" />
          )
        ) : (
          <span className="w-3 shrink-0" />
        )}
        <KindIcon entry={entry} />
        <span
          className={cn(
            "min-w-0 flex-1 truncate font-mono text-[12px]",
            failed ? "text-danger" : "text-text"
          )}
        >
          {entry.command}
        </span>
        {hasOutput && !open && (
          <span className="shrink-0 text-[10px] text-text-dim">{lineCount} lines</span>
        )}
        {tag && (
          <span className="flex shrink-0 items-center gap-1 text-[10px] text-text-dim">
            <span
              className="h-1.5 w-1.5 rounded-full"
              style={{ background: tag.color || "rgb(var(--accent))" }}
            />
            <span className="max-w-[8rem] truncate">{tag.name}</span>
          </span>
        )}
        <span className="shrink-0 text-[10px] text-text-dim">{relTime(entry.at)}</span>
      </button>
      {hasOutput && open && (
        <pre className="selectable max-h-72 max-w-full overflow-auto whitespace-pre-wrap [overflow-wrap:anywhere] border-t border-border-subtle bg-bg-panel px-2.5 py-1.5 font-mono text-[11px] leading-relaxed text-text-muted">
          {entry.output}
        </pre>
      )}
    </div>
  );
}
