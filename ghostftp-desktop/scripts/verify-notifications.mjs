// Headless runtime verification for Plan 16 Phase 3 (desktop notifications).
// Boots the mock build, drives the curated events through the real
// notifications.ts wiring, and asserts the OS-toast layer (mocked to record
// sends) fires exactly when it should: unfocused → yes, focused → no,
// toggled off → no; plus editor-save-failure and folder-sync-error events.
// Exit code 0 = all checks passed.
import { spawn } from "node:child_process";
import { setTimeout as sleep } from "node:timers/promises";
import { existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";
import puppeteer from "puppeteer-core";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, "..");
const OUT =
  process.env.SHOT_DIR || path.join(ROOT, ".ghostftp-artifacts");
const PORT = 1425;
const URL = `http://localhost:${PORT}/`;

const chrome = [
  "C:/Program Files/Google/Chrome/Application/chrome.exe",
  "C:/Program Files (x86)/Google/Chrome/Application/chrome.exe",
  "C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe",
  "C:/Program Files/Microsoft/Edge/Application/msedge.exe",
].find((p) => existsSync(p));
if (!chrome) {
  console.error("No Chrome/Edge found.");
  process.exit(2);
}
const isWin = process.platform === "win32";

let failures = 0;
function check(name, cond, detail = "") {
  const ok = !!cond;
  if (!ok) failures++;
  console.log(`  ${ok ? "✓ PASS" : "✗ FAIL"}  ${name}${detail ? "  — " + detail : ""}`);
}

async function waitForServer(url, ms = 60_000) {
  const deadline = Date.now() + ms;
  while (Date.now() < deadline) {
    try {
      const r = await fetch(url);
      if (r.ok) return;
    } catch {}
    await sleep(500);
  }
  throw new Error(`mock server never came up at ${url}`);
}
function killTree(child) {
  if (!child || child.killed) return;
  if (isWin) spawn("taskkill", ["/pid", String(child.pid), "/T", "/F"], { stdio: "ignore" });
  else try { process.kill(-child.pid, "SIGKILL"); } catch {}
}

async function main() {
  console.log("› starting mock Vite server…");
  const server = spawn("npm", ["run", "dev:mock"], {
    cwd: ROOT,
    shell: true,
    stdio: "ignore",
    detached: !isWin,
  });
  let browser;
  try {
    await waitForServer(URL);
    browser = await puppeteer.launch({
      executablePath: chrome,
      headless: true,
      defaultViewport: { width: 1440, height: 900 },
      args: ["--hide-scrollbars"],
    });
    const page = await browser.newPage();
    page.on("pageerror", (e) => {
      console.log("  page error:", e.message);
      failures++;
    });

    const drive = (fn, ...a) => page.evaluate(fn, ...a);

    let booted = false;
    for (let attempt = 0; attempt < 5 && !booted; attempt++) {
      await page.goto(URL, { waitUntil: "networkidle0" });
      try {
        await page.waitForFunction(() => !!window.__demo?.emit, { timeout: 8_000 });
        booted = true;
      } catch {
        console.log(`    · boot attempt ${attempt + 1} missed __demo, reloading…`);
        await sleep(1500);
      }
    }
    if (!booted) throw new Error("app never exposed window.__demo");
    await sleep(800); // let App effects (initNotifications) run

    const reset = () => drive(() => { window.__notifications = []; });
    const notifs = () => drive(() => window.__notifications || []);
    const setFocus = (v) => drive((f) => { window.__focused = f; }, v);
    // Emit fuller Transfer payloads: the real transfers store also listens and
    // reads t.source/kind/status, so a bare {id} would break it.
    const driveTransferBatch = async (doneIds, errIds) =>
      drive(
        (d, e) => {
          const mk = (id, status, error) => ({
            id,
            kind: "download",
            source: `/srv/data/${id}.bin`,
            status,
            error: error ?? null,
          });
          for (const id of [...d, ...e]) window.__demo.emit("transfer://added", mk(id, "transferring", null));
          for (const id of d) window.__demo.emit("transfer://done", mk(id, "completed", null));
          for (const id of e) window.__demo.emit("transfer://error", mk(id, "failed", "network error"));
        },
        doneIds,
        errIds
      );

    // ---- 1. Unfocused transfer batch → one summary toast ----
    await setFocus(false);
    await reset();
    await driveTransferBatch(["t1"], ["t2"]); // 1 done, 1 failed
    await sleep(400);
    let n = await notifs();
    check(
      "unfocused transfer batch fires one toast",
      n.length === 1,
      `got ${n.length}`
    );
    check(
      "toast summarises done/failed counts",
      n[0] && /finished with errors/i.test(n[0].title) && /1 done/.test(n[0].body) && /1 failed/.test(n[0].body),
      JSON.stringify(n[0] || null)
    );

    // ---- 2. Focused transfer batch → NO toast ----
    await setFocus(true);
    await reset();
    await driveTransferBatch(["t3", "t4"], []); // 2 done
    await sleep(400);
    n = await notifs();
    check("focused transfer batch fires no toast (unfocused-only default)", n.length === 0, `got ${n.length}`);

    // ---- 3. Toggled off → NO toast even when unfocused ----
    await setFocus(false);
    await drive(() =>
      window.__demo.useSettings.getState().setNotifications({ enabled: false, unfocusedOnly: true })
    );
    await reset();
    await driveTransferBatch(["t5"], []);
    await sleep(400);
    n = await notifs();
    check("notifications toggled off → no toast", n.length === 0, `got ${n.length}`);
    // Re-enable for the remaining checks.
    await drive(() =>
      window.__demo.useSettings.getState().setNotifications({ enabled: true, unfocusedOnly: true })
    );

    // ---- 4. Successful-only batch → "complete" toast ----
    await reset();
    await driveTransferBatch(["t6", "t7", "t8"], []); // 3 done, 0 failed
    await sleep(400);
    n = await notifs();
    check(
      "all-success batch → 'complete' toast with file count",
      n.length === 1 && /complete/i.test(n[0].title) && /3 files/.test(n[0].body),
      JSON.stringify(n[0] || null)
    );

    // ---- 5. Edit-in-place save failure → toast ----
    await reset();
    await drive(() =>
      window.__demo.emit("editor://error", {
        editId: "e1",
        remotePath: "/var/www/site/wp-config.php",
        message: "permission denied",
      })
    );
    await sleep(400);
    n = await notifs();
    check(
      "edit-in-place save failure fires a toast naming the file",
      n.length === 1 && /save/i.test(n[0].title) && /wp-config\.php/.test(n[0].body),
      JSON.stringify(n[0] || null)
    );

    // ---- 6. Folder-sync pair entering error state → toast ----
    await reset();
    await drive(() =>
      window.__demo.useSync.setState({
        pairs: [
          { id: "p1", name: "Prod uploads", state: "error", lastError: "connection refused" },
        ],
      })
    );
    await sleep(400);
    n = await notifs();
    check(
      "folder-sync pair entering error fires a toast",
      n.length === 1 && /sync error/i.test(n[0].title) && /Prod uploads/.test(n[0].body),
      JSON.stringify(n[0] || null)
    );
    // Staying in error must NOT re-fire.
    await reset();
    await drive(() =>
      window.__demo.useSync.setState({
        pairs: [
          { id: "p1", name: "Prod uploads", state: "error", lastError: "connection refused" },
          { id: "p2", name: "Other", state: "idle", lastError: null },
        ],
      })
    );
    await sleep(300);
    n = await notifs();
    check("a pair staying in error does not re-fire", n.length === 0, `got ${n.length}`);

    console.log(`\n${failures === 0 ? "✓ ALL CHECKS PASSED" : `✗ ${failures} CHECK(S) FAILED`}`);
  } finally {
    if (browser) await browser.close();
    killTree(server);
  }
  process.exit(failures === 0 ? 0 : 1);
}

main().catch((e) => {
  console.error(e);
  process.exit(2);
});
