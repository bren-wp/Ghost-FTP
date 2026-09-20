import { useCallback, useEffect, useId, useMemo, useRef, useState } from "react";
import {
  Download,
  FileText,
  Server,
  Terminal as TerminalIcon,
  ShieldOff,
  ShieldCheck,
  Cloud,
  Globe,
  Loader2,
  CheckCircle2,
  AlertTriangle,
  RefreshCw,
} from "lucide-react";
import { ipc } from "@/lib/ipc";
import { useConnections } from "@/stores/connectionsStore";
import { useDialog } from "@/hooks/useDialog";
import type {
  ImporterKind,
  ImporterPaths,
  ProfilePreview,
  Protocol,
} from "@/lib/types";
import { cn } from "@/lib/cn";

interface Props {
  onClose: () => void;
}

// Tabbed import dialog. Each tab auto-loads the importer's default-path
// listing on first activation; the user picks which entries to actually
// import. Selection state is per-tab so switching tabs doesn't lose work.
export function ImportDialog({ onClose }: Props) {
  const [tab, setTab] = useState<ImporterKind>("openssh");
  const [paths, setPaths] = useState<ImporterPaths | null>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const titleId = useId();
  useDialog(panelRef, { onClose });

  useEffect(() => {
    ipc.importerDefaultPaths().then(setPaths);
  }, []);

  return (
    <div className="fixed inset-0 z-modal flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="anim-modal flex max-h-[85vh] w-[36rem] max-w-[92vw] flex-col overflow-hidden rounded-xl border border-border bg-bg-panel shadow-elev-3"
      >
        <div className="flex items-center gap-2 border-b border-border bg-bg-subtle px-4 py-3">
          <Download size={14} className="text-accent" />
          <span id={titleId} className="text-sm font-semibold">Import connections</span>
          <span className="text-[11px] text-text-dim">
            from another client
          </span>
          <div className="flex-1" />
          <button
            onClick={onClose}
            className="rounded-md px-2 py-0.5 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text"
          >
            Close
          </button>
        </div>

        <div className="flex border-b border-border bg-bg-panel px-2 pt-2">
          {(
            [
              { id: "openssh", label: "OpenSSH config", icon: TerminalIcon },
              { id: "filezilla", label: "FileZilla", icon: Server },
              { id: "putty", label: "PuTTY", icon: ShieldCheck },
            ] as const
          ).map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              onClick={() => setTab(id)}
              className={cn(
                "flex items-center gap-1.5 rounded-t-md border-b-2 px-3 py-1.5 text-[12px] transition-colors",
                tab === id
                  ? "border-accent text-text"
                  : "border-transparent text-text-muted hover:text-text"
              )}
            >
              <Icon size={12} />
              {label}
            </button>
          ))}
        </div>

        {(["openssh", "filezilla", "putty"] as const).map((kind) => (
          <ImporterTab
            key={kind}
            active={tab === kind}
            kind={kind}
            paths={paths}
            onDone={onClose}
          />
        ))}
      </div>
    </div>
  );
}

function ImporterTab({
  active,
  kind,
  paths,
  onDone,
}: {
  active: boolean;
  kind: ImporterKind;
  paths: ImporterPaths | null;
  onDone: () => void;
}) {
  const loadProfiles = useConnections((s) => s.loadProfiles);
  const [path, setPath] = useState<string>("");
  const [pathTouched, setPathTouched] = useState(false);
  const [previews, setPreviews] = useState<ProfilePreview[] | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [savedCount, setSavedCount] = useState<number | null>(null);
  const [autoScanned, setAutoScanned] = useState(false);

  // Pre-populate the path from the auto-detected default the moment paths arrive.
  useEffect(() => {
    if (paths && !pathTouched) {
      const next =
        kind === "openssh"
          ? paths.openssh ?? ""
          : kind === "filezilla"
            ? paths.filezilla ?? ""
            : paths.putty ?? "";
      setPath(next);
    }
  }, [paths, kind, pathTouched]);

  const scan = useCallback(async () => {
    setLoading(true);
    setError(null);
    setSavedCount(null);
    try {
      let result: ProfilePreview[];
      if (kind === "openssh") {
        result = await ipc.importOpenssh(path || undefined);
      } else if (kind === "filezilla") {
        result = await ipc.importFilezilla(path || undefined);
      } else {
        result = await ipc.importPutty();
      }
      setPreviews(result);
      // Select everything by default — users usually want to bring it all in
      // and then drop the few they don't.
      setSelected(new Set(result.map((p) => p.previewId)));
    } catch (e) {
      setError(String(e));
      setPreviews(null);
    } finally {
      setLoading(false);
      setAutoScanned(true);
    }
  }, [kind, path]);

  // Auto-scan when the tab opens, but only when the path is autodetected.
  useEffect(() => {
    if (active && !autoScanned && !pathTouched && (kind === "putty" || path)) {
      scan();
    }
  }, [active, autoScanned, kind, path, pathTouched, scan]);

  const toggle = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  const allSelected = useMemo(
    () => previews !== null && previews.every((p) => selected.has(p.previewId)),
    [previews, selected]
  );

  const importSelected = async () => {
    if (!previews) return;
    const chosen = previews.filter((p) => selected.has(p.previewId));
    if (chosen.length === 0) return;
    setSaving(true);
    setError(null);
    try {
      const n = await ipc.saveImportedProfiles(chosen);
      setSavedCount(n);
      await loadProfiles();
      // Auto-close after a moment so the user sees the success state.
      setTimeout(onDone, 900);
    } catch (e) {
      setError(String(e));
    } finally {
      setSaving(false);
    }
  };

  const isRegistry = kind === "putty" && path.startsWith("HKCU\\");

  return (
    <div
      className={cn(
        "flex flex-1 flex-col overflow-hidden px-4 py-3",
        !active && "hidden"
      )}
    >
      <div className="mb-3 flex items-end gap-2">
        <label className="flex-1">
          <div className="mb-1 text-[10px] uppercase tracking-wider text-text-muted">
            {kind === "putty" && isRegistry ? "Source (registry)" : "File path"}
          </div>
          <input
            value={path}
            onChange={(e) => {
              setPath(e.target.value);
              setPathTouched(true);
            }}
            disabled={isRegistry}
            className="w-full rounded-md border border-border bg-bg-subtle px-2.5 py-1.5 font-mono text-[11.5px] outline-none focus:border-accent disabled:opacity-60"
            placeholder={hintFor(kind)}
          />
        </label>
        <button
          onClick={scan}
          disabled={loading || (kind !== "putty" && !path)}
          className="flex items-center gap-1.5 rounded-md border border-border bg-bg-subtle px-2.5 py-1.5 text-[12px] hover:bg-bg-hover disabled:opacity-40"
          title="Re-scan"
        >
          {loading ? (
            <Loader2 size={12} className="animate-spin" />
          ) : (
            <RefreshCw size={12} />
          )}
          Scan
        </button>
      </div>

      {error && (
        <div className="mb-2 flex items-start gap-2 rounded-md border border-danger/30 bg-danger-soft px-2.5 py-2 text-[11.5px] text-text">
          <AlertTriangle size={12} className="mt-0.5 shrink-0 text-danger" />
          <span className="break-all">{error}</span>
        </div>
      )}

      {savedCount !== null && (
        <div className="mb-2 flex items-center gap-2 rounded-md border border-accent/30 bg-accent-soft px-2.5 py-2 text-[11.5px]">
          <CheckCircle2 size={12} className="text-accent" />
          Imported {savedCount} connection{savedCount === 1 ? "" : "s"}.
        </div>
      )}

      <div className="flex-1 overflow-y-auto rounded-md border border-border bg-bg-subtle">
        {previews === null && !loading && !error && (
          <Empty>
            Click <strong>Scan</strong> to read connections from this source.
          </Empty>
        )}
        {previews && previews.length === 0 && (
          <Empty>
            No connections found.{" "}
            {kind === "openssh"
              ? "Try editing the path — or check that ~/.ssh/config exists."
              : kind === "filezilla"
                ? "Check that FileZilla has any saved sites."
                : "No PuTTY sessions are registered for this user."}
          </Empty>
        )}
        {previews && previews.length > 0 && (
          <>
            <div className="sticky top-0 z-sticky flex items-center gap-2 border-b border-border bg-bg-panel px-3 py-1.5 text-[11px] text-text-muted">
              <input
                type="checkbox"
                checked={allSelected}
                onChange={(e) => {
                  if (e.target.checked) {
                    setSelected(new Set(previews.map((p) => p.previewId)));
                  } else {
                    setSelected(new Set());
                  }
                }}
                className="accent-accent"
              />
              <span>
                {selected.size} of {previews.length} selected
              </span>
            </div>
            {previews.map((p) => (
              <PreviewRow
                key={p.previewId}
                preview={p}
                checked={selected.has(p.previewId)}
                onToggle={() => toggle(p.previewId)}
              />
            ))}
          </>
        )}
      </div>

      <div className="mt-3 flex items-center justify-end gap-2">
        <button
          onClick={onDone}
          className="rounded-md border border-border bg-bg-subtle px-3 py-1.5 text-xs hover:bg-bg-hover"
        >
          Cancel
        </button>
        <button
          onClick={importSelected}
          disabled={saving || !previews || selected.size === 0}
          className="btn-accent flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium text-white disabled:opacity-40"
        >
          {saving ? (
            <Loader2 size={11} className="animate-spin" />
          ) : (
            <Download size={11} />
          )}
          Import {selected.size > 0 ? `(${selected.size})` : ""}
        </button>
      </div>
    </div>
  );
}

function PreviewRow({
  preview: p,
  checked,
  onToggle,
}: {
  preview: ProfilePreview;
  checked: boolean;
  onToggle: () => void;
}) {
  return (
    <label
      className={cn(
        "flex cursor-pointer items-center gap-2.5 border-b border-border-subtle px-3 py-2 hover:bg-bg-hover",
        checked ? "bg-bg-panel" : ""
      )}
    >
      <input
        type="checkbox"
        checked={checked}
        onChange={onToggle}
        className="accent-accent"
      />
      <ProtocolBadge protocol={p.protocol} />
      <div className="min-w-0 flex-1">
        <div className="truncate text-[12.5px] font-medium">{p.name}</div>
        <div className="truncate font-mono text-[11px] text-text-dim">
          {p.username ? `${p.username}@` : ""}
          {p.host}
          {p.port && p.port !== defaultPortFor(p.protocol)
            ? `:${p.port}`
            : ""}
        </div>
      </div>
      {p.identityFile && (
        <span
          title={p.identityFile}
          className="rounded bg-bg-subtle px-1.5 py-0.5 font-mono text-[10px] text-text-dim"
        >
          <FileText size={10} className="mr-0.5 inline align-text-bottom" />
          key
        </span>
      )}
      {p.note && (
        <span className="text-[10px] text-text-dim">{p.note}</span>
      )}
    </label>
  );
}

function ProtocolBadge({ protocol }: { protocol: Protocol }) {
  let Icon = TerminalIcon;
  let label = protocol.toUpperCase();
  if (protocol === "ftp") Icon = ShieldOff;
  else if (protocol === "ftps") Icon = ShieldCheck;
  else if (protocol === "s3" || protocol === "azure" || protocol === "gcs") Icon = Cloud;
  else if (protocol === "webdav" || protocol === "http") Icon = Globe;
  else if (protocol === "dropbox" || protocol === "onedrive" || protocol === "gdrive" || protocol === "box") Icon = Cloud;
  return (
    <span className="flex h-5 items-center gap-1 rounded bg-bg-subtle px-1.5 text-[10px] font-semibold uppercase tracking-wider text-text-muted">
      <Icon size={9} />
      {label}
    </span>
  );
}

function defaultPortFor(p: Protocol): number {
  if (p === "sftp") return 22;
  if (p === "ftp" || p === "ftps") return 21;
  return 443;
}

function hintFor(kind: ImporterKind): string {
  switch (kind) {
    case "openssh":
      return "~/.ssh/config";
    case "filezilla":
      return "sitemanager.xml";
    case "putty":
      return "(PuTTY registry)";
  }
}

function Empty({ children }: { children: React.ReactNode }) {
  return (
    <div className="px-4 py-10 text-center text-[12px] text-text-dim">
      {children}
    </div>
  );
}
