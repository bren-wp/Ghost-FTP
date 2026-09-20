// Headless runtime verification for Plan 16 Phase 1/2 (in-app updater UX).
// Boots the mock build, opens Settings → About, and drives the updater card
// through: up-to-date → available → download (progress) → ready → restart,
// asserting both the rendered DOM and the mock plugin side-effects
// (__installed / __relaunched). Exit code 0 = all checks passed.
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
    const shot = async (name) => {
      await page.screenshot({ path: path.join(OUT, `verify-updater-${name}.png`) });
      console.log("    · shot", `verify-updater-${name}.png`);
    };
    const clickButton = (label) =>
      drive((l) => {
        const btn = [...document.querySelectorAll("button")].find(
          (b) => b.innerText.trim() === l
        );
        if (btn) btn.click();
        return !!btn;
      }, label);
    const text = () => drive(() => document.body.innerText);

    let booted = false;
    for (let attempt = 0; attempt < 5 && !booted; attempt++) {
      await page.goto(URL, { waitUntil: "networkidle0" });
      try {
        await page.waitForFunction(() => !!window.__demo?.useUpdater, { timeout: 8_000 });
        booted = true;
      } catch {
        console.log(`    · boot attempt ${attempt + 1} missed __demo, reloading…`);
        await sleep(1500);
      }
    }
    if (!booted) throw new Error("app never exposed window.__demo.useUpdater");
    await sleep(400);

    // Open Settings → About.
    await drive(() => window.__demo.useLayout.getState().openDialog("settings"));
    await sleep(300);
    const onTab = await drive(() => {
      const btn = [...document.querySelectorAll("nav button")].find(
        (b) => b.innerText.trim() === "About"
      );
      if (btn) btn.click();
      return !!btn;
    });
    check("opened Settings → About", onTab);
    await sleep(300);

    // ---- 1. Up to date: no mock update → "latest version" ----
    await drive(() => { delete window.__mockUpdate; });
    check("idle state offers a 'Check for updates' button", /Check for updates/.test(await text()));
    await clickButton("Check for updates");
    await sleep(400);
    check("no update available → 'latest version'", /latest version/i.test(await text()));
    await shot("1-uptodate");

    // ---- 2. Update available ----
    await drive(() => {
      window.__mockUpdate = {
        version: "1.4.0",
        currentVersion: "1.3.22",
        body: "Faster transfers and bug fixes.",
      };
    });
    await clickButton("Check for updates");
    await sleep(400);
    let t = await text();
    check("update available → shows the offered version", /Ghost FTP 1\.4\.0 is available/.test(t));
    check("available → offers Download & install", /Download & install/.test(t));
    const storeAvail = await drive(() => window.__demo.useUpdater.getState().status);
    check("store status is 'available'", storeAvail === "available", storeAvail);
    await shot("2-available");

    // ---- 3. Download & install → ready ----
    await clickButton("Download & install");
    await sleep(500);
    t = await text();
    const st = await drive(() => window.__demo.useUpdater.getState().status);
    check("download completed → status 'ready'", st === "ready", st);
    check("ready → prompts to restart", /restart Ghost FTP/i.test(t));
    const installed = await drive(() => !!window.__installed);
    check("mock artifact was installed", installed);
    await shot("3-ready");

    // ---- 4. Restart applies the update ----
    await clickButton("Restart now");
    await sleep(300);
    const relaunched = await drive(() => !!window.__relaunched);
    check("Restart now calls the process relaunch", relaunched);

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
