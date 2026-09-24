import { create } from "zustand";
import { useTransfers } from "./transfersStore";
import { toast } from "./toastStore";
import { messageOf } from "@/lib/errors";

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
let schedulePersistenceWarningShown = false;

function localDateInputValue(date = new Date()) {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

function validDateInput(value: string | undefined): value is string {
  if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  const parsed = new Date(`${value}T12:00:00`);
  return !Number.isNaN(parsed.getTime()) && localDateInputValue(parsed) === value;
}

function parseLocalOccurrence(date: string, time: string): Date | null {
  if (!validDateInput(date)) return null;
  const clock = parseClock(time);
  if (!clock) return null;
  const [hour, minute] = clock;
  const parsed = new Date(`${date}T${time}:00`);
  if (
    Number.isNaN(parsed.getTime()) ||
    localDateInputValue(parsed) !== date ||
    parsed.getHours() !== hour ||
    parsed.getMinutes() !== minute
  ) {
    // Reject rolled invalid dates and local times that do not exist because of
    // a daylight-saving transition.
    return null;
  }
  return parsed;
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
  } catch (error) {
    console.warn("Couldn't read the saved transfer schedule; using defaults", error);
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
  const date = validDateInput(input.date)
    ? input.date
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
  } catch (error) {
    console.warn("Couldn't persist the transfer schedule", error);
    if (!schedulePersistenceWarningShown) {
      schedulePersistenceWarningShown = true;
      toast.warning(
        "Transfer schedule is session-only",
        "Ghost FTP couldn't save this schedule, so it may be lost after restart."
      );
    }
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

/** Most recent occurrence that should have run at or before `now`.
 *  The selected date is the schedule anchor: recurring jobs never run before it. */
function mostRecentOccurrence(
  schedule: TransferScheduleData,
  now: Date
): Date | null {
  const anchor = parseLocalOccurrence(schedule.date, schedule.time);
  if (!anchor || now.getTime() < anchor.getTime()) return null;

  if (schedule.mode === "once") return anchor;

  const clock = parseClock(schedule.time);
  if (!clock) return null;
  const [hour, minute] = clock;

  if (schedule.mode === "daily") {
    const due = new Date(now);
    due.setHours(hour, minute, 0, 0);
    if (due.getTime() > now.getTime()) due.setDate(due.getDate() - 1);
    return due.getTime() >= anchor.getTime() ? due : null;
  }

  if (schedule.mode === "weekly") {
    const due = new Date(now);
    due.setHours(hour, minute, 0, 0);
    const daysBack = (due.getDay() - anchor.getDay() + 7) % 7;
    due.setDate(due.getDate() - daysBack);
    if (due.getTime() > now.getTime()) due.setDate(due.getDate() - 7);
    return due.getTime() >= anchor.getTime() ? due : null;
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

  const transfers = useTransfers.getState();
  if (!transfers.initialized) return;

  const target = transfers.byId[schedule.transferId];
  if (!target) {
    useTransferSchedule.getState().setArmed(false);
    toast.warning(
      "Transfer schedule disabled",
      "The saved transfer no longer exists. Choose a transfer and set the schedule again."
    );
    return;
  }

  const due = mostRecentOccurrence(schedule, new Date());
  if (!due || due.getTime() > Date.now() || due.getTime() <= schedule.lastRunAt) {
    return;
  }

  // Never duplicate a transfer that is already queued/running/paused. Treat
  // this occurrence as satisfied and wait for the next recurrence.
  if (
    target.status === "queued" ||
    target.status === "transferring" ||
    target.status === "paused"
  ) {
    useTransferSchedule.getState().markRun(due.getTime());
    if (schedule.mode === "once") {
      useTransferSchedule.getState().setArmed(false);
    }
    return;
  }

  tickBusy = true;
  try {
    await transfers.retry(schedule.transferId);
    toast.success(
      "Scheduled transfer started",
      "The saved transfer retry is running."
    );
  } catch (error) {
    toast.error("Scheduled transfer failed to start", messageOf(error));
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
  const runSafely = () =>
    void runScheduleTick().catch((error) =>
      console.error("Unexpected transfer scheduler failure", messageOf(error))
    );
  runSafely();
  const timer = window.setInterval(runSafely, 10_000);
  return () => window.clearInterval(timer);
}
