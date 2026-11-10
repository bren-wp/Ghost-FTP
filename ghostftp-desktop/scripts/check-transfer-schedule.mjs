import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import ts from "typescript";

const raw = readFileSync(new URL("../src/lib/scheduleTime.ts", import.meta.url), "utf8");
const { outputText, diagnostics } = ts.transpileModule(raw, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
  reportDiagnostics: true,
});
assert.deepEqual(diagnostics ?? [], []);
const { parseClock, validDateInput, parseLocalOccurrence, mostRecentOccurrence } =
  await import("data:text/javascript;base64," + Buffer.from(outputText).toString("base64"));

process.env.TZ = "Europe/Zagreb";
  test("reject invalid dates and invalid 24-hour clocks before arming", () => {
    for (const clock of ["24:00", "12:60", "2:30", "12:00:00", "", "99:99"]) assert.equal(parseClock(clock), null);
    assert.deepEqual(parseClock("02:30"), [2, 30]);
    assert.equal(validDateInput("2026-02-30"), false);
    assert.equal(validDateInput("2026-02-28"), true);
    assert.equal(parseLocalOccurrence("2026-03-29", "02:30"), null, "spring DST wall-clock gap must not roll to 03:30");
  });

  test("daily transfers never silently run at the wrong hour during DST gap", () => {
    const schedule = { mode: "daily", date: "2026-03-01", time: "02:30" };
    const due = mostRecentOccurrence(schedule, new Date("2026-03-29T04:00:00+02:00"));
    assert.equal(due?.getDate(), 28);
    assert.equal(due?.getHours(), 2);
    assert.equal(due?.getMinutes(), 30);
  });

  test("weekly transfers skip nonexistent DST clock without shifting to 03:30", () => {
    const schedule = { mode: "weekly", date: "2026-03-01", time: "02:30" };
    const due = mostRecentOccurrence(schedule, new Date("2026-03-29T04:00:00+02:00"));
    assert.equal(due?.getDate(), 22);
    assert.equal(due?.getHours(), 2);
  });

  test("once schedule fires at the exact anchor, not before", () => {
    const schedule = { mode: "once", date: "2026-11-09", time: "13:00" };
    assert.equal(mostRecentOccurrence(schedule, new Date("2026-11-09T12:59:00+01:00")), null);
    assert.equal(mostRecentOccurrence(schedule, new Date("2026-11-09T13:00:00+01:00"))?.getTime(),
      new Date("2026-11-09T13:00:00+01:00").getTime());
  });

  test("fall DST repeats select one stable occurrence for lastRunAt deduplication", () => {
    const schedule = { mode: "daily", date: "2026-10-01", time: "02:30" };
    const first = mostRecentOccurrence(schedule, new Date("2026-10-25T02:45:00+02:00"));
    const second = mostRecentOccurrence(schedule, new Date("2026-10-25T02:45:00+01:00"));
    assert.equal(first?.getTime(), second?.getTime());
  });

test("a missed schedule remains due after laptop sleep without generating a second occurrence", () => {
  const schedule = { mode: "daily", date: "2026-11-01", time: "13:00" };
  const wake = new Date("2026-11-10T17:00:00+01:00");
  const due = mostRecentOccurrence(schedule, wake);
  assert.equal(due?.toISOString(), new Date("2026-11-10T13:00:00+01:00").toISOString());
  const lastRunAt = due?.getTime() ?? 0;
  assert.ok(lastRunAt > 0);
  assert.equal(mostRecentOccurrence(schedule, wake)?.getTime(), lastRunAt,
    "multiple resume/focus/online events must resolve to the same deduplication key");
});

test("native-window wake triggers check transfers and remove every event listener on unmount", () => {
  const store = readFileSync(new URL("../src/stores/transferScheduleStore.ts", import.meta.url), "utf8");
  for (const [target, event] of [
    ["document", "visibilitychange"],
    ["window", "focus"],
    ["window", "online"],
    ["window", "pageshow"],
  ]) {
    assert.ok(store.includes(target + '.addEventListener("' + event + '"'),
      "wake listener missing: " + event);
    assert.ok(store.includes(target + '.removeEventListener("' + event + '"'),
      "unmount leak: wake listener not removed: " + event);
  }
  assert.ok(store.includes('document.visibilityState === "visible"'));
  assert.ok(store.includes("const timer = window.setInterval(runSafely, 10_000)"));
  assert.ok(store.includes("window.clearInterval(timer)"));
  assert.ok(store.includes("if (tickBusy) return"), "duplicate concurrent retries must be blocked");
  assert.ok(store.includes("due.getTime() <= schedule.lastRunAt"), "each occurrence is handled once");
});
