import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import {fileURLToPath} from "node:url";
import test from "node:test";
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),"../..");
const read=(file)=>fs.readFileSync(path.join(root,file),"utf8");
const native=read(".github/workflows/ghostftp-build.yml");
const server=read(".github/workflows/ghostftp-server.yml");
const release=read(".github/workflows/ghostftp-release.yml");
const pre=read(".github/scripts/publish-release.sh");
const post=read(".github/scripts/check-release-index.sh");
const quality=read(".github/workflows/ghostftp-quality.yml");
const cli=read("ghostftp-desktop/src-tauri/ghostftp-cli/src/main.rs");
test("Ghost CLI uses the exact existing desktop FTP/FTPS/SFTP engine",()=>{
  assert.ok(cli.includes("ghostftp_lib::session"));
  assert.ok(cli.includes("ghostftp_lib::remotefs"));
  assert.ok(read("ghost-cli/build.sh").includes("cargo build --release --locked -p ghostftp-cli"));
  assert.ok(read("ghost-cli/build.ps1").includes("cargo build --release --locked -p ghostftp-cli"));
  assert.ok(native.includes("cargo build --release --locked -p ghostftp-cli"));
  assert.ok(native.includes("ghostftp-cli.exe"));
  assert.ok(native.includes("ghostftp-cli dist/native/ghostftp-cli"));
});
test("Windows and Linux Ghost CLI binaries are mandatory public assets",()=>{
  for(const item of ["GhostCLI-Windows-x64-v","GhostCLI-Linux-x86_64-v"]){
    assert.ok(release.includes(item),item+": staging");
    assert.ok(pre.includes(item),item+": prepublication");
    assert.ok(post.includes(item),item+": public digest");
  }
});
test("offline Windows demo is built, locally hosted, tested and clearly labeled",()=>{
  const demo=read("ghost-demo/main.go");
  const web=read("ghost-demo/assets/index.html");
  assert.ok(demo.includes('net.Listen("tcp","127.0.0.1:0")'));
  assert.ok(demo.includes('r.Host!=host'));
  assert.ok(demo.includes("SIMULATION ONLY"));
  assert.ok(demo.includes('func selfTest() error'));
  assert.ok(web.includes("SIMULATION • OFFLINE"));
  assert.ok(web.includes("No real file transfers"));
  assert.ok(server.includes("go test -C ../ghost-demo ./..."));
  assert.ok(server.includes("go test -C ghost-demo -race ./..."));
  assert.ok(server.includes("GhostFTP-Demo-Windows-x64-Portable-v$version.exe"));
  assert.ok(release.includes("GhostFTP-Demo-Windows-x64-Portable-v$VERSION.exe"));
  assert.ok(pre.includes("GhostFTP-Demo-Windows-x64-Portable-v" + "$" + "{VERSION}.exe"));
  assert.ok(post.includes("GhostFTP-Demo-Windows-x64-Portable-v$VERSION.exe"));
  for (const product of ["GhostCLI", "GhostFTP-Demo"]) {
    assert.ok(pre.includes(product + "-Linux-x86_64-v"),product + " Linux portable");
    assert.ok(pre.includes(product + "-Linux-amd64-v"),product + " Linux DEB");
    assert.ok(pre.includes(product + "-Windows-x64-Setup-Unsigned-Preview-v"),product + " Windows Setup");
    assert.ok(post.includes(product + "-Linux-x86_64-v"),product + " public Linux package");
  }
  assert.ok(native.includes("Build, install and uninstall separate Ghost CLI Windows Setup"));
  assert.ok(server.includes("Build and smoke independent Ghost FTP Demo Windows NSIS Setup"));
  assert.ok(server.includes("packaging/linux/build-portable-packages.sh demo"));
  assert.ok(native.includes("packaging/linux/build-portable-packages.sh cli"));
});
test("combined release is checksum-complete and still an unsigned Preview",()=>{
  assert.ok(post.includes("-ge 43"));
  assert.ok(quality.includes("check-cli-demo-release.test.mjs"));
  assert.ok(release.includes("sha256sum -c Release-SHA256SUMS.txt"));
  assert.ok(release.includes("sha256sum -c Linux-SHA256SUMS.txt"));
  assert.ok(read("version.json").includes('"channel": "preview"'));
});
