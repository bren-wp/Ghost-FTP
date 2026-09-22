import { useEffect, useMemo, useState } from "react";
import {
  ArrowRight,
  ArrowLeft,
  RefreshCw,
  Trash2,
  Plus,
  AlertTriangle,
  Folder,
  FolderSync,
  Loader2,
  Cloud,
  CloudOff,
  X,
} from "lucide-react";
import { open } from "@tauri-apps/plugin-dialog";
import { ipc } from "@/lib/ipc";
import { useSync } from "@/stores/syncStore";
import { toast } from "@/stores/toastStore";
import { useConnections } from "@/stores/connectionsStore";
import { relTime } from "@/lib/format";
import { cn } from "@/lib/cn";
import type {
  PairView,
  SyncDirection,
  SyncMode,
  SyncPair,
  SyncStrategy,
} from "@/lib/types";

/// Settings → Folder Sync. Configure continuous, watched sync pairs: pick a
/// local folder + a saved connection + a remote path, choose a direction and
/// strategy, and the backend keeps them in sync in the background. The list of
/// pairs (and their live status) lives in the sync store, driven by the
/// "foldersync://changed" event.
export function SyncSettings() {
  const pairs = useSync((s) => s.pairs);
  const [adding, setAdding] = useState(false);

  // Make sure the store is populated even if the pill/App init hasn't run yet.
  useEffect(() => {
    void useSync.getState().refresh();
  }, []);

  return (
    <div className="space-y-4">
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5 text-sm">
            <FolderSync size={14} className="text-accent" />
            Keep folders in sync automatically
          </div>
          <div className="mt-1 text-xs text-text-dim">
            Watch local folders in the background and synchronize changes to or from your server. Each sync pair runs independently while Ghost FTP is open.
          </div>
        </div>
        {!adding && (
          <button
            type="button"
            onClick={() => setAdding(true)}
            className="btn-accent flex shrink-0 items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium text-white"
          >
            <Plus size={13} /> New sync pair
          </button>
        )}
      </div>

      {adding && <PairForm onDone={() => setAdding(false)} />}

      <div className="space-y-1.5">
        {pairs.length === 0 ? (
          <div className="rounded-lg border border-dashed border-border bg-bg-subtle/50 px-3 py-6 text-center text-xs text-text-dim">
            No sync pairs yet. Create one to keep a folder mirrored to a server.
          </div>
        ) : (
          pairs.map((p) => <PairRow key={p.id} pair={p} />)
        )}
      </div>
    </div>
  );
}

function PairRow({ pair }: { pair: PairView }) {
  const profiles = useConnections((s) => s.profiles);
  const setEnabled = useSync((s) => s.setEnabled);
  const remove = useSync((s) => s.remove);
  const syncNow = useSync((s) => s.syncNow);
  const [freeing, setFreeing] = useState(false);

  const profile = profiles.find((p) => p.id === pair.profileId);
  const Arrow = pair.direction === "localToRemote" ? ArrowRight : ArrowLeft;
  const syncing = pair.state === "syncing";
  const isOnDemand = pair.mode === "onDemand";

  const freeUpSpace = async () => {
    setFreeing(true);
    try {
      await ipc.virtualFsFreeUpSpace(pair.id);
    } catch (error) {
      toast.error("Couldn't free up local space", String(error));
    } finally {
      setFreeing(false);
    }
  };

  return (
    <div className="rounded-lg border border-border bg-bg-subtle p-3">
      <div className="flex items-start gap-3">
        <StatusDot pair={pair} />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="truncate text-sm font-medium text-text">
              {pair.name}
            </span>
            <span className="shrink-0 text-[11px] text-text-dim">
              {profile?.name ?? "unknown server"}
            </span>
          </div>
          <div className="mt-1 flex items-center gap-1.5 truncate font-mono text-[11px] text-text-muted">
            <Folder size={10} className="shrink-0 text-text-dim" />
            <span className="truncate">{pair.localRoot}</span>
            <Arrow size={11} className="shrink-0 text-accent" />
            <span className="truncate">{pair.remoteRoot}</span>
          </div>
          <div className="mt-1 flex flex-wrap items-center gap-x-2.5 gap-y-0.5 text-[11px] text-text-dim">
            {isOnDemand ? (
              <span className="flex items-center gap-1 uppercase tracking-wider text-accent">
                <Cloud size={10} /> on-demand
              </span>
            ) : (
              <span className="uppercase tracking-wider">{pair.strategy}</span>
            )}
            {pair.exclude?.length > 0 && (
              <span title={pair.exclude.join("\n")}>
                {pair.exclude.length} exclude
                {pair.exclude.length === 1 ? "" : "s"}
              </span>
            )}
            {isOnDemand ? (
              <span>Hydrates on open</span>
            ) : syncing ? (
              <span className="flex items-center gap-1 text-warning">
                <Loader2 size={10} className="animate-spin" />
                Syncing{pair.inFlight > 0 ? ` · ${pair.inFlight} in flight` : ""}
              </span>
            ) : pair.lastSynced ? (
              <span>Synced {relTime(pair.lastSynced)} ago</span>
            ) : (
              <span>Never synced</span>
            )}
            {!pair.enabled && <span className="text-text-dim">Paused</span>}
          </div>
          {pair.lastError && (
            <div className="mt-1 flex items-start gap-1.5 text-[11px] text-danger">
              <AlertTriangle size={11} className="mt-0.5 shrink-0" />
              <span className="break-all">{pair.lastError}</span>
            </div>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-1.5">
          {isOnDemand ? (
            <button
              type="button"
              onClick={freeUpSpace}
              disabled={!pair.enabled || freeing}
              title="Free up space — evict downloaded files back to placeholders"
              className="flex items-center gap-1 rounded-md border border-border bg-bg-panel px-2 py-1 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-40"
            >
              <CloudOff size={11} className={cn(freeing && "animate-pulse")} />
              Free up space
            </button>
          ) : (
            <button
              type="button"
              onClick={() => syncNow(pair.id)}
              disabled={!pair.enabled || syncing}
              title="Sync now"
              className="flex items-center gap-1 rounded-md border border-border bg-bg-panel px-2 py-1 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-40"
            >
              <RefreshCw size={11} className={cn(syncing && "animate-spin")} />
              Sync now
            </button>
          )}
          <Toggle
            checked={pair.enabled}
            onChange={(v) => setEnabled(pair.id, v)}
          />
          <button
            type="button"
            onClick={() => remove(pair.id)}
            title="Remove this sync pair"
            className="flex items-center rounded-md p-1.5 text-text-dim hover:bg-danger/10 hover:text-danger"
          >
            <Trash2 size={12} />
          </button>
        </div>
      </div>
    </div>
  );
}

function StatusDot({ pair }: { pair: PairView }) {
  const tone = !pair.enabled
    ? "bg-text-dim"
    : pair.state === "error"
      ? "bg-danger"
      : pair.running || pair.state === "syncing" || pair.state === "idle"
        ? "bg-success"
        : "bg-text-dim";
  const pulse = pair.state === "syncing";
  return (
    <div className="relative mt-1.5">
      <div className={cn("h-2 w-2 rounded-full", tone)} />
      {pulse && (
        <div className={cn("absolute inset-0 h-2 w-2 animate-ping rounded-full opacity-60", tone)} />
      )}
    </div>
  );
}

function PairForm({ onDone }: { onDone: () => void }) {
  const profiles = useConnections((s) => s.profiles);
  const upsert = useSync((s) => s.upsert);

  const profileOptions = useMemo(
    () => profiles.map((p) => ({ value: p.id, label: p.name })),
    [profiles]
  );

  const [name, setName] = useState("");
  const [profileId, setProfileId] = useState(profiles[0]?.id ?? "");
  const [localRoot, setLocalRoot] = useState("");
  const [remoteRoot, setRemoteRoot] = useState(
    profiles[0]?.defaultRemotePath ?? ""
  );
  const [direction, setDirection] = useState<SyncDirection>("localToRemote");
  const [strategy, setStrategy] = useState<SyncStrategy>("additive");
  const [mode, setMode] = useState<SyncMode>("mirror");
  const [vfsSupported, setVfsSupported] = useState(false);
  const [pollIntervalSecs, setPollIntervalSecs] = useState(60);
  const [excludeText, setExcludeText] = useState("");
  const [mirrorDeleteCap, setMirrorDeleteCap] = useState(100);
  const [busy, setBusy] = useState(false);

  // On-demand placeholders are Windows-only (and gated behind the `virtualfs`
  // build); only offer the mode where it actually works.
  useEffect(() => {
    void ipc
      .virtualFsSupported()
      .then(setVfsSupported)
      .catch((error) => {
        console.warn("Couldn't determine virtual filesystem support", error);
        setVfsSupported(false);
      });
  }, []);

  const pickProfile = (id: string) => {
    setProfileId(id);
    // Seed the remote path from the profile's default when it's still blank.
    const prof = profiles.find((p) => p.id === id);
    if (prof?.defaultRemotePath && !remoteRoot) setRemoteRoot(prof.defaultRemotePath);
  };

  const browse = async () => {
    const picked = await open({ directory: true, title: "Choose a local folder" });
    if (typeof picked === "string") {
      setLocalRoot(picked);
      if (!name) {
        const base = picked.split(/[/\\]/).filter(Boolean).pop();
        if (base) setName(base);
      }
    }
  };

  const canSubmit =
    name.trim() !== "" &&
    profileId !== "" &&
    localRoot.trim() !== "" &&
    remoteRoot.trim() !== "";

  const submit = async () => {
    if (!canSubmit) return;
    setBusy(true);
    const pair: SyncPair = {
      id: "",
      name: name.trim(),
      localRoot: localRoot.trim(),
      profileId,
      remoteRoot: remoteRoot.trim(),
      direction,
      strategy,
      mode,
      enabled: true,
      pollIntervalSecs,
      exclude: parseExclude(excludeText),
      mirrorDeleteCap,
    };
    try {
      await upsert(pair);
      onDone();
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-3 rounded-lg border border-border bg-bg-subtle p-3">
      <div className="flex items-center justify-between">
        <div className="text-xs font-semibold text-text">New sync pair</div>
        <button
          type="button"
          onClick={onDone}
          className="rounded-md p-1 text-text-muted hover:bg-bg-hover hover:text-text"
          title="Cancel"
        >
          <X size={13} />
        </button>
      </div>

      <Field label="Name">
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="My project"
          className="w-full rounded-md border border-border bg-bg-panel px-2.5 py-1.5 text-sm outline-none focus:border-accent"
        />
      </Field>

      <Field label="Connection">
        {profileOptions.length === 0 ? (
          <div className="text-xs text-text-dim">
            No saved connections yet. Add one first.
          </div>
        ) : (
          <Select
            value={profileId}
            onChange={pickProfile}
            options={profileOptions}
          />
        )}
      </Field>

      <Field label="Local folder">
        <div className="flex w-full items-center gap-1.5">
          <input
            value={localRoot}
            onChange={(e) => setLocalRoot(e.target.value)}
            placeholder="C:\\path\\to\\folder"
            className="min-w-0 flex-1 rounded-md border border-border bg-bg-panel px-2.5 py-1.5 text-sm outline-none focus:border-accent"
          />
          <button
            type="button"
            onClick={browse}
            className="shrink-0 rounded-md border border-border px-2.5 py-1.5 text-sm text-text-muted hover:bg-bg-hover hover:text-text"
          >
            Browse…
          </button>
        </div>
      </Field>

      <Field label="Remote path">
        <input
          value={remoteRoot}
          onChange={(e) => setRemoteRoot(e.target.value)}
          placeholder="/home/user/folder"
          className="w-full rounded-md border border-border bg-bg-panel px-2.5 py-1.5 font-mono text-sm outline-none focus:border-accent"
        />
      </Field>

      {vfsSupported && (
        <Field label="Mode">
          <Segmented<SyncMode>
            value={mode}
            onChange={setMode}
            options={[
              { value: "mirror", label: "Mirror (copy files)" },
              { value: "onDemand", label: "On-demand (placeholders)" },
            ]}
          />
          {mode === "onDemand" && (
            <div className="mt-2 flex items-start gap-2 rounded-md border border-accent/30 bg-accent-soft/40 px-2.5 py-2 text-[11.5px] text-text-muted">
              <Cloud size={12} className="mt-0.5 shrink-0 text-accent" />
              <span>
                Files show in the folder as placeholders and download only when
                opened. Use “Free up space” to evict them back to placeholders.
                Direction, strategy, and excludes don’t apply.
              </span>
            </div>
          )}
        </Field>
      )}

      {mode === "mirror" && (
        <>
      <div className="grid grid-cols-2 gap-3">
        <Field label="Direction">
          <Segmented<SyncDirection>
            value={direction}
            onChange={setDirection}
            options={[
              { value: "localToRemote", label: "Local → Remote" },
              { value: "remoteToLocal", label: "Remote → Local" },
            ]}
          />
        </Field>
        <Field label="Strategy">
          <Segmented<SyncStrategy>
            value={strategy}
            onChange={setStrategy}
            options={[
              { value: "additive", label: "Additive" },
              { value: "mirror", label: "Mirror" },
            ]}
          />
        </Field>
      </div>

      {strategy === "mirror" && (
        <div className="space-y-2 rounded-md border border-danger/30 bg-danger-soft/60 px-2.5 py-2">
          <div className="flex items-start gap-2 text-[11.5px] text-text-muted">
            <AlertTriangle size={12} className="mt-0.5 shrink-0 text-danger" />
            Mirror deletes files on the destination that don't exist on the
            source. As a safeguard, Ghost FTP refuses to delete when the source folder
            is empty or unreadable, and caps deletions per sync.
          </div>
          <div className="flex items-center gap-1.5 pl-5">
            <span className="text-[11px] text-text-muted">
              Max deletes per sync
            </span>
            <input
              type="number"
              min={0}
              max={100000}
              value={mirrorDeleteCap}
              onChange={(e) =>
                setMirrorDeleteCap(
                  Math.max(0, Math.min(100000, parseInt(e.target.value) || 0))
                )
              }
              className="w-20 rounded-md border border-border bg-bg-panel px-2 py-1 text-sm outline-none focus:border-accent"
            />
            <span className="text-[11px] text-text-dim">0 = no cap</span>
          </div>
        </div>
      )}

      <Field label="Exclude patterns">
        <textarea
          value={excludeText}
          onChange={(e) => setExcludeText(e.target.value)}
          placeholder={"node_modules\n.git\n*.tmp\n/dist"}
          rows={3}
          spellCheck={false}
          className="w-full resize-y rounded-md border border-border bg-bg-panel px-2.5 py-1.5 font-mono text-[12px] outline-none focus:border-accent"
        />
        <div className="mt-1 text-[10.5px] text-text-dim">
          One gitignore-style pattern per line (also comma-separated). Matches are
          never uploaded or mirror-deleted.
        </div>
      </Field>

      <Field label="Poll interval">
        <div className="flex items-center gap-1.5">
          <input
            type="number"
            min={5}
            max={86400}
            value={pollIntervalSecs}
            onChange={(e) =>
              setPollIntervalSecs(
                Math.max(5, Math.min(86400, parseInt(e.target.value) || 60))
              )
            }
            className="w-24 rounded-md border border-border bg-bg-panel px-2.5 py-1.5 text-sm outline-none focus:border-accent"
          />
          <span className="text-xs text-text-dim">seconds</span>
        </div>
      </Field>
        </>
      )}

      <div className="flex items-center justify-end gap-2 pt-1">
        <button
          type="button"
          onClick={onDone}
          className="rounded-md border border-border bg-bg-panel px-3 py-1.5 text-xs hover:bg-bg-hover"
        >
          Cancel
        </button>
        <button
          type="button"
          onClick={submit}
          disabled={!canSubmit || busy}
          className="btn-accent flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium text-white disabled:opacity-40"
        >
          {busy ? <Loader2 size={12} className="animate-spin" /> : <Plus size={12} />}
          Create pair
        </button>
      </div>
    </div>
  );
}

/** Split the exclude textarea into clean patterns (newline- or comma-separated,
 *  blanks and `#` comments dropped). Mirrors the backend's tolerant parsing. */
function parseExclude(text: string): string[] {
  return text
    .split(/[\n,]/)
    .map((s) => s.trim())
    .filter((s) => s !== "" && !s.startsWith("#"));
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <div className="mb-1 text-[11px] font-medium text-text-muted">{label}</div>
      {children}
    </div>
  );
}

function Select({
  value,
  onChange,
  options,
}: {
  value: string;
  onChange: (v: string) => void;
  options: { value: string; label: string }[];
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="w-full rounded-md border border-border bg-bg-panel px-2.5 py-1.5 text-sm outline-none hover:border-text-dim focus:border-accent"
    >
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  );
}

function Segmented<T extends string>({
  value,
  onChange,
  options,
}: {
  value: T;
  onChange: (v: T) => void;
  options: { value: T; label: string }[];
}) {
  return (
    <div className="inline-flex w-full rounded-md border border-border bg-bg-panel p-0.5">
      {options.map((o) => {
        const active = value === o.value;
        return (
          <button
            key={o.value}
            type="button"
            onClick={() => onChange(o.value)}
            className={cn(
              "flex-1 rounded-[5px] px-2 py-1 text-xs font-medium transition-colors",
              active
                ? "bg-bg-subtle text-text shadow-elev-1"
                : "text-text-muted hover:text-text"
            )}
          >
            {o.label}
          </button>
        );
      })}
    </div>
  );
}

function Toggle({
  checked,
  onChange,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={() => onChange(!checked)}
      className={cn(
        "relative h-5 w-9 shrink-0 rounded-full border transition-colors",
        checked
          ? "bg-accent border-accent"
          : "bg-bg-subtle border-border hover:border-text-dim"
      )}
    >
      <span
        className={cn(
          "absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-white shadow-elev-1 transition-transform",
          checked ? "translate-x-4" : "translate-x-0"
        )}
      />
    </button>
  );
}
