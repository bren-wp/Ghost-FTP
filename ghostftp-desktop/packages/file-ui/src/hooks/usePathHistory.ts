import { useEffect, useReducer, useRef } from "react";

// Per-pane browser-style navigation history. Watches the controlled `path`:
// any change that isn't a back/forward move is recorded as a new entry
// (truncating any forward history), so Back/Forward work regardless of how the
// path changed (row click, breadcrumb, up, typed, profile switch).
export function usePathHistory(path: string, navigate: (p: string) => void) {
  const stackRef = useRef<string[]>([path]);
  const idxRef = useRef(0);
  const [, force] = useReducer((x: number) => x + 1, 0);

  useEffect(() => {
    const stack = stackRef.current;
    if (stack[idxRef.current] === path) return; // back/forward landing or no-op
    stack.splice(idxRef.current + 1); // drop forward history
    stack.push(path);
    idxRef.current = stack.length - 1;
    force();
  }, [path]);

  const back = () => {
    if (idxRef.current > 0) {
      idxRef.current -= 1;
      force();
      navigate(stackRef.current[idxRef.current]);
    }
  };
  const forward = () => {
    if (idxRef.current < stackRef.current.length - 1) {
      idxRef.current += 1;
      force();
      navigate(stackRef.current[idxRef.current]);
    }
  };

  return {
    back,
    forward,
    canBack: idxRef.current > 0,
    canForward: idxRef.current < stackRef.current.length - 1,
  };
}
