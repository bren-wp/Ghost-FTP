import { create } from "zustand";
import { ipc } from "@/lib/ipc";
import { toast } from "./toastStore";
import { toastError, messageOf } from "@/lib/errors";
import type { ConnectionProfile, SessionId } from "@/lib/types";
import { useTerminals } from "./terminalsStore";

// One live connection. The backend keeps every session alive in a map, so the
// app can hold several at once; this is the frontend's view of them.
export interface LiveSession {
  sessionId: SessionId;
  profileId: string;
  /** True for Quick Connect sessions that are intentionally never persisted. */
  ephemeral?: boolean;
}

interface ConnectionsState {
  profiles: ConnectionProfile[];
  /** Every live connection, in the order they were opened. */
  sessions: LiveSession[];
  /** The focused session — drives the browser, terminal and status bar. */
  activeSessionId: SessionId | null;
  /** Profile of the focused session; kept in sync with activeSessionId. */
  activeProfileId: string | null;
  connecting: boolean;
  error: string | null;

  loadProfiles: () => Promise<void>;
  saveProfile: (p: ConnectionProfile) => Promise<void>;
  duplicateProfile: (id: string) => Promise<ConnectionProfile>;
  /** Save several profiles with a single reload at the end (group rename etc). */
  saveProfiles: (ps: ConnectionProfile[]) => Promise<void>;
  /** Persist a drag-and-drop rail order (every profile id, display order).
   *  Optionally re-homes the dragged profile into a new group first. */
  reorderProfiles: (
    ids: string[],
    groupChange?: { id: string; group: string | undefined }
  ) => Promise<void>;
  deleteProfile: (id: string) => Promise<void>;
  /** Connect to a profile, or focus its existing session if already open. */
  connect: (profileId: string) => Promise<void>;
  /** Connect with an in-memory profile without writing credentials or profile data. */
  connectTemporary: (profile: ConnectionProfile) => Promise<void>;
  /** Disconnect one session (defaults to the active one). */
  disconnect: (sessionId?: SessionId) => Promise<void>;
  /** Focus an already-open session. */
  setActiveSession: (sessionId: SessionId) => void;
}

export const useConnections = create<ConnectionsState>((set, get) => ({
  profiles: [],
  sessions: [],
  activeSessionId: null,
  activeProfileId: null,
  connecting: false,
  error: null,

  loadProfiles: async () => {
    const persisted = await ipc.listProfiles();
    const state = get();
    const ephemeralIds = new Set(
      state.sessions.filter((session) => session.ephemeral).map((session) => session.profileId)
    );
    const ephemeral = state.profiles.filter((profile) => ephemeralIds.has(profile.id));
    const persistedIds = new Set(persisted.map((profile) => profile.id));
    set({ profiles: [...persisted, ...ephemeral.filter((profile) => !persistedIds.has(profile.id))] });
  },

  saveProfile: async (p) => {
    await ipc.saveProfile(p);
    await get().loadProfiles();
  },

  duplicateProfile: async (id) => {
    const duplicated = await ipc.duplicateProfile(id);
    await get().loadProfiles();
    return duplicated;
  },

  saveProfiles: async (ps) => {
    for (const p of ps) await ipc.saveProfile(p);
    await get().loadProfiles();
  },

  reorderProfiles: async (ids, groupChange) => {
    if (groupChange) {
      const p = get().profiles.find((x) => x.id === groupChange.id);
      if (p) await ipc.saveProfile({ ...p, group: groupChange.group });
    }
    await ipc.reorderProfiles(ids);
    await get().loadProfiles();
  },

  deleteProfile: async (id) => {
    await ipc.deleteProfile(id);
    await get().loadProfiles();
  },

  connect: async (profileId) => {
    // Already connected to this profile? Just focus its tab.
    const existing = get().sessions.find((s) => s.profileId === profileId);
    if (existing) {
      get().setActiveSession(existing.sessionId);
      return;
    }

    set({ connecting: true, error: null });
    const profile = get().profiles.find((p) => p.id === profileId);
    try {
      const sessionId = await ipc.connect(profileId);
      set((s) => ({
        sessions: [...s.sessions, { sessionId, profileId }],
        activeSessionId: sessionId,
        activeProfileId: profileId,
        connecting: false,
      }));
      void ipc.bridgeSetActiveSession(sessionId);
      toast.success(
        "Connected",
        profile
          ? `${profile.name} — ${profile.username}@${profile.host}`
          : undefined
      );
      // Refresh persisted profile metadata such as lastUsed without disturbing
      // any active ephemeral Quick Connect profiles.
      void get().loadProfiles();
    } catch (e) {
      // `connect` returns a structured {kind, message} error (Plan 12 Phase 3),
      // so the toast is keyed off the kind (auth → reconnect, network → check
      // connection, …) instead of regexing the message.
      set({ connecting: false, error: messageOf(e) });
      toastError(e, profile ? `Couldn't connect to ${profile.name}` : "Connection failed");
      throw e;
    }
  },

  connectTemporary: async (profile) => {
    const existing = get().sessions.find((session) => session.profileId === profile.id);
    if (existing) {
      get().setActiveSession(existing.sessionId);
      return;
    }

    set({ connecting: true, error: null });
    try {
      const sessionId = await ipc.connectEphemeral(profile);
      set((state) => ({
        profiles: state.profiles.some((item) => item.id === profile.id)
          ? state.profiles
          : [...state.profiles, profile],
        sessions: [...state.sessions, { sessionId, profileId: profile.id, ephemeral: true }],
        activeSessionId: sessionId,
        activeProfileId: profile.id,
        connecting: false,
      }));
      void ipc.bridgeSetActiveSession(sessionId);
      toast.success("Connected", `${profile.name} — ${profile.username}@${profile.host}`);
    } catch (e) {
      set({ connecting: false, error: messageOf(e) });
      toastError(e, `Couldn't connect to ${profile.name}`);
      throw e;
    }
  },

  disconnect: async (sessionId) => {
    const sid = sessionId ?? get().activeSessionId;
    if (!sid) return;
    const target = get().sessions.find((s) => s.sessionId === sid);
    const profile = get().profiles.find((p) => p.id === target?.profileId);
    try {
      await ipc.disconnect(sid);
    } catch (error) {
      // A dropped/broken transport may make the backend disconnect fail even
      // though the UI still has to forget the dead session. Surface the real
      // backend error, then continue the local cleanup instead of failing
      // silently or leaving a stale Connected state behind.
      toastError(error, profile ? `Couldn't cleanly disconnect from ${profile.name}` : "Disconnect failed");
    }
    useTerminals.getState().dropSessionTabs(sid);
    set((s) => {
      const sessions = s.sessions.filter((x) => x.sessionId !== sid);
      const profiles = target?.ephemeral && !sessions.some((x) => x.profileId === target.profileId)
        ? s.profiles.filter((p) => p.id !== target.profileId)
        : s.profiles;
      // If we closed the focused session, fall back to the most recent one.
      let activeSessionId = s.activeSessionId;
      let activeProfileId = s.activeProfileId;
      if (s.activeSessionId === sid) {
        const next = sessions[sessions.length - 1] ?? null;
        activeSessionId = next?.sessionId ?? null;
        activeProfileId = next?.profileId ?? null;
      }
      return { profiles, sessions, activeSessionId, activeProfileId };
    });
    void ipc.bridgeSetActiveSession(get().activeSessionId);
    toast.info("Disconnected", profile?.name);
  },

  setActiveSession: (sessionId) => {
    const target = get().sessions.find((s) => s.sessionId === sessionId);
    if (!target) return;
    set({ activeSessionId: sessionId, activeProfileId: target.profileId });
    // Keep the Agent Bridge aware of which connection the user is focused on.
    void ipc.bridgeSetActiveSession(sessionId);
  },
}));
