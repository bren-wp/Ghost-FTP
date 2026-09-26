import { useId, useRef, useState } from "react";
import {
  ArrowRightLeft,
  ArrowRight,
  ArrowLeft,
  RefreshCw,
  Trash2,
  Loader2,
  AlertTriangle,
  Plus,
  Pencil,
  Maximize2,
  Folder,
} from "lucide-react";
import { ipc } from "@/lib/ipc";
import { useConnections } from "@/stores/connectionsStore";
import { useTransfers } from "@/stores/transfersStore";
import { useDialog } from "@/hooks/useDialog";
import type {
  SyncDirection,
  SyncPlan,
  SyncReason,
  SyncStrategy,
} from "@/lib/types";
import { cn } from "@/lib/cn";

interface Props {
  localPath: string;
  remotePath: string;
  onClose: () => void;
}

/// Pick direction + strategy → compute plan → preview → execute. Execution
/// reuses the existing TransferManager so progress lands in the same queue
/// the user already knows.
export function SyncDialog({ localPath, remotePath, onClose }: Props) {
  const sessionId = useConnections((s) => s.activeSessionId);
  const activeProfile = useConnections((s) =>
    s.profiles.find((p) => p.id === s.activeProfileId)
  );
  const togglePanel = useTransfers((s) => s.togglePanel);

  const [direction, setDirection] = useState<SyncDirection>("localToRemote");
  const [strategy, setStrategy] = useState<SyncStrategy>("additive");
  const [plan, setPlan] = useState<SyncPlan | null>(null);
  const [planning, setPlanning] = useState(false);
  const [executing, setExecuting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const titleId = useId();
  useDialog(panelRef, { onClose });

  // The remote pane can be browsing into a path that doesn't make sense as
  // a sync root (e.g. an object-store flat prefix). We don't gate on that
  // here — the planner returns an empty list and the user sees the result.

  const computePlan = async () => {
    if (!sessionId) return;
    setPlanning(true);
    setError(null);
    setPlan(null);
    try {
      const result = await ipc.syncPlan(
        sessionId,
        localPath,
        remotePath,
        direction,
        strategy
      );
      setPlan(result);
    } catch (e) {
      setError(String(e));
    } finally {
      setPlanning(false);
    }
  };

  const execute = async () => {
    if (!sessionId || !plan) return;
    setExecuting(true);
    setError(null);
    try {
      await ipc.syncExecute(sessionId, plan);
      togglePanel(); // open the transfer queue so progress is visible
      onClose();
    } catch (e) {
      setError(String(e));
      setExecuting(false);
    }
  };

  const totalOps =
    (plan?.copies.length ?? 0) + (plan?.deletes.length ?? 0);

  return (
    <div className="fixed inset-0 z-modal flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="anim-modal flex max-h-[88vh] w-[40rem] max-w-[92vw] flex-col overflow-hidden rounded-xl border border-border bg-bg-panel shadow-elev-3"
      >
        <div className="flex items-center gap-2 border-b border-border bg-bg-subtle px-4 py-3">
          <ArrowRightLeft size={14} className="text-accent" />
          <span id={titleId} className="text-sm font-semibold">Sync directory</span>
          {activeProfile && (
            <span className="text-[11px] text-text-dim">
              · {activeProfile.name}
            </span>
          )}
          <div className="flex-1" />
          <button
            onClick={onClose}
            className="rounded-md px-2 py-0.5 text-[11px] text-text-muted hover:bg-bg-hover hover:text-text"
          >
            Close
          </button>
        </div>

        <div className="space-y-3 px-4 py-3">
          <PathBar localPath={localPath} remotePath={remotePath} direction={direction} />

          <div className="grid grid-cols-2 gap-2">
            <div>
              <div className="mb-1 text-[10px] uppercase tracking-wider text-text-muted">
                Direction
              </div>
              <div className="grid grid-cols-2 gap-1 rounded-md border border-border bg-bg-subtle p-1">
                <DirectionButton
                  active={direction === "localToRemote"}
                  onClick={() => setDirection("localToRemote")}
                  icon={<ArrowRight size={12} />}
                  label="Local → Remote"
                />
                <DirectionButton
                  active={direction === "remoteToLocal"}
                  onClick={() => setDirection("remoteToLocal")}
                  icon={<ArrowLeft size={12} />}
                  label="Remote → Local"
                />
              </div>
            </div>
            <div>
              <div className="mb-1 text-[10px] uppercase tracking-wider text-text-muted">
                Strategy
              </div>
              <div className="grid grid-cols-2 gap-1 rounded-md border border-border bg-bg-subtle p-1">
                <DirectionButton
                  active={strategy === "additive"}
                  onClick={() => setStrategy("additive")}
                  icon={<Plus size={12} />}
                  label="Additive"
                  hint="copy only"
                />
                <DirectionButton
                  active={strategy === "mirror"}
                  onClick={() => setStrategy("mirror")}
                  icon={<Trash2 size={12} />}
                  label="Mirror"
                  hint="includes deletes"
                  danger
                />
              </div>
            </div>
          </div>

          {strategy === "mirror" && (
            <div className="flex items-start gap-2 rounded-md border border-danger/30 bg-danger-soft/60 px-2.5 py-2 text-[11.5px] text-text-muted">
              <AlertTriangle size={12} className="mt-0.5 shrink-0 text-danger" />
              Mirror deletes files on the destination that don't exist on the
              source. Review the plan before executing.
            </div>
          )}

          <div className="flex justify-end">
            <button
              onClick={computePlan}
              disabled={!sessionId || planning}
              className="flex items-center gap-1.5 rounded-md border border-border bg-bg-subtle px-2.5 py-1.5 text-[12px] hover:bg-bg-hover disabled:opacity-40"
            >
              {planning ? (
                <Loader2 size={12} className="animate-spin" />
              ) : (
                <RefreshCw size={12} />
              )}
              {plan ? "Re-plan" : "Plan"}
            </button>
          </div>
        </div>

        {error && (
          <div className="mx-4 mb-3 flex items-start gap-2 rounded-md border border-danger/30 bg-danger-soft px-2.5 py-2 text-[11.5px]">
            <AlertTriangle size={12} className="mt-0.5 shrink-0 text-danger" />
            <span className="break-all">{error}</span>
          </div>
        )}

        {plan && (
          <PlanSummary plan={plan} />
        )}

        <div className="mt-auto flex items-center justify-between gap-2 border-t border-border bg-bg-subtle px-4 py-3 text-[11px] text-text-dim">
          <span>
            {plan
              ? `${plan.copies.length} copies · ${plan.deletes.length} deletes · ${fmtBytes(plan.totalBytes)}`
              : "Nothing planned yet."}
          </span>
          <div className="flex items-center gap-2">
            <button
              onClick={onClose}
              className="rounded-md border border-border bg-bg-panel px-3 py-1.5 text-xs hover:bg-bg-hover"
            >
              Cancel
            </button>
            <button
              onClick={execute}
              disabled={!plan || totalOps === 0 || executing}
              className="btn-accent flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium text-white disabled:opacity-40"
            >
              {executing ? (
                <Loader2 size={11} className="animate-spin" />
              ) : (
                <ArrowRightLeft size={11} />
              )}
              Execute{totalOps > 0 ? ` (${totalOps})` : ""}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function PathBar({
  localPath,
  remotePath,
  direction,
}: {
  localPath: string;
  remotePath: string;
  direction: SyncDirection;
}) {
  const Arrow = direction === "localToRemote" ? ArrowRight : ArrowLeft;
  return (
    <div className="flex items-center gap-2 rounded-md border border-border bg-bg-subtle px-2.5 py-2">
      <PathCell label="Local" path={localPath} />
      <Arrow size={14} className="shrink-0 text-accent" />
      <PathCell label="Remote" path={remotePath} />
    </div>
  );
}

function PathCell({ label, path }: { label: string; path: string }) {
  return (
    <div className="min-w-0 flex-1">
      <div className="text-[9px] uppercase tracking-wider text-text-dim">
        {label}
      </div>
      <div className="flex items-center gap-1 truncate font-mono text-[11.5px]">
        <Folder size={10} className="shrink-0 text-text-dim" />
        <span className="truncate">{path}</span>
      </div>
    </div>
  );
}

function DirectionButton({
  active,
  onClick,
  icon,
  label,
  hint,
  danger,
}: {
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  label: string;
  hint?: string;
  danger?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex flex-col items-start rounded-sm px-2 py-1.5 text-left transition-colors",
        active
          ? danger
            ? "bg-danger-soft text-text ring-1 ring-inset ring-danger/40"
            : "bg-accent-soft text-text ring-1 ring-inset ring-accent/40"
          : "text-text-muted hover:bg-bg-hover hover:text-text"
      )}
    >
      <span className="flex items-center gap-1 text-[11px] font-semibold">
        <span
          className={
            active ? (danger ? "text-danger" : "text-accent") : ""
          }
        >
          {icon}
        </span>
        {label}
      </span>
      {hint && <span className="text-[10px] text-text-dim">{hint}</span>}
    </button>
  );
}

function PlanSummary({ plan }: { plan: SyncPlan }) {
  const both = plan.copies.length > 0 && plan.deletes.length > 0;
  return (
    <div className="mx-4 mb-3 flex-1 overflow-hidden rounded-md border border-border bg-bg-subtle">
      {plan.copies.length === 0 && plan.deletes.length === 0 ? (
        <div className="px-4 py-10 text-center text-[12px] text-text-dim">
          Everything is already in sync.
        </div>
      ) : (
        <div className="max-h-[36vh] overflow-y-auto">
          {plan.copies.length > 0 && (
            <PlanSection
              title={`Copy (${plan.copies.length})`}
              icon={<ArrowRightLeft size={11} className="text-accent" />}
            >
              {plan.copies.slice(0, 200).map((c) => (
                <Row key={c.relative}>
                  <ReasonBadge reason={c.reason} />
                  <span className="min-w-0 flex-1 truncate font-mono">
                    {c.relative}
                  </span>
                  <span className="shrink-0 text-text-dim">
                    {fmtBytes(c.size)}
                  </span>
                </Row>
              ))}
              {plan.copies.length > 200 && (
                <Row>
                  <Maximize2 size={10} className="text-text-dim" />
                  <span className="text-text-dim">
                    …and {plan.copies.length - 200} more
                  </span>
                </Row>
              )}
            </PlanSection>
          )}
          {both && <div className="h-px bg-border" />}
          {plan.deletes.length > 0 && (
            <PlanSection
              title={`Delete (${plan.deletes.length})`}
              icon={<Trash2 size={11} className="text-danger" />}
            >
              {plan.deletes.slice(0, 200).map((d) => (
                <Row key={d.relative} dangerTone>
                  <Pencil size={10} className="text-danger" />
                  <span className="min-w-0 flex-1 truncate font-mono">
                    {d.relative}
                  </span>
                  <span className="shrink-0 text-text-dim">
                    {fmtBytes(d.size)}
                  </span>
                </Row>
              ))}
              {plan.deletes.length > 200 && (
                <Row>
                  <span className="text-text-dim">
                    …and {plan.deletes.length - 200} more
                  </span>
                </Row>
              )}
            </PlanSection>
          )}
        </div>
      )}
    </div>
  );
}

function PlanSection({
  title,
  icon,
  children,
}: {
  title: string;
  icon: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <>
      <div className="sticky top-0 z-sticky flex items-center gap-1.5 border-b border-border bg-bg-panel px-3 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-text-muted">
        {icon}
        {title}
      </div>
      {children}
    </>
  );
}

function Row({
  children,
  dangerTone,
}: {
  children: React.ReactNode;
  dangerTone?: boolean;
}) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 border-b border-border-subtle px-3 py-1 text-[11.5px]",
        dangerTone && "bg-danger-soft/20"
      )}
    >
      {children}
    </div>
  );
}

function ReasonBadge({ reason }: { reason: SyncReason }) {
  const label =
    reason === "missing"
      ? "new"
      : reason === "newer"
        ? "newer"
        : reason === "edited"
          ? "edited"
          : "size";
  const tone =
    reason === "missing" ? "bg-success/15 text-success" : "bg-accent-soft text-accent";
  return (
    <span
      className={cn(
        "rounded px-1.5 py-0 text-[9px] font-semibold uppercase tracking-wider",
        tone
      )}
    >
      {label}
    </span>
  );
}

function fmtBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024)
    return `${(n / (1024 * 1024)).toFixed(1)} MB`;
  return `${(n / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}
