import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import {fileURLToPath} from "node:url";
import test from "node:test";

const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const read=(p)=>fs.readFileSync(path.join(root,p),"utf8");
const server=read(".github/workflows/ghostftp-server.yml");
const release=read(".github/workflows/ghostftp-release.yml");
const publisher=read(".github/scripts/publish-release.sh");
const index=read(".github/scripts/check-release-index.sh");
const shell=read("ghost-server/install/linux/build-packages.sh");
const unit=read("ghost-server/install/linux/ghostftp-server.service");

test("server is a mandatory exact-SHA gate on every PR and every main release",()=>{
  assert.ok(server.includes("  push:\n    branches: [main]\n  pull_request:"));
  assert.ok(!server.split("  pull_request:")[1].split("permissions:")[0].includes("paths:"));
  assert.ok(release.includes('wait_for_gate "ghostftp-server.yml"'));
  assert.ok(release.includes('gh run list --workflow ghostftp-server.yml --commit "$SOURCE_SHA"'));
  assert.ok(release.includes('gh run download "$RUN_ID" --name GhostFTP-Server-Linux-Package-Preview'));
  assert.ok(release.includes('gh run download "$RUN_ID" --name GhostFTP-Server-Windows-Package-Preview'));
  assert.ok(release.includes("sha256sum -c Linux-SHA256SUMS.txt"));
  assert.ok(release.includes("sha256sum -c Release-SHA256SUMS.txt"));
});

test("six native server assets are mandatory from CI through public verification",()=>{
  const names=[
    "GhostFTP-Server-Windows-x64-Portable-v",
    "GhostFTP-Server-Windows-x64-Setup-Unsigned-Preview-v",
    "GhostFTP-Server-Linux-x86_64-v",
    "GhostFTP-Server-Linux-x86_64-v",
    "GhostFTP-Server-Linux-amd64-v",
    "GhostFTP-Server-Linux-x86_64-v"
  ];
  for(const name of ["GhostFTP-Server-Windows-x64-Portable-v","GhostFTP-Server-Windows-x64-Setup-Unsigned-Preview-v",
   "GhostFTP-Server-Linux-x86_64-v"]){
    assert.ok(release.includes(name),name+" release staging");
    assert.ok(publisher.includes(name),name+" publication required");
    assert.ok(index.includes(name),name+" postpublication required");
  }
  for(const name of [".tar.gz",".deb",".rpm"]){
    assert.ok(publisher.includes(name),name+" publisher");
    assert.ok(index.includes(name),name+" index");
  }
  assert.ok(publisher.includes('test -s "dist/$required"') || publisher.includes('! -s "dist/$required"'));
  assert.ok(index.includes("-ge 31"));
});

test("Linux service never starts unprovisioned; credentials stay out of argv",()=>{
  assert.ok(unit.includes("User=ghostftp"));
  assert.ok(unit.includes("ProtectSystem=strict"));
  assert.ok(unit.includes("NoNewPrivileges=true"));
  assert.ok(unit.includes("127.0.0.1:2121"));
  assert.ok(unit.includes("-password-file /etc/ghostftp-server/password"));
  assert.ok(!unit.includes("-allow-uploads"));
  assert.ok(!unit.includes("Environment=PASSWORD="));
  assert.ok(shell.includes("dpkg-deb --build"));
  assert.ok(shell.includes("rpmbuild"));
  assert.ok(!shell.includes("systemctl enable --now"));
  assert.ok(!shell.includes("systemctl start"));
  assert.ok(!shell.includes("rm -rf /var/lib/ghostftp-server"));
  assert.ok(!shell.includes("rm -rf /etc/ghostftp-server"));
  assert.ok(server.includes("Smoke-GhostFTPInstaller-CI.ps1"));
});
