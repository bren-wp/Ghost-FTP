import { useEffect, useMemo, useState } from "react";
import { Search, RotateCcw, Circle, X } from "lucide-react";
import { useCommands } from "@/lib/commands";
import { useBindings } from "@/stores/bindingsStore";
import { effectiveCombo } from "@/lib/keybindings";
import { FILE_BROWSER_KEYS, FILE_BROWSER_GROUP } from "@/lib/fileBrowserKeys";
import { keyCombo, formatCombo } from "@/lib/shortcuts";
import { cn } from "@/lib/cn";

// Settings → Keyboard (Plan 15 Phase 2). A searchable, grouped list of every
// remappable binding — the app commands that carry a default combo plus the
// file-browser action catalog — each with a click-to-record capture field,
// conflict detection, per-row reset, and reset-all. Overrides live in the shared
// bindings store (persisted as `shortcut.*` rows in ghostftp.db).

interface KeyEntry {
  id: string;
  title: string;
  group: string;
  defaultCombo: string;
  /** Conflict scope: command shortcuts (global + terminal, they share the
   *  window) never collide with the pane-local file-browser keys. */
  scope: "cmd" | "fb";
}

const MOD_KEYS = new Set(["Control", "Shift", "Alt", "Meta", "OS"]);

export function KeyboardSettings() {
  const commands = useCommands();
  const overrides = useBindings((s) => s.overrides);
  const setOverride = useBindings((s) => s.setOverride);
  const clearOverride = useBindings((s) => s.clearOverride);
  const resetAll = useBindings((s) => s.resetAll);

  const [q, setQ] = useState("");
  const [recordingId, setRecordingId] = useState<string | null>(null);
  const [conflict, setConflict] = useState<{
    targetId: string;
    combo: string;
    otherTitle: string;
    otherId: string;
  } | null>(null);

  const entries = useMemo<KeyEntry[]>(() => {
    const list: KeyEntry[] = [];
    const seen = new Set<string>();
    for (const c of commands) {
      if (!c.combo && !(c.id in overrides)) continue;
      if (seen.has(c.id)) continue;
      seen.add(c.id);
      list.push({
        id: c.id,
        title: c.title,
        group: c.group,
        defaultCombo: c.combo ?? "",
        scope: "cmd",
      });
    }
    for (const k of FILE_BROWSER_KEYS) {
      list.push({
        id: k.id,
        title: k.title,
        group: FILE_BROWSER_GROUP,
        defaultCombo: k.defaultCombo,
        scope: "fb",
      });
    }
    return list;
  }, [commands, overrides]);

  const apply = (target: KeyEntry, combo: string) => {
    // Recording the built-in back = a reset, so we don't store a redundant row.
    if (combo === target.defaultCombo) clearOverride(target.id);
    else setOverride(target.id, combo);
  };

  // Capture the next keystroke while a row is recording. A window capture-phase
  // listener + stopPropagation keeps it from leaking to the global dispatcher or
  // the dialog's own Escape-to-close.
  useEffect(() => {
    if (!recordingId) return;
    const target = entries.find((e) => e.id === recordingId);
    const onKey = (e: KeyboardEvent) => {
      e.preventDefault();
      e.stopPropagation();
      if (MOD_KEYS.has(e.key)) return; // wait for a non-modifier key
      if (e.key === "Escape") {
        setRecordingId(null);
        return;
      }
      if (e.key === "Backspace") {
        // Clear → restore the default binding.
        clearOverride(recordingId);
        setRecordingId(null);
        return;
      }
      if (!target) {
        setRecordingId(null);
        return;
      }
      const combo = keyCombo(e);
      const cur = useBindings.getState().overrides;
      const other = entries.find(
        (x) =>
          x.id !== target.id &&
          x.scope === target.scope &&
          effectiveCombo(x.id, x.defaultCombo, cur) === combo
      );
      if (other) {
        setConflict({
          targetId: target.id,
          combo,
          otherTitle: other.title,
          otherId: other.id,
        });
        setRecordingId(null);
        return;
      }
      apply(target, combo);
      setRecordingId(null);
    };
    window.addEventListener("keydown", onKey, true);
    return () => window.removeEventListener("keydown", onKey, true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recordingId, entries]);

  const resolveConflict = () => {
    if (!conflict) return;
    const target = entries.find((e) => e.id === conflict.targetId);
    if (target) apply(target, conflict.combo);
    // Unbind the previous holder so one combo never drives two actions.
    setOverride(conflict.otherId, "");
    setConflict(null);
  };

  const query = q.trim().toLowerCase();
  const filtered = entries.filter(
    (e) =>
      query === "" ||
      e.title.toLowerCase().includes(query) ||
      e.group.toLowerCase().includes(query)
  );
  // Preserve the natural group order (registry order, then File browser).
  const groupOrder: string[] = [];
  const byGroup = new Map<string, KeyEntry[]>();
  for (const e of filtered) {
    if (!byGroup.has(e.group)) {
      byGroup.set(e.group, []);
      groupOrder.push(e.group);
    }
    byGroup.get(e.group)!.push(e);
  }

  const overrideCount = Object.keys(overrides).length;

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search
            size={13}
            className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-text-dim"
          />
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Search shortcuts…"
            className="w-full rounded-md border border-border bg-bg-subtle py-1.5 pl-8 pr-2.5 text-sm outline-none focus:border-accent"
          />
        </div>
        <button
          onClick={resetAll}
          disabled={overrideCount === 0}
          className="flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1.5 text-xs text-text-muted hover:bg-bg-hover hover:text-text disabled:opacity-40"
          title="Reset every shortcut to its default"
        >
          <RotateCcw size={12} /> Reset all
          {overrideCount > 0 ? ` (${overrideCount})` : ""}
        </button>
      </div>

      {conflict && (
        <div className="flex items-start gap-2 rounded-md border border-warning/40 bg-warning/10 px-3 py-2 text-xs">
          <div className="flex-1">
            <span className="font-mono font-medium">
              {formatCombo(conflict.combo)}
            </span>{" "}
            is already bound to{" "}
            <span className="font-medium">{conflict.otherTitle}</span>. Reassign
            it here and unbind the other?
          </div>
          <button
            onClick={resolveConflict}
            className="btn-accent shrink-0 rounded px-2 py-1 text-[11px] font-medium text-white"
          >
            Reassign
          </button>
          <button
            onClick={() => setConflict(null)}
            className="shrink-0 rounded p-1 text-text-muted hover:bg-bg-hover hover:text-text"
          >
            <X size={13} />
          </button>
        </div>
      )}

      {groupOrder.length === 0 ? (
        <div className="py-8 text-center text-xs text-text-dim">
          No shortcuts match.
        </div>
      ) : (
        groupOrder.map((group) => (
          <div key={group}>
            <div className="mb-1.5 mt-1 text-[11px] font-semibold uppercase tracking-[0.08em] text-text-muted">
              {group}
            </div>
            <div className="overflow-hidden rounded-md border border-border-subtle">
              {byGroup.get(group)!.map((e, i) => {
                const eff = effectiveCombo(e.id, e.defaultCombo, overrides);
                const overridden = e.id in overrides;
                const recording = recordingId === e.id;
                return (
                  <div
                    key={e.id}
                    className={cn(
                      "flex items-center gap-2 px-3 py-1.5 text-sm",
                      i > 0 && "border-t border-border-subtle"
                    )}
                  >
                    <span className="min-w-0 flex-1 truncate text-text-muted">
                      {e.title}
                    </span>
                    {overridden && (
                      <button
                        onClick={() => clearOverride(e.id)}
                        title="Reset to default"
                        className="rounded p-1 text-text-dim hover:bg-bg-hover hover:text-text"
                      >
                        <RotateCcw size={12} />
                      </button>
                    )}
                    <button
                      onClick={() =>
                        setRecordingId(recording ? null : e.id)
                      }
                      className={cn(
                        "min-w-[8rem] rounded border px-2 py-1 text-center font-mono text-[11px] transition-colors",
                        recording
                          ? "border-accent bg-accent/10 text-accent"
                          : "border-border bg-bg-subtle text-text-dim hover:border-accent/50 hover:text-text"
                      )}
                      title="Click, then press the key combination"
                    >
                      {recording ? (
                        <span className="inline-flex items-center gap-1.5">
                          <Circle size={7} className="animate-pulse fill-current" />
                          Press keys…
                        </span>
                      ) : eff ? (
                        formatCombo(eff)
                      ) : (
                        <span className="italic text-text-dim/70">Unbound</span>
                      )}
                    </button>
                  </div>
                );
              })}
            </div>
          </div>
        ))
      )}

      <p className="pt-1 text-[11px] leading-relaxed text-text-dim">
        点击快捷键后按下按键。<kbd>Esc</kbd> 取消，<kbd>Backspace</kbd> 恢复默认。文件浏览器快捷键仅在文件面板获得焦点时生效，输入文字时不会触发。
      </p>
    </div>
  );
}
