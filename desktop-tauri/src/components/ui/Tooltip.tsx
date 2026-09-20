import { useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { cn } from "@/lib/cn";

type Side = "top" | "bottom" | "left" | "right";

// Absolute-mode placement (default): positioned relative to the trigger.
const POS: Record<Side, string> = {
  top: "bottom-full left-1/2 -translate-x-1/2 mb-1.5",
  bottom: "top-full left-1/2 -translate-x-1/2 mt-1.5",
  left: "right-full top-1/2 -translate-y-1/2 mr-1.5",
  right: "left-full top-1/2 -translate-y-1/2 ml-1.5",
};

// Portal-mode placement: fixed to the viewport, so the tooltip escapes any
// ancestor with `overflow` clipping (e.g. the server rail's scroll column).
const GAP = 6;
const FIXED_TRANSFORM: Record<Side, string> = {
  top: "translate(-50%, -100%)",
  bottom: "translate(-50%, 0)",
  left: "translate(-100%, -50%)",
  right: "translate(0, -50%)",
};
function fixedCoords(side: Side, r: DOMRect): { left: number; top: number } {
  switch (side) {
    case "top":
      return { left: r.left + r.width / 2, top: r.top - GAP };
    case "bottom":
      return { left: r.left + r.width / 2, top: r.bottom + GAP };
    case "left":
      return { left: r.left - GAP, top: r.top + r.height / 2 };
    case "right":
      return { left: r.right + GAP, top: r.top + r.height / 2 };
  }
}

// Lightweight hover tooltip. Lets icon-only buttons stay compact while remaining
// discoverable. Pass `portal` when the trigger sits inside an `overflow`-clipping
// container (a scroll area) so the tooltip renders to <body> and isn't cut off.
export function Tooltip({
  label,
  children,
  side = "top",
  portal = false,
  className,
}: {
  label: ReactNode;
  children: ReactNode;
  side?: Side;
  portal?: boolean;
  /** Extra classes for the wrapper span (e.g. `w-full` for list rows). */
  className?: string;
}) {
  const [open, setOpen] = useState(false);
  const [coords, setCoords] = useState<{ left: number; top: number } | null>(
    null
  );
  const ref = useRef<HTMLSpanElement>(null);

  const show = () => {
    if (portal && ref.current) {
      setCoords(fixedCoords(side, ref.current.getBoundingClientRect()));
    }
    setOpen(true);
  };
  const hide = () => setOpen(false);

  const tipClass =
    "anim-fade pointer-events-none z-tooltip whitespace-nowrap rounded-md border border-border bg-bg-panel px-2 py-1 text-[10px] font-medium text-text shadow-elev-2";

  return (
    <span
      ref={ref}
      className={cn("relative inline-flex", className)}
      onMouseEnter={show}
      onMouseLeave={hide}
      onFocus={show}
      onBlur={hide}
    >
      {children}
      {open &&
        label &&
        (portal
          ? coords &&
            createPortal(
              <span
                role="tooltip"
                style={{
                  left: coords.left,
                  top: coords.top,
                  transform: FIXED_TRANSFORM[side],
                }}
                className={cn("fixed", tipClass)}
              >
                {label}
              </span>,
              document.body
            )
          : (
            <span role="tooltip" className={cn("absolute", tipClass, POS[side])}>
              {label}
            </span>
          ))}
    </span>
  );
}
