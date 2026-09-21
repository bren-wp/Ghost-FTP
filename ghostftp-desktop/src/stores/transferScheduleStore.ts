import { create } from "zustand";
import { useTransfers } from "./transfersStore";
import { toast } from "./toastStore";

export type TransferScheduleMode = "off" | "once" | "daily" | "weekly";

interface TransferScheduleData {
  mode: TransferScheduleMode;
  date: string;
  time: string;
  transferId: string | null;
  armed: boolean;
  /** Epoch milliseconds of the scheduled occurrence most recently attempted. */
  lastRunAt: number;
}

interface TransferScheduleState extends TransferScheduleData {
  setMode: (mode: TransferScheduleMode) => void;
  setDate: (date: string) => void;
  setTime: (time: string) => void;
  setTarget: (transferId: string | null) => void;
  setArmed: (armed: boolean) => void;
  markRun: (scheduledAt: number) => void;
}

const STORAGE_KEY = "ghostftp.transferSchedule.v2";

function localDateInputValue(date = new Date()) {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

const DEFAULTS: TransferScheduleData = {
  mode: "off",
  date: localDateInputValue(),
  time: "13:00",
  transferId: null,
  armed: false,
  lastRunAt: 0,
};

function loadSchedule(): TransferScheduleData {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) {
      // One-time migration from the RC13 v1 scheduler shape.
      const legacy = localStorage.getItem("ghostftp.transferSchedule.v1");
      if (!legacy) return DEFAULTS;
      const parsed = JSON.parse(legacy) as Partial<TransferScheduleData>;
      const migrated = normalize(parsed);
      persistSchedule(migrated);
      localStorage.removeItem("ghostftp.transferSchedule.v1");
      return migrated;
    }
    return normalize(JSON.parse(raw) as Partial<TransferScheduleData>);
  } catch {
    return DEFAULTS;
  }
}

function normalize(input: Partial<TransferScheduleData>): TransferScheduleData {
  const mode: TransferScheduleMode =
    input.mode === "once" ||
    input.mode === "daily" ||
    input.mode === "weekly" ||
    input.mode === "off"
      ? input.mode
      : "off";
  const date = /^\d{4}-\d{2}-\d{2}$/.test(input.date ?? "")
    ? input.date!
    : localDateInputValue();
  const time = /^\d{2}:\d{2}$/.test(input.time ?? "") ? input.time! : "13:00";
  const transferId =
    typeof input.transferId === "string" && input.transferId
      ? input.transferId
      : null;
  return {
    mode,
    date,
    time,
    transferId,
    armed: Boolean(input.armed && mode !== "off" && transferId),
    lastRunAt:
      typeof input.lastRunAt === "number" && Number.isFinite(input.lastRunAt)
        ? Math.max(0, input.lastRunAt)
        : 0,
  };
}

function persistSchedule(data: TransferScheduleData) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
  } catch {
    // A locked-down WebView may reject storage. The in-memory schedule still
    // functions for this process rather than turning the controls into no-ops.
  }
}

function dataFromState(state: TransferScheduleState): TransferScheduleData {
  return {
    mode: state.mode,
    date: state.date,
    time: state.time,
    transferId: state.transferId,
    armed: state.armed,
    lastRunAt: state.lastRunAt,
  };
}

function update(
  set: (partial: Partial<TransferScheduleState>) => void,
  get: () => TransferScheduleState,
  patch: Partial<TransferScheduleData>
) {
  const current = dataFromState(get());
  const next = normalize({ ...current, ...patch });
  persistSchedule(next);
  set(next);
}

export const useTransferSchedule = create<TransferScheduleState>((set, get) => ({
  ...loadSchedule(),
  setMode: (mode) =>
    update(set, get, mode === "off" ? { mode, armed: false } : { mode }),
  setDate: (date) => update(set, get, { date }),
  setTime: (time) => update(set, get, { time }),
  setTarget: (transferId) =>
    update(set, get, {
      transferId,
      armed: transferId ? get().armed : false,
    }),
  setArmed: (armed) =>
    update(
      set,
      get,
      armed
        ? { armed: true, lastRunAt: Date.now() }
        : { armed: false }
    ),
  markRun: (scheduledAt) => update(set, get, { lastRunAt: scheduledAt }),
}));

function parseClock(time: string): [number, number] | null {
  const [hour, minute] = time.split(":").map(Number);
  if (
    !Number.isInteger(hour) ||
    !Number.isInteger(minute) ||
    hour < 0 ||
    hour > 23 ||
    minute < 0 ||
    minute > 59
  ) {
    return null;
  }
  return [hour, minute];
}

/** Most recent occurrence that should have run at or before `now`. */
function mostRecentOccurrence(
  schedule: TransferScheduleData,
  now: Date
): Date | null {
  const clock = parseClock(schedule.time);
  if (!clock) return null;
  const [hour, minute] = clock;

  if (schedule.mode === "once") {
    const due = new Date(`${schedule.date}T${schedule.time}:00`);
    return Number.isNaN(due.getTime()) ? null : due;
  }

  if (schedule.mode === "daily") {
    const due = new Date(now);
    due.setHours(hour, minute, 0, 0);
    if (due.getTime() > now.getTime()) due.setDate(due.getDate() - 1);
    return due;
  }

  if (schedule.mode === "weekly") {
    const anchor = new Date(`${schedule.date}T${schedule.time}:00`);
    if (Number.isNaN(anchor.getTime())) return null;
    const due = new Date(now);
    due.setHours(hour, minute, 0, 0);
    const daysBack = (due.getDay() - anchor.getDay() + 7) % 7;
    due.setDate(due.getDate() - daysBack);
    if (due.getTime() > now.getTime()) due.setDate(due.getDate() - 7);
    return due;
  }

  return null;
}

let tickBusy = false;

async function runScheduleTick() {
  if (tickBusy) return;
  const schedule = dataFromState(useTransferSchedule.getState());
  if (
    !schedule.armed ||
    schedule.mode === "off" ||
    !schedule.transferId
  ) {
    return;
  }

  const due = mostRecentOccurrence(schedule, new Date());
  if (!due || due.getTime() > Date.now() || due.getTime() <= schedule.lastRunAt) {
    return;
  }

  tickBusy = true;
  try {
    await useTransfers.getState().retry(schedule.transferId);
    toast.success("Scheduled transfer started", "The saved transfer retry is running.");
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    toast.error("Scheduled transfer failed to start", message);
  } finally {
    // Mark the occurrence even on failure so a broken/stale transfer id cannot
    // hammer the native backend every few seconds.
    useTransferSchedule.getState().markRun(due.getTime());
    if (schedule.mode === "once") {
      useTransferSchedule.getState().setArmed(false);
    }
    tickBusy = false;
  }
}

/**
 * Keep schedules alive independently of the Transfer Center screen. This means
 * switching back to File Manager, Preferences or About does not cancel a timer.
 * The schedule is local-only and never creates fake transfer rows or activity.
 */
export function initTransferScheduler() {
  void runScheduleTick();
  const timer = window.setInterval(() => void runScheduleTick(), 10_000);
  return () => window.clearInterval(timer);
}
