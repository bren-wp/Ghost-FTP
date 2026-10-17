import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { cn } from "@/lib/cn";
import { toastError } from "@/lib/errors";

export interface MenuItem {
  label: string;
  onClick?: () => void | Promise<void>;
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
  const [busyIndex, setBusyIndex] = useState<number | null>(null);
  // Synchronously lock actions so rapid double-clicks cannot dispatch two
  // destructive operations before React commits a disabled state.
  const busyRef = useRef(false);
  const [position, setPosition] = useState({ left: Math.max(8, x), top: Math.max(8, y) });
  const closeIfIdle = () => {
    if (!busyRef.current) onClose();
  };

  useLayoutEffect(() => {
    const menu = ref.current;
    if (!menu) return;

    const positionWithinViewport = () => {
      // Use measured dimensions, not a guessed row height: long labels,
      // separators, DPI changes and translated UI all affect the real size.
      const rect = menu.getBoundingClientRect();
      const left = Math.max(8, Math.min(x, window.innerWidth - rect.width - 8));
      const top = Math.max(8, Math.min(y, window.innerHeight - rect.height - 8));
      setPosition((current) =>
        current.left === left && current.top === top ? current : { left, top }
      );
    };

    positionWithinViewport();
    window.addEventListener("resize", positionWithinViewport);
    const observer = typeof ResizeObserver === "undefined"
      ? null
      : new ResizeObserver(positionWithinViewport);
    observer?.observe(menu);
    return () => {
      observer?.disconnect();
      window.removeEventListener("resize", positionWithinViewport);
    };
  }, [x, y, items.length]);

  useEffect(() => {
    const onAnyClick = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) closeIfIdle();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") closeIfIdle();
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

  const runItem = async (item: MenuItem, index: number) => {
    if (!item.onClick || item.disabled || busyRef.current) return;
    busyRef.current = true;
    setBusyIndex(index);
    try {
      await item.onClick();
      onClose();
    } catch (error) {
      toastError(error, `Couldn't run ${item.label}`);
      busyRef.current = false;
      setBusyIndex(null);
    }
  };

  return (
    <div
      ref={ref}
      role="menu"
      aria-label="Context menu"
      aria-busy={busyIndex !== null}
      onKeyDown={onMenuKeyDown}
      style={{
        left: position.left,
        top: position.top,
        minWidth: "min(200px, calc(100vw - 16px))",
        maxWidth: "calc(100vw - 16px)",
        maxHeight: "calc(100vh - 16px)",
      }}
      className="anim-modal fixed z-menu overflow-x-hidden overflow-y-auto rounded-lg border border-border bg-bg-panel py-1 shadow-elev-3"
    >
      {items.map((item, i) => (
        <div key={i}>
          <button
            role="menuitem"
            disabled={item.disabled || !item.onClick || busyIndex !== null}
            aria-busy={busyIndex === i}
            onClick={() => void runItem(item, i)}
            className={cn(
              "flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm hover:bg-bg-hover disabled:opacity-40 disabled:hover:bg-transparent",
              item.destructive && "text-danger hover:bg-danger-soft",
              busyIndex === i && "cursor-wait opacity-70"
            )}
          >
            {item.icon && (
              <span className="flex h-3.5 w-3.5 items-center justify-center text-text-muted">
                {item.icon}
              </span>
            )}
            <span className="min-w-0 flex-1 break-words">
              {busyIndex === i ? "Working…" : item.label}
            </span>
          </button>
          {item.separatorAfter && (
            <div className="my-1 border-t border-border" />
          )}
        </div>
      ))}
    </div>
  );
}
