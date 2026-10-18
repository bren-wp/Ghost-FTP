import assert from "node:assert/strict";
import fs from "node:fs";
import ts from "typescript";
const source = fs.readFileSync(new URL("../src/lib/syncScripts.ts", import.meta.url), "utf8");
const js = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
}).outputText;
const { generateSyncScript } = await import("data:text/javascript;base64," + Buffer.from(js).toString("base64"));
const pair={ name:"Site",profileId:"8d7b9e",localRoot:"C:\\Projects\\my app",remoteRoot:"/public_html",
  direction:"localToRemote",strategy:"additive",mode:"mirror",exclude:[] };
const ps=generateSyncScript(pair,"powershell");
assert.match(ps,/ghostftp-cli/);
assert.match(ps,/--direction \$direction/);
assert.ok(!ps.includes("SUPER_SECRET_TEST"));
const cmd=generateSyncScript(pair,"cmd");
assert.match(cmd,/setlocal DisableDelayedExpansion/);
assert.match(cmd,/ghostftp-cli\.exe sync "C:\\Projects\\my app" "8d7b9e:\/public_html"/);
assert.match(generateSyncScript({...pair,localRoot:"C:\\\\"},"cmd"), /"C:\\\\\\\\"/);
const watcher=generateSyncScript(pair,"dotnet-watcher");
assert.match(watcher,/FileSystemWatcher/);
assert.match(watcher,/Wait-Event -Timeout 60/);
assert.match(watcher,/event.SourceIdentifier.StartsWith\(\$prefix\)/);
assert.match(watcher,/Get-Event \| Where-Object/);
assert.match(watcher,/EventName \$name/);
const quoted={...pair,remoteRoot:"/O'Reilly site"};
assert.match(generateSyncScript(quoted,"powershell"),/O''Reilly site/);
assert.throws(()=>generateSyncScript({...pair,localRoot:"C:\\%PATH%"},"cmd"),/PowerShell/);
assert.throws(()=>generateSyncScript({...pair,remoteRoot:"/bad\ninject"},"powershell"),/control-character/);
assert.throws(()=>generateSyncScript({...pair,profileId:"bad;rm"},"powershell"),/Profile ID/);
assert.throws(()=>generateSyncScript({...pair,remoteRoot:"relative"},"powershell"),/absolute POSIX/);
assert.throws(()=>generateSyncScript({...pair,strategy:"mirror"},"cmd"),/Mirror/);
assert.throws(()=>generateSyncScript({...pair,exclude:["node_modules"]},"cmd"),/exclude/);
assert.throws(()=>generateSyncScript({...pair,mode:"onDemand"},"cmd"),/On-demand/);
assert.throws(()=>generateSyncScript({...pair,direction:"remoteToLocal"},"dotnet-watcher"),/local-to-remote/);
console.log("Safe CMD, PowerShell and .NET watcher script contracts passed.");
