import { create } from "zustand";
import { ipc } from "@/lib/ipc";
import { toast } from "./toastStore";
import {
  extractVariables,
  resolveVariables,
  stripTrailingNewline,
} from "@/lib/snippets";
import { insertIntoTerminal } from "@/lib/termInput";
import type { Snippet } from "@/lib/types";

// Command snippets store (Plan 11 Phase 4). Mirrors skillsStore/bridgeStore:
// each mutation returns the fresh, re-ordered list which we drop into state.
// The store also owns the full-screen panel `open` flag and the pending-insert
// target (the snippet whose {{variable}} dialog / multi-line confirm is open).
interface SnippetsState {
  /** Whether the full-screen Snippets panel is open. */
  open: boolean;
  snippets: Snippet[];
  booted: boolean;
  /** The snippet whose insert dialog is open (variables to fill and/or a
   *  multi-line confirm), or null when no dialog is showing. */
  insertTarget: Snippet | null;

  /** Boot once: fetch the list. Cheap and idempotent. */
  init: () => Promise<void>;
  load: () => Promise<void>;
  openPanel: () => void;
  closePanel: () => void;
  save: (s: Snippet) => Promise<void>;
  remove: (id: string) => Promise<void>;

  /** User picked a snippet to insert. Opens the dialog when it needs variables
   *  or is multi-line (a preview/confirm); otherwise injects immediately. */
  requestInsert: (s: Snippet) => void;
  cancelInsert: () => void;
  /** Resolve `values` into the body and inject it at the focused terminal's
   *  prompt (never auto-submitted). Bumps the snippet's use count on success. */
  commitInsert: (s: Snippet, values: Record<string, string>) => Promise<void>;
}

function newSnippet(): Snippet {
  return {
    id: crypto.randomUUID(),
    name: "",
    body: "",
    folder: null,
    useCount: 0,
    createdMs: 0,
    updatedMs: 0,
  };
}

/** True when inserting this body would need a dialog first: it has variables to
 *  fill, or interior newlines the user should confirm before they run. */
function needsDialog(body: string): boolean {
  if (extractVariables(body).length > 0) return true;
  return stripTrailingNewline(body).includes("\n");
}

function inject(s: Snippet, values: Record<string, string>): boolean {
  const resolved = stripTrailingNewline(resolveVariables(s.body, values));
  const ok = insertIntoTerminal(resolved);
  if (!ok) {
    toast.warning(
      "No terminal to insert into",
      "Open a terminal on this connection first, then insert the snippet."
    );
    return false;
  }
  // Fire-and-forget usage bump; refresh the list when it lands so ordering
  // reflects the new count on the next palette open.
  ipc
    .snippetRun(s.id)
    .then((list) => useSnippets.setState({ snippets: list }))
    .catch(() => {});
  return true;
}

export const useSnippets = create<SnippetsState>((set, get) => ({
  open: false,
  snippets: [],
  booted: false,
  insertTarget: null,

  init: async () => {
    if (get().booted) return;
    set({ booted: true });
    await get().load();
  },

  load: async () => {
    try {
      set({ snippets: (await ipc.snippetList()) ?? [] });
    } catch {
      // Backend not ready yet — a later openPanel() / init() retries.
    }
  },

  openPanel: () => {
    set({ open: true });
    get().load();
  },
  closePanel: () => set({ open: false }),

  save: async (s) => {
    try {
      set({ snippets: await ipc.snippetSave(s) });
    } catch (e) {
      toast.error("Couldn't save snippet", String(e));
    }
  },

  remove: async (id) => {
    try {
      set({ snippets: await ipc.snippetDelete(id) });
    } catch (e) {
      toast.error("Couldn't delete snippet", String(e));
    }
  },

  requestInsert: (s) => {
    if (needsDialog(s.body)) {
      set({ insertTarget: s });
    } else {
      inject(s, {});
    }
  },
  cancelInsert: () => set({ insertTarget: null }),
  commitInsert: async (s, values) => {
    inject(s, values);
    set({ insertTarget: null });
  },
}));

export { newSnippet };
