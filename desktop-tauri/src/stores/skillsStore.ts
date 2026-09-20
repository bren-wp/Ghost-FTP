import { create } from "zustand";
import { ipc, onBridgeSkillProposed } from "@/lib/ipc";
import { toast } from "./toastStore";
import type { Skill, SkillRunResult, SkillDryRunResult } from "@/lib/types";

// Skills panel + store (Plan 8). Mirrors the saved-commands flow in bridgeStore:
// each mutation returns the fresh list, which we drop straight into state. The
// full-screen panel opens over the app (see SkillsHost), so this store also owns
// the `open` flag like searchStore/diskScan.
interface SkillsState {
  /** Whether the full-screen Skills panel is open. */
  open: boolean;
  skills: Skill[];
  loaded: boolean;
  /** Guards the one-time init (list fetch + proposal listener). */
  booted: boolean;

  /** Boot once: fetch skills and subscribe to AI proposals. Returns a cleanup. */
  init: () => Promise<() => void>;
  load: () => Promise<void>;
  openPanel: () => void;
  close: () => void;
  saveSkill: (s: Skill) => Promise<void>;
  deleteSkill: (id: string) => Promise<void>;
  approveSkill: (id: string) => Promise<void>;
  /** Run (or dry-run) a skill; returns the result, or null on a hard failure. */
  run: (
    name: string,
    params: Record<string, string>,
    targets: string[] | null,
    dryRun: boolean
  ) => Promise<SkillRunResult | SkillDryRunResult | null>;
}

export const useSkills = create<SkillsState>((set, get) => ({
  open: false,
  skills: [],
  loaded: false,
  booted: false,

  init: async () => {
    if (!get().booted) {
      set({ booted: true });
      await get().load();
    }
    // When the AI proposes a skill, refresh and nudge the user to approve it.
    const un = await onBridgeSkillProposed((skill) => {
      get().load();
      toast.info(
        "New skill proposal",
        `The AI proposed "${skill.name}". Review and approve it in the Skills panel before it can run.`
      );
    });
    return un;
  },

  load: async () => {
    try {
      set({ skills: await ipc.bridgeListSkills(), loaded: true });
    } catch {
      // backend not ready yet — the proposal listener will still refresh later.
    }
  },

  openPanel: () => {
    set({ open: true });
    get().load();
  },
  close: () => set({ open: false }),

  saveSkill: async (s) => {
    try {
      set({ skills: await ipc.bridgeSaveSkill(s) });
    } catch (e) {
      toast.error("Couldn't save skill", String(e));
    }
  },

  deleteSkill: async (id) => {
    try {
      set({ skills: await ipc.bridgeDeleteSkill(id) });
    } catch (e) {
      toast.error("Couldn't delete skill", String(e));
    }
  },

  approveSkill: async (id) => {
    try {
      set({ skills: await ipc.bridgeApproveSkill(id) });
    } catch (e) {
      toast.error("Couldn't approve skill", String(e));
    }
  },

  run: async (name, params, targets, dryRun) => {
    try {
      return await ipc.bridgeRunSkill(name, params, targets, dryRun);
    } catch (e) {
      toast.error(dryRun ? "Dry-run failed" : "Skill run failed", String(e));
      return null;
    }
  },
}));
