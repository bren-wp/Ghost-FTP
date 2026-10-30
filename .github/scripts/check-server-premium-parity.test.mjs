import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import {fileURLToPath} from "node:url";
import test from "node:test";

const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),"../..");
const read=(relative)=>fs.readFileSync(path.join(root,relative),"utf8");
const palette=JSON.parse(read("docs/branding/premium-ui-tokens.json"));
const css=read("ghost-server/admin-assets/style.css");
const html=read("ghost-server/admin-assets/index.html");
const app=read("ghost-server/admin-assets/app.js");
const api=read("ghost-server/admin.go");
const workflow=read(".github/workflows/ghostftp-server.yml");

test("Ghost Server and actual Windows/Linux UI share the approved design-source palette and geometry",()=>{
  const keys=["background","panel","raised","navy","slate","accent","cyan","text","muted","success","danger"];
  const variables=["bg","panel","raised","navy","slate","accent","cyan","text","muted","success","danger"];
  for(let i=0;i<keys.length;i++) {
    assert.ok(css.includes(`--${variables[i]}: ${palette.colors[keys[i]]};`),keys[i]+" must match immutable brand source");
  }
  assert.ok(css.includes(`--sidebar-width: ${palette.desktop.sidebarWidthPx}px;`));
  assert.ok(css.includes(`--nav-row-height: ${palette.desktop.navRowHeightPx}px;`));
  assert.ok(css.includes(`--topbar-height: ${palette.desktop.topChromeHeightPx}px;`));
  assert.equal(read("ghost-server/admin-assets/ghostftp-symbol.svg"),read("ghostftp-desktop/branding/ghostftp-symbol.svg"));
  assert.ok(css.includes("@media (max-width: 720px)"));
  assert.ok(css.includes("@media (max-width: 440px)"));
  assert.ok(css.includes("prefers-reduced-motion"));
});

test("all server UI sections have real accessible navigation and no fictional management actions",()=>{
  for(const page of ["overview","sessions","security","limits","diagnostics"]){
    assert.ok(html.includes(`data-page="${page}"`),page+" missing clickable nav");
    assert.ok(html.includes(`data-view="${page}"`),page+" missing real section");
    assert.ok(app.includes(page),page+" missing navigation controller");
  }
  for(const key of ["status","clients","uploads","uptime","limit-upload","diagnostic-version"]){
    assert.ok(html.includes(`id="${key}"`),key);
  }
  assert.ok(html.includes('aria-live="polite"'));
  assert.ok(html.includes('id="lock"'));
  assert.ok(html.includes('id="refresh"'));
  assert.ok(app.includes('textContent = String(value)'),"server data must render as text, never injection-capable HTML");
  assert.ok(!app.includes("innerHTML"),"never render server values as HTML");
  assert.ok(!app.includes("localStorage"),"never persist admin credentials");
  assert.ok(!app.includes("sessionStorage"),"never persist admin credentials");
  assert.ok(!app.includes("document.cookie"),"never persist admin credentials");
  assert.ok(!app.includes("new WebSocket("),"UI must use real status API not phantom stream");
  assert.ok(!html.includes('onclick='),"no inline handlers under CSP");
});

test("versioned server status contract remains local, authenticated, cross-platform-safe and release-synchronized",()=>{
  assert.ok(api.includes('mux.HandleFunc("/api/v1/status",statusHandler)'));
  assert.ok(api.includes('mux.HandleFunc("/api/status",statusHandler)'));
  assert.ok(api.includes('ContractVersion:1,ServerVersion:BuildVersion'));
  assert.ok(api.includes('subtle.ConstantTimeCompare'));
  assert.ok(api.includes("img-src 'self'"));
  assert.ok(api.includes("default-src 'none'"));
  assert.ok(api.includes("r.Host!=s.adminListen"));
  assert.ok(app.includes('fetch("/api/v1/status"'));
  assert.ok(app.includes("credentials: \"omit\""));
  assert.ok(app.includes("Number.isSafeInteger(data.active_connections)"));
  assert.ok(app.includes("clearDisplayedState();"),"stale live metrics must clear on network error");
  assert.ok(app.includes('window.addEventListener("pagehide"'));
  assert.ok(app.includes("controller.abort()") || app.includes("activeRequest.abort()"));
  assert.equal((workflow.match(/-X ghostftp\/server\.BuildVersion/g)||[]).length,4);
});
