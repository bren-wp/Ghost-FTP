import { create } from "zustand";

export type ToastVariant = "info" | "success" | "error" | "warning";

export interface Toast {
  id: string;
  variant: ToastVariant;
  title: string;
  message?: string;
  createdAt: number;
}

interface PushArgs {
  variant: ToastVariant;
  title: string;
  message?: string;
  duration?: number; // ms; 0 = sticky until dismissed
}

interface ToastState {
  toasts: Toast[]; // currently on-screen
  history: Toast[]; // bounded, persisted log for the notification center
  unreadCount: number;

  push: (args: PushArgs) => string;
  dismiss: (id: string) => void;
  markAllRead: () => void;
  clearHistory: () => void;
}

const HISTORY_KEY = "ghostftp.notifications.v1";
const MAX_HISTORY = 100;

let _seq = 0;
const nextId = () => `n${Date.now().toString(36)}${++_seq}`;

function loadHistory(): Toast[] {
  try {
    const raw = localStorage.getItem(HISTORY_KEY);
    return raw ? (JSON.parse(raw) as Toast[]) : [];
  } catch (error) {
    console.warn("Couldn't read Ghost FTP notification history", error);
    return [];
  }
}

function saveHistory(h: Toast[]) {
  try {
    localStorage.setItem(HISTORY_KEY, JSON.stringify(h));
  } catch (error) {
    console.warn("Couldn't persist Ghost FTP notification history", error);
  }
}

export const useToasts = create<ToastState>((set, get) => ({
  toasts: [],
  history: loadHistory(),
  unreadCount: 0,

  push: ({ variant, title, message, duration }) => {
    const item: Toast = { id: nextId(), variant, title, message, createdAt: Date.now() };
    const history = [item, ...get().history].slice(0, MAX_HISTORY);
    saveHistory(history);
    set((s) => ({
      toasts: [...s.toasts, item],
      history,
      unreadCount: s.unreadCount + 1,
    }));
    const ms = duration ?? (variant === "error" ? 8000 : 4000);
    if (ms > 0) setTimeout(() => get().dismiss(item.id), ms);
    return item.id;
  },

  dismiss: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
  markAllRead: () => set({ unreadCount: 0 }),
  clearHistory: () => {
    saveHistory([]);
    set({ history: [] });
  },
}));

// Call-from-anywhere helper (stores, ipc handlers, components).
export const toast = {
  info: (title: string, message?: string) =>
    useToasts.getState().push({ variant: "info", title, message }),
  success: (title: string, message?: string) =>
    useToasts.getState().push({ variant: "success", title, message }),
  error: (title: string, message?: string) =>
    useToasts.getState().push({ variant: "error", title, message }),
  warning: (title: string, message?: string) =>
    useToasts.getState().push({ variant: "warning", title, message }),
};
