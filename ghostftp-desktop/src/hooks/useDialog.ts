import { useEffect, useRef, type RefObject } from "react";

const FOCUSABLE = [
  "a[href]",
  "button:not([disabled])",
  "input:not([disabled])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  '[tabindex]:not([tabindex="-1"])',
].join(",");

interface Options {
  /** Called on Escape. For a security prompt, wire this to the *safe* action. */
  onClose?: () => void;
  /** Element to focus on open. Defaults to the first focusable, else the panel. */
  initialFocus?: RefObject<HTMLElement | null>;
  /**
   * Whether the dialog is open. Conditionally-mounted modals can leave this at
   * the default; always-mounted ones (that toggle an internal flag) must pass
   * their open state so focus management re-runs each time they appear.
   */
  enabled?: boolean;
}

// The accessible-dialog behaviors every modal needs, in one place:
//   • moves focus into the dialog on open (the safe action, if given)
//   • traps Tab / Shift+Tab inside it
//   • closes on Escape
//   • restores focus to whatever was focused before it opened
// Pass a ref to the dialog panel (the box, not the backdrop).
export function useDialog(
  ref: RefObject<HTMLElement | null>,
  { onClose, initialFocus, enabled = true }: Options = {}
) {
  // Keep the latest onClose without re-binding the listener every render.
  const onCloseRef = useRef(onClose);
  onCloseRef.current = onClose;

  useEffect(() => {
    if (!enabled) return;
    const node = ref.current;
    if (!node) return;
    const previouslyFocused = document.activeElement as HTMLElement | null;

    const focusables = () =>
      Array.from(node.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
        (el) => el.offsetParent !== null || el === document.activeElement
      );

    // Move focus in. rAF so it lands after the entrance animation mounts.
    const initial = initialFocus?.current ?? focusables()[0] ?? node;
    if (initial === node && !node.hasAttribute("tabindex")) {
      node.setAttribute("tabindex", "-1");
    }
    const raf = requestAnimationFrame(() => initial.focus());

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.stopPropagation();
        onCloseRef.current?.();
        return;
      }
      if (e.key !== "Tab") return;
      const items = focusables();
      if (items.length === 0) {
        e.preventDefault();
        node.focus();
        return;
      }
      const first = items[0];
      const last = items[items.length - 1];
      const active = document.activeElement;
      if (e.shiftKey && (active === first || !node.contains(active))) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && (active === last || !node.contains(active))) {
        e.preventDefault();
        first.focus();
      }
    };

    node.addEventListener("keydown", onKeyDown);
    return () => {
      cancelAnimationFrame(raf);
      node.removeEventListener("keydown", onKeyDown);
      // Return focus to the opener if it's still around.
      if (previouslyFocused && document.contains(previouslyFocused)) {
        previouslyFocused.focus();
      }
    };
    // ref / initialFocus are stable useRef objects; re-bind only when the
    // dialog's open state flips (for always-mounted modals).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled]);
}
