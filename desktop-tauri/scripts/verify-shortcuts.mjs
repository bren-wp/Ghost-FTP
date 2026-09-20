// Headless runtime verification for Plan 15 (keyboard shortcuts: remapping +
// non-modifier file-browser keys). Spins up the mock Vite build (no Rust),
// drives the real UI, and asserts on the rendered DOM + store state.
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

const CHROME_CANDIDATES = [
  "C:/Program Files/Google/Chrome/Application/chrome.exe",
  "C:/Program Files (x86)/Google/Chrome/Application/chrome.exe",
  "C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe",
  "C:/Program Files/Microsoft/Edge/Application/msedge.exe",
];
const chrome = CHROME_CANDIDATES.find((p) => existsSync(p));
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
      defaultViewport: { width: 1440, height: 900, deviceScaleFactor: 1 },
      args: ["--hide-scrollbars"],
    });
    const page = await browser.newPage();
    page.on("pageerror", (e) => {
      console.log("  page error:", e.message);
      failures++;
    });
    await page.goto(URL, { waitUntil: "networkidle0" });

    const drive = (fn, ...a) => page.evaluate(fn, ...a);
    const shot = async (name) => {
      await page.screenshot({ path: path.join(OUT, `verify-shortcuts-${name}.png`) });
      console.log("    · shot", `verify-shortcuts-${name}.png`);
    };

    // Auto-connect settles into an active (sftp) session.
    await page.waitForFunction(
      () => !!window.__demo?.useConnections.getState().activeSessionId,
      { timeout: 15_000 }
    );
    await sleep(600);

    // ---- 1. Override a command binding, dispatch it, confirm it fires ----
    await drive(() => {
      window.__demo.useLayout.getState().setTerminalOpen(false);
      window.__demo.useBindings.getState().setOverride("toggle-terminal", "mod+shift+`");
    });
    await sleep(150);
    // The NEW combo fires the command…
    await drive(() =>
      window.dispatchEvent(
        new KeyboardEvent("keydown", { key: "`", ctrlKey: true, shiftKey: true, bubbles: true })
      )
    );
    await sleep(250);
    const afterNew = await drive(() => window.__demo.useLayout.getState().terminalOpen);
    check("remapped combo (mod+shift+`) toggles the terminal", afterNew === true);

    // …and the OLD default (mod+`) no longer does.
    await drive(() => window.__demo.useLayout.getState().setTerminalOpen(false));
    await sleep(100);
    await drive(() =>
      window.dispatchEvent(
        new KeyboardEvent("keydown", { key: "`", ctrlKey: true, bubbles: true })
      )
    );
    await sleep(200);
    const afterOld = await drive(() => window.__demo.useLayout.getState().terminalOpen);
    check("stale default combo (mod+`) no longer toggles it", afterOld === false);
    await shot("1-remap");

    // ---- 2. Cheat-sheet reflects the effective combo ----
    await drive(() => window.__demo.useLayout.getState().setShortcutsOpen(true));
    await sleep(400);
    const cheat = await drive(() => document.body.innerText);
    check(
      "cheat-sheet shows the remapped Ctrl+Shift+` for Toggle Terminal",
      cheat.includes("Ctrl+Shift+`")
    );
    check(
      "cheat-sheet lists the new file-browser keys (Rename, Quick info)",
      /Rename/i.test(cheat) && /Quick info/i.test(cheat)
    );
    await shot("2-cheatsheet");
    await drive(() => window.__demo.useLayout.getState().setShortcutsOpen(false));
    await sleep(200);

    // ---- 3. F2 starts a rename in the file browser (bare key) ----
    // Click a file row so the pane has focus + an anchor, then press F2.
    const clicked = await drive(() => {
      const rows = [...document.querySelectorAll('[role="option"]')];
      const row = rows.find((r) => r.innerText.includes("server.js"));
      if (row) row.click();
      return !!row;
    });
    check("found a file row to select", clicked);
    await sleep(150);
    await page.keyboard.press("F2");
    await sleep(300);
    const renaming = await drive(() => {
      const txt = document.body.innerText;
      const input = document.querySelector('[role="dialog"] input');
      return { txt, hasInput: !!input, val: input?.value ?? "" };
    });
    check(
      "F2 opens the rename dialog for the selected file",
      /Rename/i.test(renaming.txt) && renaming.hasInput && renaming.val.includes("server.js"),
      `hasInput=${renaming.hasInput} val=${renaming.val}`
    );
    await shot("3-rename");
    // Close the rename dialog.
    await page.keyboard.press("Escape");
    await sleep(200);

    // ---- 4. F2 does NOTHING while typing in an input ----
    await drive(() => {
      const inp = document.querySelector('input[placeholder="Filter"]');
      if (inp) inp.focus();
    });
    await page.keyboard.type("se");
    await page.keyboard.press("F2");
    await sleep(250);
    const whileTyping = await drive(() => {
      const dlg = document.querySelector('[role="dialog"] input');
      return !dlg; // no rename dialog should have opened
    });
    check("F2 is inert while typing in the filter box", whileTyping);
    // Clear the filter so it doesn't affect later steps.
    await drive(() => {
      const inp = document.querySelector('input[placeholder="Filter"]');
      if (inp) {
        const setter = Object.getOwnPropertyDescriptor(
          window.HTMLInputElement.prototype,
          "value"
        ).set;
        setter.call(inp, "");
        inp.dispatchEvent(new Event("input", { bubbles: true }));
      }
    });
    await sleep(150);

    // ---- 5. Settings → Keyboard: conflict detection rejects a taken combo ----
    await drive(() => window.__demo.useLayout.getState().openDialog("settings"));
    await sleep(400);
    // Switch to the Keyboard tab.
    const onTab = await drive(() => {
      const btn = [...document.querySelectorAll("nav button")].find(
        (b) => b.innerText.trim() === "Keyboard"
      );
      if (btn) btn.click();
      return !!btn;
    });
    check("opened the Settings → Keyboard tab", onTab);
    await sleep(300);
    // Click the record field on the "Reload Window" row, located by its current
    // combo label (Ctrl+R).
    const recording = await drive(() => {
      const recBtns = [...document.querySelectorAll("button")].filter((b) =>
        /Ctrl\+R/.test(b.innerText)
      );
      if (recBtns[0]) recBtns[0].click();
      return !!recBtns[0];
    });
    check("clicked the record field for Reload Window (Ctrl+R)", recording);
    await sleep(200);
    // Press Ctrl+T — already bound to Toggle Transfer Panel → conflict.
    await page.keyboard.down("Control");
    await page.keyboard.press("t");
    await page.keyboard.up("Control");
    await sleep(300);
    const conflictTxt = await drive(() => document.body.innerText);
    check(
      "recording a taken combo (Ctrl+T) surfaces a conflict",
      /already bound to/i.test(conflictTxt) && /Transfer Panel/i.test(conflictTxt)
    );
    await shot("5-conflict");

    // ---- 6. Reset all clears the override we set earlier ----
    await drive(() => window.__demo.useBindings.getState().resetAll());
    await sleep(150);
    const resetOk = await drive(
      () => Object.keys(window.__demo.useBindings.getState().overrides).length === 0
    );
    check("reset-all clears every override", resetOk);

    if (failures === 0) console.log("\n✅ all keyboard-shortcut checks passed");
    else console.log(`\n❌ ${failures} check(s) failed`);
  } finally {
    if (browser) await browser.close();
    killTree(server);
  }
  process.exit(failures === 0 ? 0 : 1);
}

main().catch((e) => {
  console.error("verify crashed:", e);
  process.exit(3);
});
