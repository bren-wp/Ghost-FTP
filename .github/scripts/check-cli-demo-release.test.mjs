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
test("Ghost FTP Demo is exclusively a Windows portable demo binary in release inventory",()=>{
  const demo=read("ghost-demo/main.go");
  assert.ok(demo.includes('net.Listen("tcp","127.0.0.1:0")'));
  assert.ok(server.includes("go test -C ../ghost-demo ./..."));
  assert.ok(server.includes("GhostFTP-Demo-Windows-x64-Portable-v$version.exe"));
  assert.ok(pre.includes("GhostFTP-Demo-Windows-x64-Portable-v" + "$" + "{VERSION}.exe"));
  assert.ok(post.includes("GhostFTP-Demo-Windows-x64-Portable-v$VERSION.exe"));
  for(const source of [server,release,pre,post]){
    assert.ok(!source.includes("GhostFTP-Demo-Linux-"),"Linux demo must not be packaged");
    assert.ok(!source.includes("GhostFTP-Demo-Windows-x64-Setup-"),"Demo setup must not be packaged");
  }
  assert.ok(!server.includes("build-portable-packages.sh demo"));
  assert.ok(!server.includes("Build and smoke independent Ghost FTP Demo Windows NSIS Setup"));
});
test("Native Windows Server Control Center has independent executable, is not WebView, and is actually installed",()=>{
  const entry=read("ghost-server/cmd/ghost-server/main.go");
  const launcher=read("ghost-server/cmd/ghost-server/controlcenter_windows.go");
  const desktop=read("ghost-server/cmd/ghost-server/control_center.ps1");
  const installer=read("ghost-server/install/GhostFTPServer.nsi");
  assert.ok(entry.includes("runNativeControlCenter()"));
  assert.ok(launcher.includes("go:embed control_center.ps1"));
  assert.ok(desktop.includes("System.Windows.Forms") && desktop.includes("Start-Service"));
  assert.ok(!desktop.includes("WebView2") && !desktop.includes("WebBrowser"));
  assert.ok(installer.includes("GhostFTPServer-ControlCenter.exe"));
  for(const source of [server,release,pre,post])
    assert.ok(source.includes("GhostFTP-Server-Windows-x64-ControlCenter-v") || source.includes("ghost-server-controlcenter.exe"));
});
test("client release selectors cannot accidentally promote CLI/Demo packages as Ghost FTP Client",()=>{
  for(const literal of [
    "GhostFTP-Windows-x64-Setup.exe",
    "GhostFTP-Windows-x64-Setup.msi",
    "GhostFTP-Linux-amd64.deb",
    "GhostFTP-Linux-x86_64.rpm"
  ]) assert.ok(release.includes(" -name '"+literal+"'"),"exact client package selector "+literal);
  assert.ok(!release.includes("-iname '*setup*.exe' ! -iname '*.sig' | head -n1"),"generic Setup selector is unsafe");
  assert.ok(!release.includes("-iname '*.deb' | head -n1"),"generic DEB selector is unsafe");
  assert.ok(!release.includes("-iname '*.rpm' | head -n1"),"generic RPM selector is unsafe");
});
test("release normalizes CLI installers from their real per-platform artifact folders",()=>{
  assert.ok(release.includes("WINDOWS_CLI_SETUP=\"$(find dist/native -type f -name 'GhostCLI-Windows-x64-Setup-Unsigned-Preview.exe'"));
  assert.ok(release.includes('LINUX_CLI_PACKAGE_DIR="$(dirname "$LINUX_CLI")"'));
  assert.ok(release.includes('test -s "$WINDOWS_CLI_SETUP"'));
  assert.ok(release.includes('test -s "$LINUX_CLI_PACKAGE_DIR/$name"'));
  assert.ok(release.includes('cp "$WINDOWS_CLI_SETUP" "dist/GhostCLI-Windows-x64-Setup-Unsigned-Preview-v$VERSION.exe"'));
  assert.ok(release.includes('cp "$LINUX_CLI_PACKAGE_DIR/$name" "dist/$name"'));
  assert.ok(!release.includes('test -s "dist/native/GhostCLI-Windows-x64-Setup-Unsigned-Preview.exe"'));
  assert.ok(!release.includes('cp "dist/native/$name" "dist/$name"'));
});
test("Linux Ghost CLI is smoke-tested with real shared-desktop GTK runtime rather than silently skipped",()=>{
  const pkg=read("packaging/linux/build-portable-packages.sh");
  assert.ok(release.includes("Install genuine Linux Ghost CLI runtime dependencies"));
  assert.ok(release.includes("sudo apt-get install -y --no-install-recommends libwebkit2gtk-4.1-0"));
  assert.ok(release.includes('test "$(dist/GhostCLI-Linux-x86_64-v$VERSION --version | grep -Fc "$VERSION")" -ge 1'));
  assert.ok(pkg.includes('DEB_DEPENDS="libwebkit2gtk-4.1-0, libgtk-3-0, libayatana-appindicator3-1"'));
  assert.ok(pkg.includes("RPM_REQUIRES='Requires: libwebkit2gtk-4.1.so.0()(64bit)'"));
  assert.ok(pkg.includes('if [ -n "$DEB_DEPENDS" ]; then'));
  assert.ok(pkg.includes('sed -i "/^Architecture: amd64$/a Depends: $DEB_DEPENDS"'));
  assert.ok(pkg.includes("$RPM_REQUIRES"));
});
test("Release title is plain product name and version; security classification stays truthful",()=>{
  assert.ok(pre.includes('RELEASE_NAME="Ghost FTP ${VERSION}"'));
  assert.ok(!pre.includes('RELEASE_NAME="Ghost FTP ${VERSION} Preview"'));
  assert.ok(pre.includes('draft:false,prerelease:true'));
});
test("combined release is checksum-complete and still an unsigned Preview",()=>{
  assert.ok(post.includes("-ge 39"));
  assert.ok(quality.includes("check-cli-demo-release.test.mjs"));
  assert.ok(release.includes("sha256sum -c Release-SHA256SUMS.txt"));
  assert.ok(release.includes("sha256sum -c Linux-SHA256SUMS.txt"));
  assert.ok(read("version.json").includes('"channel": "preview"'));
});
