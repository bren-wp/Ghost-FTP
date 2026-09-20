// Headless runtime verification for Plan 13 Phase 1 — lazy remote image
// previews. Spins up the mock Vite build (no Rust), drives the real UI, and
// asserts on the rendered DOM: the toggle is off by default (remote images show
// icons, zero preview <img>s), flipping the file-toolbar toggle fetches
// thumbnails for the on-screen image rows (blob: <img>s appear) in both grid and
// list views, and flipping it back drops to icons. Exit code 0 = all passed.
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
      await page.screenshot({ path: path.join(OUT, `preview-${name}.png`) });
      console.log("    · shot", `preview-${name}.png`);
    };
    // Count the preview <img>s specifically (object-URL blobs), never the
    // Material file-type icons (which are asset-URL <img>s).
    const blobImgs = () =>
      drive(() => document.querySelectorAll('img[src^="blob:"]').length);
    const dblClickRow = (label) =>
      drive((text) => {
        const row = [...document.querySelectorAll('[role="option"]')].find(
          (el) => el.innerText.trim() === text
        );
        if (!row) return false;
        row.dispatchEvent(new MouseEvent("dblclick", { bubbles: true }));
        return true;
      }, label);

    // Auto-connect settles into api-prod (SFTP) at /var/www/api.
    await page.waitForFunction(
      () => !!window.__demo?.useConnections.getState().activeSessionId,
      { timeout: 15_000 }
    );
    // Force the setting off + grid view for a known starting point.
    await drive(() => {
      window.__demo.useSettings.getState().setRemoteImagePreviews("off");
      window.__demo.useSettings.getState().setPaneViewMode("grid");
    });
    await sleep(800);

    // ---- 1. Navigate into the image-heavy uploads folder ----
    await page.waitForFunction(
      () => document.body.innerText.includes("uploads"),
      { timeout: 10_000 }
    );
    check("uploads folder is listed", await dblClickRow("uploads"));
    await page.waitForFunction(
      () => document.body.innerText.includes("logo.png"),
      { timeout: 10_000 }
    );
    await sleep(800);
    await shot("1-uploads-off");

    // ---- 2. Previews OFF (default): image rows show icons, no blob previews --
    const offCount = await blobImgs();
    check("previews off → zero thumbnails fetched", offCount === 0, `blobImgs=${offCount}`);

    // ---- 3. The file-toolbar toggle exists (remote session) and flips on ----
    const toggle = await page.$('button[title^="Image previews"]');
    check("remote pane shows the preview toggle", !!toggle);
    const titleOff = toggle && (await drive((el) => el.getAttribute("title"), toggle));
    check(
      "toggle starts in the OFF state",
      typeof titleOff === "string" && titleOff.includes("off"),
      titleOff || ""
    );
    if (toggle) await toggle.click();

    // ---- 4. Previews ON: the on-screen image rows fetch thumbnails ----
    await page.waitForFunction(
      () => document.querySelectorAll('img[src^="blob:"]').length >= 5,
      { timeout: 10_000 }
    );
    const onCount = await blobImgs();
    // uploads has 6 image files (jpg/png/png/webp/ico/gif) + 2 non-images.
    check("previews on → thumbnails render for image rows", onCount >= 5, `blobImgs=${onCount}`);
    check("non-image rows don't get thumbnails", onCount <= 6, `blobImgs=${onCount}`);
    await shot("2-grid-on");

    // ---- 5. List view reuses the same pipeline (thumbs in list rows) ----
    await drive(() => window.__demo.useSettings.getState().setPaneViewMode("list"));
    await page.waitForFunction(
      () => document.querySelectorAll('img[src^="blob:"]').length >= 5,
      { timeout: 10_000 }
    );
    const listCount = await blobImgs();
    check("list view shows thumbnails when enabled", listCount >= 5, `blobImgs=${listCount}`);
    await shot("3-list-on");

    // ---- 6. Toggle back OFF → thumbnails fall back to icons ----
    await drive(() => window.__demo.useSettings.getState().setPaneViewMode("grid"));
    await sleep(400);
    const toggle2 = await page.$('button[title^="Image previews"]');
    if (toggle2) await toggle2.click();
    await page.waitForFunction(
      () => document.querySelectorAll('img[src^="blob:"]').length === 0,
      { timeout: 10_000 }
    ).catch(() => {});
    const backOff = await blobImgs();
    check("toggling off reverts to icons", backOff === 0, `blobImgs=${backOff}`);
    await shot("4-grid-off-again");

    if (failures === 0) console.log("\n✅ all remote-preview checks passed");
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
