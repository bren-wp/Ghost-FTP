/**
 * Calendar-safe desktop transfer scheduling shared by Windows and Linux.
 * No persistence, IPC, timers or browser globals: deterministic QA tests can
 * evaluate DST gaps and local calendar rollovers without mocking the backend.
 */
export type ScheduleMode = "off" | "once" | "daily" | "weekly";
export interface ScheduleClock {
  mode: ScheduleMode;
  date: string;
  time: string;
}

export function localDateInputValue(date = new Date()): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

export function validDateInput(value: unknown): value is string {
  if (typeof value !== "string" || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  const parsed = new Date(`${value}T12:00:00`);
  return Number.isFinite(parsed.getTime()) && localDateInputValue(parsed) === value;
}

export function parseClock(time: unknown): [number, number] | null {
  if (typeof time !== "string" || !/^\d{2}:\d{2}$/.test(time)) return null;
  const [hour, minute] = time.split(":").map(Number);
  return hour <= 23 && minute <= 59 ? [hour, minute] : null;
}

export function validClock(value: unknown): value is string {
  return parseClock(value) !== null;
}

export function parseLocalOccurrence(date: string, time: string): Date | null {
  if (!validDateInput(date)) return null;
  const clock = parseClock(time);
  if (!clock) return null;
  const [hour, minute] = clock;
  const parsed = new Date(`${date}T${time}:00`);
  if (!Number.isFinite(parsed.getTime()) ||
      localDateInputValue(parsed) !== date ||
      parsed.getHours() !== hour || parsed.getMinutes() !== minute) {
    // Reject nonexistent local times (spring DST gap), never silently
    // shift a transfer to a different hour.
    return null;
  }
  return parsed;
}

function calendarDay(date: Date, daysBack: number): string {
  // Move calendar days at noon to avoid midnight/DST offset transitions.
  const midday = new Date(date.getFullYear(), date.getMonth(), date.getDate() - daysBack, 12);
  return localDateInputValue(midday);
}

/** Latest *valid* calendar occurrence, never a DST-shifted wall-clock time.
 * During fall DST ambiguity, the first occurrence is selected exactly once,
 * keyed by its epoch timestamp in persisted lastRunAt.
 */
export function mostRecentOccurrence(schedule: ScheduleClock, now: Date): Date | null {
  if (!Number.isFinite(now.getTime())) return null;
  const anchor = parseLocalOccurrence(schedule.date, schedule.time);
  if (!anchor || now.getTime() < anchor.getTime()) return null;
  if (schedule.mode === "once") return anchor;

  if (schedule.mode === "daily") {
    for (const daysBack of [0, 1]) {
      const due = parseLocalOccurrence(calendarDay(now, daysBack), schedule.time);
      if (due && due.getTime() <= now.getTime() && due.getTime() >= anchor.getTime()) return due;
    }
  } else if (schedule.mode === "weekly") {
    const daysBack = (now.getDay() - anchor.getDay() + 7) % 7;
    for (const back of [daysBack, daysBack + 7]) {
      const due = parseLocalOccurrence(calendarDay(now, back), schedule.time);
      if (due && due.getTime() <= now.getTime() && due.getTime() >= anchor.getTime()) return due;
    }
  }
  return null;
}
