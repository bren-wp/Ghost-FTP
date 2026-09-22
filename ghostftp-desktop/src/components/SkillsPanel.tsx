import { createPortal } from "react-dom";
import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  Wand2,
  X,
  Plus,
  Play,
  Pencil,
  Trash2,
  Check,
  ShieldAlert,
  Server,
  Loader2,
  Eye,
  CircleCheck,
  CircleX,
} from "lucide-react";
import { useSkills } from "@/stores/skillsStore";
import { useConnections } from "@/stores/connectionsStore";
import { useBridge } from "@/stores/bridgeStore";
import { ConfirmModal } from "./ConfirmModal";
import { useDialog } from "@/hooks/useDialog";
import { toast } from "@/stores/toastStore";
import { cn } from "@/lib/cn";
import type {
  Skill,
  SkillParam,
  SkillStep,
  TargetSelector,
  SkillRunResult,
  SkillDryRunResult,
} from "@/lib/types";

// Fleet Skills panel (Plan 8). A Skill is a named, parameterized, multi-step
// shell workflow that fans across one or many connected servers. This full-screen
// panel (mirroring FleetSearch's portal overlay) lets the user browse/author/
// approve skills, pick targets, dry-run, run, and watch the aggregated output.

/** Protocols that can run commands (exec) — everything else is skipped by the
 *  backend, so we only offer these as targets. */
const EXEC_PROTOCOLS = new Set(["sftp", "ghostftp-agent"]);

function emptySkill(): Skill {
  return {
    id: "",
    name: "",
    description: "",
    params: [],
    steps: [{ name: "", command: "" }],
    targets: { all: false, sessions: [] },
    status: "approved",
    createdBy: "user",
    stopOnError: true,
  };
}

function targetLabel(t: TargetSelector): string {
  if (t.all) return "all servers";
  if (t.sessions.length === 0) return "no default targets";
  return t.sessions.join(", ");
}

function splitNames(raw: string): string[] {
  return raw
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
}

/** Connected, agent-enabled, exec-capable connections, by display name. */
function useTargetable(): { name: string; protocol: string }[] {
  const sessions = useConnections((s) => s.sessions);
  const profiles = useConnections((s) => s.profiles);
  const enabled = useBridge((s) => s.status.enabledSessions);
  return useMemo(() => {
    const out: { name: string; protocol: string }[] = [];
    const seen = new Set<string>();
    for (const s of sessions) {
      if (!enabled.includes(s.sessionId)) continue;
      const p = profiles.find((x) => x.id === s.profileId);
      if (!p || !EXEC_PROTOCOLS.has(p.protocol)) continue;
      if (seen.has(p.name)) continue;
      seen.add(p.name);
      out.push({ name: p.name, protocol: p.protocol });
    }
    return out;
  }, [sessions, profiles, enabled]);
}

/** Always-mounted host: boots the store (list + proposal listener) and renders
 *  the panel only while it's open. */
export function SkillsHost() {
  const open = useSkills((s) => s.open);
  const init = useSkills((s) => s.init);
  useEffect(() => {
    let disposed = false;
    let cleanup: (() => void) | undefined;
    void init()
      .then((fn) => {
        if (disposed) fn();
        else cleanup = fn;
      })
      .catch((error) => {
        toast.error("Couldn't initialize Skills", String(error));
      });
    return () => {
      disposed = true;
      cleanup?.();
    };
  }, [init]);
  return open ? <SkillsPanel /> : null;
}

function SkillsPanel() {
  const skills = useSkills((s) => s.skills);
  const close = useSkills((s) => s.close);
  const deleteSkill = useSkills((s) => s.deleteSkill);
  const approveSkill = useSkills((s) => s.approveSkill);

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [editing, setEditing] = useState<Skill | "new" | null>(null);
  const [deleting, setDeleting] = useState<Skill | null>(null);

  const selected = useMemo(
    () => skills.find((s) => s.id === selectedId) ?? null,
    [skills, selectedId]
  );

  // Keep a valid selection; default to the first skill.
  useEffect(() => {
    if (skills.length === 0) {
      setSelectedId(null);
      return;
    }
    if (!selectedId || !skills.some((s) => s.id === selectedId)) {
      setSelectedId(skills[0].id);
    }
  }, [skills, selectedId]);

  // Esc closes the panel (but not while an inner dialog is open).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !editing && !deleting) close();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [close, editing, deleting]);

  return createPortal(
    <div className="anim-modal fixed inset-0 z-modal flex flex-col bg-bg">
      {/* Header */}
      <div className="flex shrink-0 items-center gap-2 border-b border-border bg-bg-panel px-3 py-2">
        <Wand2 size={15} className="shrink-0 text-accent" />
        <span className="shrink-0 text-sm font-semibold">Fleet Skills</span>
        <span className="hidden shrink-0 text-xs text-text-dim sm:inline">
          Named, multi-step automations you (or the AI) run across servers
        </span>
        <div className="ml-auto flex items-center gap-2">
          <button
            onClick={() => setEditing("new")}
            className="flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs hover:bg-bg-subtle"
          >
            <Plus size={13} /> New skill
          </button>
          <button
            onClick={close}
            className="rounded-md p-1 hover:bg-bg-subtle"
            title="Close (Esc)"
          >
            <X size={16} />
          </button>
        </div>
      </div>

      {/* Body: list + detail */}
      <div className="flex min-h-0 flex-1">
        <div className="w-72 shrink-0 overflow-y-auto border-r border-border">
          {skills.length === 0 ? (
            <div className="p-4 text-xs leading-relaxed text-text-dim">
              No skills yet. Click <b>New skill</b> to author one, or ask your AI
              agent (over the Agent Bridge) to propose one — it'll show up here as
              a proposal for you to approve.
            </div>
          ) : (
            skills.map((s) => (
              <SkillRow
                key={s.id}
                skill={s}
                selected={s.id === selectedId}
                onSelect={() => setSelectedId(s.id)}
                onEdit={() => setEditing(s)}
                onDelete={() => setDeleting(s)}
              />
            ))
          )}
        </div>

        <div className="min-w-0 flex-1 overflow-y-auto">
          {selected ? (
            <SkillDetail
              key={selected.id}
              skill={selected}
              onApprove={() => approveSkill(selected.id)}
              onEdit={() => setEditing(selected)}
            />
          ) : (
            <div className="flex h-full items-center justify-center text-sm text-text-dim">
              Select a skill to run it.
            </div>
          )}
        </div>
      </div>

      {editing && (
        <SkillEditor
          skill={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
      {deleting && (
        <ConfirmModal
          title={`Delete "${deleting.name}"?`}
          message="This removes the skill and its skill tool. This can't be undone."
          destructive
          confirmLabel="Delete"
          onClose={() => setDeleting(null)}
          onConfirm={() => {
            deleteSkill(deleting.id);
            setDeleting(null);
          }}
        />
      )}
    </div>,
    document.body
  );
}

function SkillRow({
  skill,
  selected,
  onSelect,
  onEdit,
  onDelete,
}: {
  skill: Skill;
  selected: boolean;
  onSelect: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const proposed = skill.status === "proposed";
  return (
    <div
      onClick={onSelect}
      className={cn(
        "group cursor-pointer border-b border-border/50 px-3 py-2",
        selected ? "bg-bg-subtle" : "hover:bg-bg-subtle/50"
      )}
    >
      <div className="flex items-center gap-2">
        <Wand2
          size={12}
          className={cn("shrink-0", proposed ? "text-warning" : "text-accent")}
        />
        <span className="min-w-0 flex-1 truncate text-[13px] font-medium">
          {skill.name || "(unnamed)"}
        </span>
        {proposed && (
          <span className="shrink-0 rounded bg-warning/15 px-1.5 py-0.5 text-[10px] font-medium text-warning">
            proposal
          </span>
        )}
        <button
          onClick={(e) => {
            e.stopPropagation();
            onEdit();
          }}
          className="shrink-0 rounded p-0.5 text-text-dim opacity-0 hover:text-text group-hover:opacity-100"
          title="Edit"
        >
          <Pencil size={12} />
        </button>
        <button
          onClick={(e) => {
            e.stopPropagation();
            onDelete();
          }}
          className="shrink-0 rounded p-0.5 text-text-dim opacity-0 hover:text-danger group-hover:opacity-100"
          title="Delete"
        >
          <Trash2 size={12} />
        </button>
      </div>
      {skill.description && (
        <div className="mt-0.5 truncate text-[11px] text-text-dim">
          {skill.description}
        </div>
      )}
      <div className="mt-0.5 text-[10px] text-text-muted">
        {skill.steps.length} step{skill.steps.length === 1 ? "" : "s"} ·{" "}
        {targetLabel(skill.targets)}
        {skill.createdBy === "ai" && " · AI-authored"}
      </div>
    </div>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div>
      <div className="mb-1 text-[10px] font-semibold uppercase tracking-wide text-text-dim">
        {title}
      </div>
      {children}
    </div>
  );
}

function SkillDetail({
  skill,
  onApprove,
  onEdit,
}: {
  skill: Skill;
  onApprove: () => void;
  onEdit: () => void;
}) {
  const run = useSkills((s) => s.run);
  const targetable = useTargetable();

  const [paramValues, setParamValues] = useState<Record<string, string>>(() => {
    const init: Record<string, string> = {};
    for (const p of skill.params) init[p.name] = p.default ?? "";
    return init;
  });
  const [targetMode, setTargetMode] = useState<"default" | "all" | "pick">(
    "default"
  );
  const [picked, setPicked] = useState<string[]>([]);
  const [busy, setBusy] = useState<"" | "dry" | "run">("");
  const [dry, setDry] = useState<SkillDryRunResult | null>(null);
  const [result, setResult] = useState<SkillRunResult | null>(null);

  const proposed = skill.status === "proposed";
  const missingRequired = skill.params.filter(
    (p) => p.required && !(paramValues[p.name] ?? "").trim()
  );

  const overrideTargets = (): string[] | null => {
    if (targetMode === "all") return ["all"];
    if (targetMode === "pick") return picked;
    return null;
  };

  const doRun = async (dryRun: boolean) => {
    if (missingRequired.length) {
      toast.error(
        "Missing parameters",
        `Fill in: ${missingRequired.map((p) => p.name).join(", ")}`
      );
      return;
    }
    if (targetMode === "pick" && picked.length === 0) {
      toast.error("No targets picked", "Choose at least one server, or use Default / All.");
      return;
    }
    setBusy(dryRun ? "dry" : "run");
    setDry(null);
    setResult(null);
    const res = await run(skill.name, paramValues, overrideTargets(), dryRun);
    setBusy("");
    if (!res) return;
    if ("dryRun" in res && res.dryRun) setDry(res);
    else setResult(res as SkillRunResult);
  };

  return (
    <div className="space-y-4 p-4">
      <div>
        <div className="flex items-center gap-2">
          <h2 className="text-base font-semibold">{skill.name}</h2>
          {proposed && (
            <span className="rounded bg-warning/15 px-1.5 py-0.5 text-[10px] font-medium text-warning">
              proposal
            </span>
          )}
          <button
            onClick={onEdit}
            className="ml-auto flex items-center gap-1 rounded-md border border-border px-2 py-1 text-xs hover:bg-bg-subtle"
          >
            <Pencil size={12} /> Edit
          </button>
        </div>
        {skill.description && (
          <p className="mt-1 text-sm text-text-muted">{skill.description}</p>
        )}
      </div>

      {proposed && (
        <div className="flex items-start gap-2 rounded-lg border border-warning/40 bg-warning/10 p-3 text-xs">
          <ShieldAlert size={16} className="mt-0.5 shrink-0 text-warning" />
          <div className="flex-1">
            <div className="font-medium text-warning">
              AI-authored proposal — not runnable yet
            </div>
            <div className="mt-0.5 text-text-muted">
              Review the steps below. It won't run until you approve it; you can
              Dry-run to preview exactly what it would do first.
            </div>
          </div>
          <button
            onClick={onApprove}
            className="flex shrink-0 items-center gap-1 rounded-md bg-warning px-2 py-1 text-[11px] font-medium text-black hover:brightness-110"
          >
            <Check size={12} /> Approve
          </button>
        </div>
      )}

      <Section title="Steps">
        <ol className="space-y-1">
          {skill.steps.map((st, i) => (
            <li
              key={i}
              className="rounded-md bg-bg-subtle px-2 py-1.5 font-mono text-[11px]"
            >
              <span className="mr-2 text-text-dim">{i + 1}.</span>
              {st.name && (
                <span className="mr-2 font-sans text-[10px] text-accent">
                  {st.name}
                </span>
              )}
              {st.command}
            </li>
          ))}
        </ol>
      </Section>

      {skill.params.length > 0 && (
        <Section title="Parameters">
          <div className="space-y-2">
            {skill.params.map((p) => (
              <div key={p.name}>
                <label className="text-[11px] text-text-muted">
                  {p.name}
                  {p.required && <span className="text-danger"> *</span>}
                  {p.description && (
                    <span className="ml-1 text-text-dim">— {p.description}</span>
                  )}
                </label>
                <input
                  value={paramValues[p.name] ?? ""}
                  onChange={(e) =>
                    setParamValues((v) => ({ ...v, [p.name]: e.target.value }))
                  }
                  placeholder={p.default ?? ""}
                  className="mt-0.5 w-full rounded-md border border-border bg-bg px-2 py-1 text-xs"
                />
              </div>
            ))}
          </div>
        </Section>
      )}

      <Section title="Run on">
        <div className="flex w-fit overflow-hidden rounded-md border border-border text-[11px]">
          {(["default", "all", "pick"] as const).map((m) => (
            <button
              key={m}
              onClick={() => setTargetMode(m)}
              className={cn(
                "px-2.5 py-1",
                targetMode === m ? "bg-accent text-white" : "hover:bg-bg-subtle"
              )}
            >
              {m === "default"
                ? `Default (${targetLabel(skill.targets)})`
                : m === "all"
                  ? "All servers"
                  : "Pick"}
            </button>
          ))}
        </div>
        {targetMode === "pick" && (
          <div className="mt-2 space-y-1">
            {targetable.length === 0 ? (
              <div className="text-[11px] text-text-dim">
                No exec-capable connections have granted agent access. Enable one
                in the Agent Bridge panel.
              </div>
            ) : (
              targetable.map((t) => (
                <label
                  key={t.name}
                  className="flex items-center gap-2 text-xs"
                >
                  <input
                    type="checkbox"
                    checked={picked.includes(t.name)}
                    onChange={(e) =>
                      setPicked((cur) =>
                        e.target.checked
                          ? [...cur, t.name]
                          : cur.filter((n) => n !== t.name)
                      )
                    }
                  />
                  <Server size={12} className="text-text-dim" />
                  {t.name}
                  <span className="text-[10px] text-text-dim">
                    {t.protocol === "ghostftp-agent" ? "agent" : "ssh"}
                  </span>
                </label>
              ))
            )}
          </div>
        )}
      </Section>

      <div className="flex items-center gap-2">
        <button
          onClick={() => doRun(true)}
          disabled={busy !== ""}
          className="flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-xs hover:bg-bg-subtle disabled:opacity-50"
        >
          {busy === "dry" ? (
            <Loader2 size={13} className="animate-spin" />
          ) : (
            <Eye size={13} />
          )}
          Dry-run
        </button>
        <button
          onClick={() => doRun(false)}
          disabled={busy !== "" || proposed}
          title={proposed ? "Approve the proposal first" : undefined}
          className="flex items-center gap-1 rounded-md bg-accent px-3 py-1.5 text-xs font-medium text-white hover:brightness-110 disabled:opacity-50"
        >
          {busy === "run" ? (
            <Loader2 size={13} className="animate-spin" />
          ) : (
            <Play size={13} />
          )}
          Run
        </button>
      </div>

      {dry && <DryRunResult dry={dry} />}
      {result && <RunResult result={result} />}
    </div>
  );
}

function SkippedList({
  skipped,
}: {
  skipped: { target: string; reason: string }[];
}) {
  if (!skipped || skipped.length === 0) return null;
  return (
    <div className="mt-2 space-y-0.5">
      {skipped.map((s, i) => (
        <div key={i} className="text-[11px] text-warning">
          ⚠ skipped {s.target}: {s.reason}
        </div>
      ))}
    </div>
  );
}

function DryRunResult({ dry }: { dry: SkillDryRunResult }) {
  return (
    <Section title="Dry run — resolved commands (nothing ran)">
      {dry.targets.length === 0 && (
        <div className="text-[11px] text-warning">No runnable targets.</div>
      )}
      <div className="space-y-2">
        {dry.targets.map((t) => (
          <div key={t.sessionId} className="rounded-md border border-border">
            <div className="flex items-center gap-1.5 border-b border-border bg-bg-subtle px-2 py-1 text-[11px] font-medium">
              <Server size={12} className="text-text-dim" /> {t.sessionName}
            </div>
            <ol className="space-y-1 p-2">
              {t.commands.map((c, i) => (
                <li key={i} className="font-mono text-[11px]">
                  <span className="mr-2 text-text-dim">{i + 1}.</span>
                  {c}
                </li>
              ))}
            </ol>
          </div>
        ))}
      </div>
      <SkippedList skipped={dry.skipped} />
      {dry.needsApproval && (
        <div className="mt-2 text-[11px] text-text-dim">
          A real run will ask you to approve the whole fleet run once.
        </div>
      )}
    </Section>
  );
}

function RunResult({ result }: { result: SkillRunResult }) {
  return (
    <Section title={`Result — ${result.succeeded} ok, ${result.failed} failed`}>
      <div className="space-y-2">
        {result.results.map((t) => (
          <div key={t.sessionId} className="rounded-md border border-border">
            <div
              className={cn(
                "flex items-center gap-1.5 border-b border-border px-2 py-1 text-[11px] font-medium",
                t.ok ? "bg-success/10" : "bg-danger/10"
              )}
            >
              {t.ok ? (
                <CircleCheck size={12} className="text-success" />
              ) : (
                <CircleX size={12} className="text-danger" />
              )}
              {t.sessionName}
            </div>
            <div className="space-y-1.5 p-2">
              {t.steps.map((st, i) => (
                <div key={i}>
                  <div className="flex items-center gap-1.5 text-[11px]">
                    {st.ok ? (
                      <CircleCheck size={11} className="shrink-0 text-success" />
                    ) : (
                      <CircleX size={11} className="shrink-0 text-danger" />
                    )}
                    <span className="min-w-0 truncate font-mono">
                      {st.command}
                    </span>
                    {st.exitCode != null && (
                      <span className="shrink-0 text-text-dim">
                        exit {st.exitCode}
                      </span>
                    )}
                    {st.timedOut && (
                      <span className="shrink-0 text-warning">timed out</span>
                    )}
                  </div>
                  {st.error && (
                    <pre className="mt-0.5 whitespace-pre-wrap rounded bg-danger/10 p-1.5 text-[10px] text-danger">
                      {st.error}
                    </pre>
                  )}
                  {st.stdout && (
                    <pre className="mt-0.5 max-h-40 overflow-auto whitespace-pre-wrap rounded bg-bg-subtle p-1.5 font-mono text-[10px]">
                      {st.stdout}
                    </pre>
                  )}
                  {st.stderr && (
                    <pre className="mt-0.5 max-h-40 overflow-auto whitespace-pre-wrap rounded bg-bg-subtle p-1.5 font-mono text-[10px] text-warning">
                      {st.stderr}
                    </pre>
                  )}
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
      <SkippedList skipped={result.skipped} />
    </Section>
  );
}

function SkillEditor({
  skill,
  onClose,
}: {
  skill: Skill | null;
  onClose: () => void;
}) {
  const saveSkill = useSkills((s) => s.saveSkill);
  const targetable = useTargetable();
  const panelRef = useRef<HTMLDivElement>(null);
  useDialog(panelRef, { onClose });

  const [draft, setDraft] = useState<Skill>(() =>
    skill ? structuredClone(skill) : emptySkill()
  );

  const set = (patch: Partial<Skill>) => setDraft((d) => ({ ...d, ...patch }));

  const updateStep = (i: number, patch: Partial<SkillStep>) =>
    setDraft((d) => ({
      ...d,
      steps: d.steps.map((s, j) => (j === i ? { ...s, ...patch } : s)),
    }));
  const addStep = () =>
    setDraft((d) => ({ ...d, steps: [...d.steps, { name: "", command: "" }] }));
  const removeStep = (i: number) =>
    setDraft((d) => ({ ...d, steps: d.steps.filter((_, j) => j !== i) }));

  const updateParam = (i: number, patch: Partial<SkillParam>) =>
    setDraft((d) => ({
      ...d,
      params: d.params.map((p, j) => (j === i ? { ...p, ...patch } : p)),
    }));
  const addParam = () =>
    setDraft((d) => ({
      ...d,
      params: [
        ...d.params,
        { name: "", description: "", required: false, default: null },
      ],
    }));
  const removeParam = (i: number) =>
    setDraft((d) => ({ ...d, params: d.params.filter((_, j) => j !== i) }));

  const addTargetName = (name: string) =>
    setDraft((d) =>
      d.targets.sessions.includes(name)
        ? d
        : { ...d, targets: { ...d.targets, sessions: [...d.targets.sessions, name] } }
    );

  const canSave =
    draft.name.trim() !== "" && draft.steps.some((s) => s.command.trim() !== "");

  const submit = () => {
    if (!canSave) return;
    const cleaned: Skill = {
      ...draft,
      name: draft.name.trim(),
      description: draft.description.trim(),
      steps: draft.steps
        .filter((s) => s.command.trim() !== "")
        .map((s) => ({ name: s.name.trim(), command: s.command })),
      params: draft.params
        .filter((p) => p.name.trim() !== "")
        .map((p) => ({ ...p, name: p.name.trim() })),
    };
    saveSkill(cleaned);
    onClose();
  };

  return (
    <div className="fixed inset-0 z-palette flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        className="anim-modal flex max-h-[88vh] w-[42rem] max-w-[94vw] flex-col rounded-xl border border-border bg-bg-panel shadow-elev-3"
      >
        <div className="flex items-center gap-2 border-b border-border px-4 py-3">
          <Wand2 size={15} className="text-accent" />
          <span className="text-sm font-semibold">
            {skill ? "Edit skill" : "New skill"}
          </span>
          <button
            onClick={onClose}
            className="ml-auto rounded-md p-1 hover:bg-bg-subtle"
          >
            <X size={15} />
          </button>
        </div>

        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-4">
          <div>
            <label className="text-[11px] text-text-muted">Name</label>
            <input
              value={draft.name}
              onChange={(e) => set({ name: e.target.value })}
              placeholder="e.g. restart-web"
              className="mt-0.5 w-full rounded-md border border-border bg-bg px-2 py-1 text-sm"
            />
          </div>
          <div>
            <label className="text-[11px] text-text-muted">Description</label>
            <input
              value={draft.description}
              onChange={(e) => set({ description: e.target.value })}
              placeholder="What it does and when to use it"
              className="mt-0.5 w-full rounded-md border border-border bg-bg px-2 py-1 text-sm"
            />
          </div>

          <Section title="Steps (run in order on each target)">
            {draft.steps.map((st, i) => (
              <div key={i} className="mb-2 rounded-md border border-border p-2">
                <div className="mb-1 flex items-center gap-2">
                  <span className="text-[10px] text-text-dim">Step {i + 1}</span>
                  <input
                    value={st.name}
                    placeholder="label (optional)"
                    onChange={(e) => updateStep(i, { name: e.target.value })}
                    className="flex-1 rounded border border-border bg-bg px-1.5 py-0.5 text-[11px]"
                  />
                  <button
                    onClick={() => removeStep(i)}
                    disabled={draft.steps.length === 1}
                    className="rounded p-0.5 text-text-dim hover:text-danger disabled:opacity-30"
                    title="Remove step"
                  >
                    <Trash2 size={12} />
                  </button>
                </div>
                <textarea
                  value={st.command}
                  placeholder="shell command — use ${param} placeholders"
                  onChange={(e) => updateStep(i, { command: e.target.value })}
                  rows={2}
                  className="w-full resize-y rounded border border-border bg-bg px-2 py-1 font-mono text-[11px]"
                />
              </div>
            ))}
            <button
              onClick={addStep}
              className="flex items-center gap-1 text-[11px] text-accent hover:underline"
            >
              <Plus size={12} /> Add step
            </button>
          </Section>

          <Section title="Parameters (interpolated as ${name})">
            {draft.params.length === 0 && (
              <div className="mb-1 text-[11px] text-text-dim">
                No parameters. Add one if a step needs a value at run time.
              </div>
            )}
            {draft.params.map((p, i) => (
              <div key={i} className="mb-2 rounded-md border border-border p-2">
                <div className="flex items-center gap-2">
                  <input
                    value={p.name}
                    placeholder="name"
                    onChange={(e) => updateParam(i, { name: e.target.value })}
                    className="w-32 rounded border border-border bg-bg px-1.5 py-0.5 text-[11px]"
                  />
                  <input
                    value={p.description}
                    placeholder="description (optional)"
                    onChange={(e) =>
                      updateParam(i, { description: e.target.value })
                    }
                    className="flex-1 rounded border border-border bg-bg px-1.5 py-0.5 text-[11px]"
                  />
                  <button
                    onClick={() => removeParam(i)}
                    className="rounded p-0.5 text-text-dim hover:text-danger"
                    title="Remove parameter"
                  >
                    <Trash2 size={12} />
                  </button>
                </div>
                <div className="mt-1 flex items-center gap-3">
                  <label className="flex items-center gap-1 text-[11px] text-text-muted">
                    <input
                      type="checkbox"
                      checked={p.required}
                      onChange={(e) =>
                        updateParam(i, { required: e.target.checked })
                      }
                    />
                    required
                  </label>
                  <input
                    value={p.default ?? ""}
                    placeholder="default value (optional)"
                    onChange={(e) =>
                      updateParam(i, {
                        default: e.target.value === "" ? null : e.target.value,
                      })
                    }
                    className="flex-1 rounded border border-border bg-bg px-1.5 py-0.5 text-[11px]"
                  />
                </div>
              </div>
            ))}
            <button
              onClick={addParam}
              className="flex items-center gap-1 text-[11px] text-accent hover:underline"
            >
              <Plus size={12} /> Add parameter
            </button>
          </Section>

          <Section title="Default targets">
            <label className="flex items-center gap-2 text-xs">
              <input
                type="checkbox"
                checked={draft.targets.all}
                onChange={(e) =>
                  set({ targets: { ...draft.targets, all: e.target.checked } })
                }
              />
              Run on all exec-capable servers
            </label>
            {!draft.targets.all && (
              <div className="mt-2">
                <input
                  value={draft.targets.sessions.join(", ")}
                  onChange={(e) =>
                    set({
                      targets: {
                        ...draft.targets,
                        sessions: splitNames(e.target.value),
                      },
                    })
                  }
                  placeholder="server names, comma-separated"
                  className="w-full rounded-md border border-border bg-bg px-2 py-1 text-xs"
                />
                {targetable.length > 0 && (
                  <div className="mt-1 flex flex-wrap gap-1">
                    {targetable.map((t) => (
                      <button
                        key={t.name}
                        onClick={() => addTargetName(t.name)}
                        className="rounded border border-border px-1.5 py-0.5 text-[10px] hover:bg-bg-subtle"
                      >
                        + {t.name}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}
          </Section>

          <label className="flex items-center gap-2 text-xs">
            <input
              type="checkbox"
              checked={draft.stopOnError}
              onChange={(e) => set({ stopOnError: e.target.checked })}
            />
            Stop a server's remaining steps after the first failing step
          </label>
        </div>

        <div className="flex items-center justify-end gap-2 border-t border-border px-4 py-3">
          <button
            onClick={onClose}
            className="rounded-md border border-border px-3 py-1.5 text-xs hover:bg-bg-subtle"
          >
            Cancel
          </button>
          <button
            onClick={submit}
            disabled={!canSave}
            className="rounded-md bg-accent px-3 py-1.5 text-xs font-medium text-white hover:brightness-110 disabled:opacity-50"
          >
            Save
          </button>
        </div>
      </div>
    </div>
  );
}
