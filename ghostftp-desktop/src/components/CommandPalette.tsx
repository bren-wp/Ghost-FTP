import { useEffect, useId, useMemo, useRef, useState, type ReactNode } from "react";
import { Search } from "lucide-react";
import { useLayout } from "@/stores/layoutStore";
import { type Command } from "@/lib/commands";
import { useResolvedCommands } from "@/lib/keybindings";
import { useDialog } from "@/hooks/useDialog";
import { fuzzyMatch } from "@/lib/fuzzy";
import { formatCombo } from "@/lib/shortcuts";
import { cn } from "@/lib/cn";

interface Ranked {
  c: Command;
  score: number;
  ranges: [number, number][];
}

export function CommandPalette() {
  const open = useLayout((s) => s.paletteOpen);
  const setOpen = useLayout((s) => s.setPaletteOpen);
  const commands = useResolvedCommands();
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const listId = useId();
  const optionId = (i: number) => `${listId}-opt-${i}`;
  useDialog(panelRef, {
    onClose: () => setOpen(false),
    enabled: open,
    initialFocus: inputRef,
  });

  useEffect(() => {
    if (open) {
      setQuery("");
      setActive(0);
      requestAnimationFrame(() => inputRef.current?.focus());
    }
  }, [open]);

  const results = useMemo<Ranked[]>(() => {
    const out: Ranked[] = [];
    for (const c of commands) {
      if (c.enabled === false) continue;
      const tm = fuzzyMatch(query, c.title);
      if (tm.matched) {
        out.push({ c, score: tm.score + 100, ranges: tm.ranges });
        continue;
      }
      if (c.keywords) {
        const km = fuzzyMatch(query, c.keywords);
        if (km.matched) out.push({ c, score: km.score, ranges: [] });
      }
    }
    out.sort((a, b) => b.score - a.score);
    return out;
  }, [commands, query]);

  useEffect(() => {
    setActive((a) => Math.min(a, Math.max(0, results.length - 1)));
  }, [results.length]);

  useEffect(() => {
    listRef.current
      ?.querySelector(`[data-idx="${active}"]`)
      ?.scrollIntoView({ block: "nearest" });
  }, [active]);

  if (!open) return null;

  const run = (c: Command) => {
    setOpen(false);
    setTimeout(() => c.run(), 0);
  };

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      setActive((a) => (results.length ? (a + 1) % results.length : 0));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      setActive((a) =>
        results.length ? (a - 1 + results.length) % results.length : 0
      );
    } else if (e.key === "Enter") {
      e.preventDefault();
      const r = results[active];
      if (r) run(r.c);
    } else if (e.key === "Escape") {
      e.preventDefault();
      setOpen(false);
    }
  };

  return (
    <div
      className="fixed inset-0 z-palette flex items-start justify-center bg-black/50 pt-[12vh] backdrop-blur-sm"
      onClick={() => setOpen(false)}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-label="Command palette"
        onClick={(e) => e.stopPropagation()}
        className="anim-modal flex max-h-[70vh] w-[34rem] max-w-[92vw] flex-col overflow-hidden rounded-xl border border-border bg-bg-panel shadow-elev-3"
      >
        <div className="flex items-center gap-2 border-b border-border px-3.5 py-2.5">
          <Search size={15} className="shrink-0 text-text-dim" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setActive(0);
            }}
            onKeyDown={onKeyDown}
            placeholder="Type a command or search…"
            role="combobox"
            aria-expanded={results.length > 0}
            aria-controls={listId}
            aria-activedescendant={
              results.length ? optionId(active) : undefined
            }
            aria-label="Search commands"
            className="w-full bg-transparent text-sm outline-none placeholder:text-text-dim"
          />
        </div>
        <div
          ref={listRef}
          id={listId}
          role="listbox"
          aria-label="Commands"
          className="overflow-y-auto py-1"
        >
          {results.length === 0 ? (
            <div className="px-4 py-8 text-center text-xs text-text-dim">
              No matching commands
            </div>
          ) : (
            results.map((r, i) => (
              <button
                key={r.c.id}
                data-idx={i}
                role="option"
                id={optionId(i)}
                aria-selected={i === active}
                tabIndex={-1}
                onMouseMove={() => setActive(i)}
                onClick={() => run(r.c)}
                className={cn(
                  "flex w-full items-center gap-2.5 px-3.5 py-2 text-left text-sm",
                  i === active ? "bg-accent/15 text-text" : "text-text-muted"
                )}
              >
                <span
                  className={cn(
                    "flex h-4 w-4 shrink-0 items-center justify-center",
                    i === active ? "text-accent" : "text-text-dim"
                  )}
                >
                  {r.c.icon}
                </span>
                <span className="min-w-0 flex-1 truncate">
                  <Highlight text={r.c.title} ranges={r.ranges} />
                </span>
                <span className="shrink-0 text-[10px] uppercase tracking-wider text-text-dim">
                  {r.c.group}
                </span>
                {r.c.combo && (
                  <span className="shrink-0 rounded border border-border bg-bg-subtle px-1.5 py-0.5 font-mono text-[10px] text-text-dim">
                    {formatCombo(r.c.combo)}
                  </span>
                )}
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

function Highlight({
  text,
  ranges,
}: {
  text: string;
  ranges: [number, number][];
}) {
  if (!ranges.length) return <>{text}</>;
  const out: ReactNode[] = [];
  let i = 0;
  ranges.forEach(([s, e], k) => {
    if (s > i) out.push(<span key={`n${k}`}>{text.slice(i, s)}</span>);
    out.push(
      <span key={`h${k}`} className="font-semibold text-accent">
        {text.slice(s, e)}
      </span>
    );
    i = e;
  });
  if (i < text.length) out.push(<span key="end">{text.slice(i)}</span>);
  return <>{out}</>;
}
