// Headless runtime verification for Plan 11 terminal depth (registry + splits)
// and snippets. Spins up the mock Vite build (no Rust), drives the UI via
// window.__demo, and asserts on the real rendered DOM. Writes PNGs to the
// scratchpad for a visual record. Exit code 0 = all checks passed.
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
      await page.screenshot({ path: path.join(OUT, `verify-${name}.png`) });
      console.log("    · shot", `verify-${name}.png`);
    };

    // Auto-connect settles into an active session.
    await page.waitForFunction(
      () => !!window.__demo?.useConnections.getState().activeSessionId,
      { timeout: 15_000 }
    );
    await sleep(600);

    // ---- 1. Registry: open the terminal, transcript renders ----
    await drive(() => window.__demo.useLayout.getState().setTerminalOpen(true));
    await sleep(1200);
    const t1 = await drive(() => {
      const xterms = [...document.querySelectorAll(".xterm")];
      return { count: xterms.length, text: document.body.innerText };
    });
    check("terminal opens with exactly 1 pane", t1.count === 1, `xterms=${t1.count}`);
    check("pane rendered the shell transcript", t1.text.includes("package.json"));
    await shot("1-open");

    // ---- 2. Splits: split right → 2 live panes, both with content ----
    await drive(() => {
      const t = window.__demo.useTerminals.getState();
      t.splitActivePane(t.activeId, "row");
    });
    await sleep(1200);
    const t2 = await drive(() => {
      const xterms = [...document.querySelectorAll(".xterm")];
      const withText = xterms.filter((x) => x.innerText.includes("package.json"));
      return { count: xterms.length, withText: withText.length };
    });
    check("split creates a 2nd pane", t2.count === 2, `xterms=${t2.count}`);
    check("both panes have their own transcript", t2.withText === 2, `withText=${t2.withText}`);
    await shot("2-split");

    // ---- 3. Zoom: collapses to 1 visible pane; unzoom restores 2 (registry
    //         kept the detached instance alive, content preserved) ----
    await drive(() => {
      const t = window.__demo.useTerminals.getState();
      t.toggleZoom(t.activeId);
    });
    await sleep(500);
    const zoomed = await drive(() => document.querySelectorAll(".xterm").length);
    check("zoom shows a single pane", zoomed === 1, `xterms=${zoomed}`);
    await shot("3-zoom");
    await drive(() => {
      const t = window.__demo.useTerminals.getState();
      t.toggleZoom(t.activeId);
    });
    await sleep(600);
    const unzoomed = await drive(() => {
      const xterms = [...document.querySelectorAll(".xterm")];
      return {
        count: xterms.length,
        withText: xterms.filter((x) => x.innerText.includes("package.json")).length,
      };
    });
    check("unzoom restores both panes", unzoomed.count === 2, `xterms=${unzoomed.count}`);
    check(
      "restored pane kept its scrollback (registry survival)",
      unzoomed.withText === 2,
      `withText=${unzoomed.withText}`
    );

    // ---- 4. Dock toggle: hide + reshow, scrollback persists ----
    await drive(() => window.__demo.useLayout.getState().setTerminalOpen(false));
    await sleep(400);
    await drive(() => window.__demo.useLayout.getState().setTerminalOpen(true));
    await sleep(700);
    const reopened = await drive(() => {
      const xterms = [...document.querySelectorAll(".xterm")];
      return {
        count: xterms.length,
        withText: xterms.filter((x) => x.innerText.includes("package.json")).length,
      };
    });
    check("dock reopen keeps both panes", reopened.count === 2, `xterms=${reopened.count}`);
    check(
      "scrollback survived a dock toggle",
      reopened.withText === 2,
      `withText=${reopened.withText}`
    );
    await shot("4-reopen");

    // ---- 5. Snippets panel renders the canned list ----
    await drive(() => window.__demo.useSnippets.getState().openPanel());
    await sleep(500);
    const snip = await drive(() => document.body.innerText);
    check(
      "Snippets panel lists canned snippets",
      snip.includes("Tail nginx errors") && snip.includes("WP core update")
    );
    await shot("5-snippets");
    await drive(() => window.__demo.useSnippets.getState().closePanel());

    // ---- 6. Variable dialog: inserting a {{site}} snippet prompts for it ----
    await sleep(200);
    await drive(() => {
      const s = window.__demo.useSnippets
        .getState()
        .snippets.find((x) => x.body.includes("{{"));
      if (s) window.__demo.useSnippets.getState().requestInsert(s);
    });
    await sleep(400);
    const dlg = await drive(() => document.body.innerText.toLowerCase());
    check(
      "variable snippet opens the fill-in dialog",
      dlg.includes("insert snippet") && dlg.includes("preview") && dlg.includes("site")
    );
    await shot("6-var-dialog");

    if (failures === 0) console.log("\n✅ all terminal + snippet checks passed");
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
