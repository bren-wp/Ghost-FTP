import { useEffect, useId, useRef, useState } from "react";
import { KeyRound, Loader2, X } from "lucide-react";
import { ipc } from "@/lib/ipc";
import { messageOf } from "@/lib/errors";
import { useDialog } from "@/hooks/useDialog";
import { useConnections } from "@/stores/connectionsStore";
import { useLayout } from "@/stores/layoutStore";
import { toast } from "@/stores/toastStore";
import type { GrantConnection, GrantManifest } from "@/lib/types";

interface Props {
  onClose: () => void;
}

type Phase =
  | { state: "loading" }
  | { state: "error"; message: string }
  // `acceptError` is set when the Accept call failed; the manifest stays
  // visible and Accept can be retried.
  | { state: "ready"; manifest: GrantManifest; acceptError?: string }
  | { state: "accepting"; manifest: GrantManifest };

/// Consent dialog for ghostftp://grant deep links (docs/grant-links.md). Fetches
/// the grant manifest from the issuer and shows exactly which servers access
/// is being granted to before anything happens. Only Accept runs the key
/// exchange: a fresh keypair is generated on this device, the *public* key is
/// uploaded, and the servers are imported as ordinary profiles. No password is
/// ever shared.
export function GrantDialog({ onClose }: Props) {
  const grantPrefill = useLayout((s) => s.grantPrefill);
  const [phase, setPhase] = useState<Phase>({ state: "loading" });

  const panelRef = useRef<HTMLDivElement>(null);
  const titleId = useId();
  useDialog(panelRef, { onClose });

  useEffect(() => {
    if (!grantPrefill) return;
    let cancelled = false;
    setPhase({ state: "loading" });
    ipc
      .fetchGrantManifest(grantPrefill.issuer, grantPrefill.token)
      .then((manifest) => {
        if (!cancelled) setPhase({ state: "ready", manifest });
      })
      .catch((e) => {
        if (!cancelled) setPhase({ state: "error", message: messageOf(e) });
      });
    return () => {
      cancelled = true;
    };
  }, [grantPrefill]);

  if (!grantPrefill) return null;

  const accept = async (manifest: GrantManifest) => {
    setPhase({ state: "accepting", manifest });
    try {
      const res = await ipc.acceptGrant(
        grantPrefill.issuer,
        grantPrefill.token,
        manifest
      );
      onClose();
      await useConnections.getState().loadProfiles();
      const imported = res.imported.length;
      toast.success(
        `Imported ${imported} ${imported === 1 ? "connection" : "connections"} into "${res.group}"`,
        res.failed.length > 0
          ? `${res.failed.length} failed: ${res.failed
              .map((f) => `${f.name} (${f.error})`)
              .join(", ")}`
          : undefined
      );
    } catch (e) {
      setPhase({ state: "ready", manifest, acceptError: messageOf(e) });
    }
  };

  const manifest =
    phase.state === "ready" || phase.state === "accepting"
      ? phase.manifest
      : null;

  return (
    <div
      className="fixed inset-0 z-modal flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="anim-modal w-[30rem] max-w-[92vw] overflow-hidden rounded-xl border border-border bg-bg-panel shadow-elev-3"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex flex-col items-center gap-2 border-b border-border bg-bg-subtle px-5 py-5">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl text-white shadow-elev-2 btn-accent">
            <KeyRound size={22} />
          </div>
          <div id={titleId} className="text-base font-semibold">
            Access grant
          </div>
          <div className="max-w-sm text-center text-[12px] leading-relaxed text-text-muted">
            {manifest
              ? `${manifest.issuer} is granting you access`
              : (grantPrefill.name ?? "Someone is granting you server access")}
          </div>
        </div>

        <div className="max-h-72 overflow-y-auto px-5 py-4">
          {phase.state === "loading" && (
            <div className="flex items-center justify-center gap-2 py-8 text-[12px] text-text-muted">
              <Loader2 size={14} className="animate-spin text-accent" />
              Fetching grant details…
            </div>
          )}

          {phase.state === "error" && (
            <div className="rounded-md border border-danger/40 bg-danger/10 px-3 py-2 text-[12px] text-danger">
              {phase.message}
            </div>
          )}

          {manifest && (
            <>
              <div className="grid grid-cols-1 gap-1 text-[12px]">
                <Row label="Grant" value={manifest.name} />
                <Row label="Issuer" value={manifest.issuer} />
                {manifest.expiresAt && (
                  <Row label="Expires" value={formatExpiry(manifest.expiresAt)} />
                )}
              </div>

              <div className="mt-3 mb-1.5 text-[11px] font-medium uppercase tracking-wider text-text-dim">
                {manifest.connections.length}{" "}
                {manifest.connections.length === 1 ? "server" : "servers"}
              </div>
              <div className="grid grid-cols-1 gap-1">
                {manifest.connections.map((c, i) => (
                  <ServerRow key={i} conn={c} />
                ))}
              </div>

              <p className="mt-3 text-[11px] leading-relaxed text-text-dim">
                Accepting generates a new key on this device and grants you
                access. No password is shared.
              </p>

              {phase.state === "ready" && phase.acceptError && (
                <div className="mt-2 rounded-md border border-danger/40 bg-danger/10 px-3 py-2 text-[12px] text-danger">
                  {phase.acceptError}
                </div>
              )}
            </>
          )}
        </div>

        <div className="flex items-center justify-end gap-2 border-t border-border bg-bg-subtle px-3 py-2">
          {phase.state === "error" ? (
            <button
              onClick={onClose}
              className="flex items-center gap-1 rounded-md px-2 py-1 text-[11.5px] text-text-muted hover:bg-bg-hover hover:text-text"
            >
              <X size={11} />
              Close
            </button>
          ) : (
            <>
              <button
                onClick={onClose}
                disabled={phase.state === "accepting"}
                className="rounded-md border border-border px-3.5 py-1.5 text-sm hover:bg-bg-hover disabled:opacity-40 disabled:cursor-not-allowed"
              >
                Cancel
              </button>
              <button
                onClick={() => manifest && void accept(manifest)}
                disabled={!manifest || phase.state === "accepting"}
                className="btn-accent flex items-center gap-1.5 rounded-md px-3.5 py-1.5 text-sm font-medium text-white disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {phase.state === "accepting" && (
                  <Loader2 size={13} className="animate-spin" />
                )}
                {phase.state === "accepting"
                  ? "Generating key and importing…"
                  : "Accept access"}
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-text-dim">{label}</span>
      <span className="truncate text-text-muted">{value}</span>
    </div>
  );
}

function ServerRow({ conn }: { conn: GrantConnection }) {
  const port = conn.port ?? 22;
  return (
    <div className="rounded-md border border-border bg-bg-subtle px-2.5 py-1.5">
      <div className="text-[12px] font-medium">
        {conn.name || conn.host}
      </div>
      <div className="font-mono text-[11px] text-text-muted">
        {conn.username}@{conn.host}:{port}
      </div>
      {conn.jump && (
        <div className="text-[11px] text-text-dim">
          via bastion {conn.jump.host}
        </div>
      )}
    </div>
  );
}

function formatExpiry(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}
