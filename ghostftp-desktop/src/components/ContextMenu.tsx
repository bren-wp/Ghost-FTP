import { useEffect, useRef } from "react";
import { cn } from "@/lib/cn";

export interface MenuItem {
  label: string;
  onClick: () => void;
  icon?: React.ReactNode;
  disabled?: boolean;
  destructive?: boolean;
  separatorAfter?: boolean;
}

interface Props {
  x: number;
  y: number;
  items: MenuItem[];
  onClose: () => void;
}

export function ContextMenu({ x, y, items, onClose }: Props) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onAnyClick = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) onClose();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("mousedown", onAnyClick);
    window.addEventListener("keydown", onKey);
    // Move focus into the menu so it's keyboard-operable straight away.
    const raf = requestAnimationFrame(() =>
      ref.current
        ?.querySelector<HTMLButtonElement>("button:not([disabled])")
        ?.focus()
    );
    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener("mousedown", onAnyClick);
      window.removeEventListener("keydown", onKey);
    };
  }, [onClose]);

  const onMenuKeyDown = (e: React.KeyboardEvent) => {
    if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(e.key)) return;
    e.preventDefault();
    const btns = Array.from(
      ref.current?.querySelectorAll<HTMLButtonElement>(
        "button:not([disabled])"
      ) ?? []
    );
    if (btns.length === 0) return;
    const idx = btns.indexOf(document.activeElement as HTMLButtonElement);
    const next =
      e.key === "Home"
        ? 0
        : e.key === "End"
          ? btns.length - 1
          : e.key === "ArrowDown"
            ? (idx + 1) % btns.length
            : (idx - 1 + btns.length) % btns.length;
    btns[next]?.focus();
  };

  // Clamp to viewport.
  const maxX = window.innerWidth - 220;
  const maxY = window.innerHeight - items.length * 28 - 16;
  const left = Math.min(x, maxX);
  const top = Math.min(y, maxY);

  return (
    <div
      ref={ref}
      role="menu"
      aria-label="Context menu"
      onKeyDown={onMenuKeyDown}
      style={{ left, top }}
      className="anim-modal fixed z-menu min-w-[200px] rounded-lg border border-border bg-bg-panel py-1 shadow-elev-3"
    >
      {items.map((item, i) => (
        <div key={i}>
          <button
            role="menuitem"
            disabled={item.disabled}
            onClick={() => {
              item.onClick();
              onClose();
            }}
            className={cn(
              "flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm hover:bg-bg-hover disabled:opacity-40 disabled:hover:bg-transparent",
              item.destructive && "text-danger hover:bg-danger-soft"
            )}
          >
            {item.icon && (
              <span className="flex h-3.5 w-3.5 items-center justify-center text-text-muted">
                {item.icon}
              </span>
            )}
            <span className="flex-1">{item.label}</span>
          </button>
          {item.separatorAfter && (
            <div className="my-1 border-t border-border" />
          )}
        </div>
      ))}
    </div>
  );
}
