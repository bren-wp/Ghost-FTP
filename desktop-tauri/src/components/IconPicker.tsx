// Preset icon picker for a connection's rail bubble (profile.icon). Click the
// preview to open a searchable grid of curated emojis + every bundled Iconify
// glyph — nobody should have to memorise `mdi:rocket-launch`. The stored value
// stays a plain string (emoji char or "prefix:name"), so the rail, the mock
// layer, and the backend need no changes.
import { useEffect, useRef, useState } from "react";
import { ChevronDown, Search, X } from "lucide-react";
import { BrandIcon } from "@/lib/brandIcons";
import { BRAND_ICONS } from "@/lib/brandIconData";
import { cn } from "@/lib/cn";

const EMOJI_CHOICES: { char: string; label: string }[] = [
  { char: "🚀", label: "rocket" },
  { char: "🛰️", label: "satellite" },
  { char: "🌐", label: "globe" },
  { char: "🔥", label: "fire" },
  { char: "⚡", label: "lightning" },
  { char: "⭐", label: "star" },
  { char: "❤️", label: "heart" },
  { char: "🛡️", label: "shield" },
  { char: "🏠", label: "home" },
  { char: "🏢", label: "office" },
  { char: "☁️", label: "cloud" },
  { char: "🗄️", label: "database" },
  { char: "💾", label: "floppy disk" },
  { char: "📦", label: "package" },
  { char: "🔧", label: "wrench" },
  { char: "🐱", label: "cat" },
  { char: "🐶", label: "dog" },
  { char: "🤖", label: "robot" },
  { char: "👽", label: "alien" },
  { char: "👻", label: "ghost" },
  { char: "🥷", label: "ninja" },
  { char: "🍕", label: "pizza" },
  { char: "☕", label: "coffee" },
  { char: "🌵", label: "cactus" },
  { char: "🐧", label: "penguin" },
  { char: "🏰", label: "castle" },
  { char: "⛵", label: "sailboat" },
  { char: "✈️", label: "airplane" },
];

const POPOVER_W = 320; // w-80
const POPOVER_MAX_H = 320; // max-h-80

export function IconPicker({
  value,
  onChange,
  fallback,
}: {
  value: string;
  onChange: (v: string) => void;
  /** What the preview shows when nothing is picked (the name monogram). */
  fallback: React.ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null);
  const btnRef = useRef<HTMLButtonElement>(null);

  const trimmed = value.trim();
  const valueIsKnownKey = trimmed.includes(":") && !!BRAND_ICONS[trimmed];

  const openPicker = () => {
    const r = btnRef.current?.getBoundingClientRect();
    if (r) {
      setPos({
        top: Math.max(8, Math.min(r.bottom + 6, window.innerHeight - POPOVER_MAX_H - 8)),
        left: Math.max(8, Math.min(r.left, window.innerWidth - POPOVER_W - 8)),
      });
    }
    setQuery("");
    setOpen(true);
  };

  // Escape closes; overlay-click covers the mouse case.
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);

  const q = query.trim().toLowerCase();
  const emoji = EMOJI_CHOICES.filter((e) => !q || e.label.includes(q));
  const keys = Object.keys(BRAND_ICONS).filter((k) => !q || k.includes(q));

  const pick = (v: string) => {
    onChange(v);
    setOpen(false);
  };

  const cellCls = (active: boolean) =>
    cn(
      "flex h-8 w-8 items-center justify-center rounded-md text-base transition-colors",
      active ? "bg-accent-soft ring-1 ring-accent/50" : "hover:bg-bg-hover"
    );

  return (
    <>
      <button
        ref={btnRef}
        type="button"
        onClick={openPicker}
        title="Pick a bubble icon"
        className="flex h-[34px] w-full items-center gap-1.5 rounded-md border border-border bg-bg-subtle px-2 text-sm hover:border-accent/60"
      >
        <span className="flex h-6 w-6 items-center justify-center rounded-md bg-bg text-[13px] font-bold leading-none">
          {trimmed && valueIsKnownKey ? (
            <BrandIcon icon={trimmed} size={15} />
          ) : trimmed && !trimmed.includes(":") ? (
            trimmed
          ) : (
            <span className="text-text-dim">{fallback}</span>
          )}
        </span>
        <span className="flex-1 truncate text-left text-[11px] text-text-muted">
          {trimmed || "Default (monogram)"}
        </span>
        <ChevronDown size={13} className="shrink-0 text-text-dim" />
      </button>

      {open && pos && (
        <div className="fixed inset-0 z-palette" onClick={() => setOpen(false)}>
          <div
            role="dialog"
            aria-label="Pick a bubble icon"
            onClick={(e) => e.stopPropagation()}
            style={{ top: pos.top, left: pos.left }}
            className="anim-modal fixed flex max-h-80 w-80 flex-col rounded-xl border border-border bg-bg-panel shadow-elev-3"
          >
            <div className="flex items-center gap-1.5 border-b border-border px-2.5 py-2">
              <Search size={13} className="shrink-0 text-text-dim" />
              <input
                autoFocus
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && q) pick(q);
                }}
                placeholder="Search icons…"
                className="w-full bg-transparent text-xs outline-none placeholder:text-text-dim"
              />
              <button
                type="button"
                onClick={() => setOpen(false)}
                aria-label="Close"
                className="rounded p-0.5 text-text-muted hover:bg-bg-hover hover:text-text"
              >
                <X size={13} />
              </button>
            </div>

            <div className="flex-1 overflow-y-auto p-2">
              <button
                type="button"
                onClick={() => pick("")}
                className={cn(
                  "mb-1 flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs transition-colors",
                  !trimmed ? "bg-accent-soft" : "hover:bg-bg-hover"
                )}
              >
                <X size={12} className="text-text-dim" />
                Default (name monogram)
              </button>

              {emoji.length > 0 && (
                <>
                  <div className="px-1 pb-1 pt-1.5 text-[10px] font-semibold uppercase tracking-wider text-text-dim">
                    Emoji
                  </div>
                  <div className="grid grid-cols-8 gap-0.5">
                    {emoji.map((e) => (
                      <button
                        key={e.char}
                        type="button"
                        title={e.label}
                        onClick={() => pick(e.char)}
                        className={cellCls(trimmed === e.char)}
                      >
                        {e.char}
                      </button>
                    ))}
                  </div>
                </>
              )}

              {keys.length > 0 && (
                <>
                  <div className="px-1 pb-1 pt-2 text-[10px] font-semibold uppercase tracking-wider text-text-dim">
                    Icons
                  </div>
                  <div className="grid grid-cols-8 gap-0.5">
                    {keys.map((k) => (
                      <button
                        key={k}
                        type="button"
                        title={k}
                        onClick={() => pick(k)}
                        className={cellCls(trimmed === k)}
                      >
                        <BrandIcon icon={k} size={16} />
                      </button>
                    ))}
                  </div>
                </>
              )}

              {q && (
                <button
                  type="button"
                  onClick={() => pick(q)}
                  className="mt-1.5 flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-text-muted transition-colors hover:bg-bg-hover"
                >
                  Use “{query.trim()}” as a custom icon
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </>
  );
}
