import { create } from "zustand";
import {
  ipc,
  onBridgeApproval,
  onBridgeActivity,
  onAgentExecStart,
  onAgentOutput,
} from "@/lib/ipc";
import { toast } from "./toastStore";
import type {
  BridgeStatus,
  BridgeActivity,
  BridgeApproval,
  ApprovalDecision,
  ApprovalPolicy,
  AgentConsoleEntry,
  SavedCommand,
} from "@/lib/types";

const MAX_CONSOLE = 200;
// Cap a single entry's streamed output so a runaway command can't bloat memory.
const MAX_ENTRY_OUTPUT = 256 * 1024;

const EMPTY: BridgeStatus = {
  running: false,
  enabled: false,
  url: null,
  port: null,
  token: null,
  enabledSessions: [],
  policy: { allowAll: false, autoRead: false, autoSafeExec: false, allowSudo: false },
};

interface BridgeStoreState {
  status: BridgeStatus;
  activity: BridgeActivity[];
  approvals: BridgeApproval[];
  console: AgentConsoleEntry[];
  savedCommands: SavedCommand[];
  loaded: boolean;

  init: () => Promise<() => void>;
  refresh: () => Promise<void>;
  start: () => Promise<void>;
  stop: () => Promise<void>;
  setEnabled: (enabled: boolean) => Promise<void>;
  setSessionAccess: (sessionId: string, enabled: boolean) => Promise<void>;
  setPolicy: (policy: ApprovalPolicy) => Promise<void>;
  respond: (requestId: string, decision: ApprovalDecision) => Promise<void>;
  clearConsole: () => void;
  // History (in-memory activity log, last 200).
  refreshActivity: () => Promise<void>;
  clearActivity: () => Promise<void>;
  // Saved commands (pre-approved; managed only in the UI).
  loadCommands: () => Promise<void>;
  saveCommand: (command: SavedCommand) => Promise<void>;
  deleteCommand: (id: string) => Promise<void>;
}

export const useBridge = create<BridgeStoreState>((set, get) => ({
  status: EMPTY,
  activity: [],
  approvals: [],
  console: [],
  savedCommands: [],
  loaded: false,

  init: async () => {
    if (!get().loaded) {
      set({ loaded: true });
      try {
        const [status, activity, savedCommands] = await Promise.all([
          ipc.bridgeStatus(),
          ipc.bridgeActivity(),
          ipc.bridgeListCommands(),
        ]);
        set({ status, activity: activity.slice().reverse(), savedCommands });
      } catch {
        // backend not ready yet — listeners below still attach
      }
    }
    const unApproval = await onBridgeApproval((a) => {
      set((s) => ({ approvals: [...s.approvals, a] }));
    });
    const unActivity = await onBridgeActivity((e) => {
      set((s) => {
        // Finalize a streamed exec entry (same id), or add a fresh done entry
        // for ops that don't stream (read/list/download/upload/search/denied).
        const idx = s.console.findIndex((c) => c.id === e.id);
        let console_: AgentConsoleEntry[];
        if (idx >= 0) {
          console_ = s.console.slice();
          console_[idx] = {
            ...console_[idx],
            status: "done" as const,
            ok: e.ok,
            kind: e.kind,
            command: console_[idx].command || e.detail,
          };
        } else {
          console_ = [
            ...s.console,
            {
              id: e.id,
              sessionId: e.sessionId,
              kind: e.kind,
              command: e.detail,
              output: "",
              status: "done" as const,
              ok: e.ok,
              at: e.at,
            },
          ].slice(-MAX_CONSOLE);
        }
        return { activity: [e, ...s.activity].slice(0, 200), console: console_ };
      });
    });
    const unExecStart = await onAgentExecStart((e) => {
      set((s) => ({
        console: [
          ...s.console,
          {
            id: e.opId,
            sessionId: e.sessionId,
            sessionName: e.sessionName,
            kind: "exec",
            command: e.command,
            output: "",
            status: "running" as const,
            at: Date.now(),
          },
        ].slice(-MAX_CONSOLE),
      }));
    });
    const unOutput = await onAgentOutput((e) => {
      set((s) => ({
        console: s.console.map((c) =>
          c.id === e.opId
            ? { ...c, output: (c.output + e.chunk).slice(-MAX_ENTRY_OUTPUT) }
            : c
        ),
      }));
    });
    return () => {
      unApproval();
      unActivity();
      unExecStart();
      unOutput();
    };
  },

  refresh: async () => {
    try {
      set({ status: await ipc.bridgeStatus() });
    } catch {
      // ignore
    }
  },

  start: async () => {
    try {
      const status = await ipc.bridgeStart();
      set({ status });
      toast.success("Agent Bridge started", status.url ?? undefined);
    } catch (e) {
      toast.error("Couldn't start Agent Bridge", String(e));
    }
  },

  stop: async () => {
    try {
      const status = await ipc.bridgeStop();
      set({ status });
      toast.info("Agent Bridge stopped");
    } catch (e) {
      toast.error("Couldn't stop Agent Bridge", String(e));
    }
  },

  setEnabled: async (enabled) => {
    // Optimistic: reflect the master switch immediately.
    set((s) => ({ status: { ...s.status, enabled } }));
    try {
      const status = await ipc.bridgeSetEnabled(enabled);
      set({ status });
      toast.info(enabled ? "Agent Bridge enabled" : "Agent Bridge disabled");
    } catch (e) {
      toast.error("Couldn't update Agent Bridge", String(e));
      // The bridge may have failed to bind a port; reconcile with the backend.
      void get().refresh();
    }
  },

  setSessionAccess: async (sessionId, enabled) => {
    try {
      const status = await ipc.bridgeSetSessionAccess(sessionId, enabled);
      set({ status });
    } catch (e) {
      toast.error("Couldn't update agent access", String(e));
    }
  },

  setPolicy: async (policy) => {
    // Optimistic: reflect the toggle immediately, reconcile with the backend.
    set((s) => ({ status: { ...s.status, policy } }));
    try {
      const status = await ipc.bridgeSetPolicy(policy);
      set({ status });
    } catch (e) {
      toast.error("Couldn't update auto-approve policy", String(e));
      void get().refresh();
    }
  },

  respond: async (requestId, decision) => {
    // Drop it from the local queue immediately for snappy UI.
    set((s) => ({
      approvals: s.approvals.filter((a) => a.requestId !== requestId),
    }));
    try {
      await ipc.respondToBridgeApproval(requestId, decision);
    } catch {
      // the request may have already timed out on the backend
    }
  },

  clearConsole: () => set({ console: [] }),

  refreshActivity: async () => {
    try {
      set({ activity: (await ipc.bridgeActivity()).slice().reverse() });
    } catch {
      // ignore
    }
  },

  clearActivity: async () => {
    set({ activity: [] });
    try {
      await ipc.bridgeClearActivity();
    } catch {
      // best effort — the view is already cleared
    }
  },

  loadCommands: async () => {
    try {
      set({ savedCommands: await ipc.bridgeListCommands() });
    } catch {
      // ignore
    }
  },

  saveCommand: async (command) => {
    try {
      set({ savedCommands: await ipc.bridgeSaveCommand(command) });
    } catch (e) {
      toast.error("Couldn't save command", String(e));
    }
  },

  deleteCommand: async (id) => {
    try {
      set({ savedCommands: await ipc.bridgeDeleteCommand(id) });
    } catch (e) {
      toast.error("Couldn't delete command", String(e));
    }
  },
}));
