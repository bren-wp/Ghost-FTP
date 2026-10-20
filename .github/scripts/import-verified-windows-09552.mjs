#!/usr/bin/env node
// One-time native capture import: exact successful 0.95.52 release build only.
import fs from "node:fs";
import path from "node:path";
import { createHash } from "node:crypto";
const dir = process.argv[2];
if (!dir || !fs.statSync(dir).isDirectory()) throw new Error("missing native QA directory");
const sha="b2b90a029a7217aa269c42f293e0d322531c8387";
const run="39493900331";
const records=[
  ["Native-Window","files","main","98bb6986291ae4f032f1f37ee06a8d503b1031d974cd33e2f01a7e72b3c6a210"],
  ["New-Connection","new-connection","newConnection","60e398206ad1b901a832139cc450591d972312baacf39c79751313f9bac563bf"],
  ["Site-Manager","sites","siteManager","9f99f623caa15b18a8a326614ad7df8b4d727bc60ce57cc82f65bf2584ea8e5b"],
  ["Transfer-Center","transfers","transferCenter","c172359036bbf2eeba6cdebea2e69f2e34c71b23b84d186d61d84645a9deec9d"],
  ["Preferences","settings","settings","c81c86324929cfa5aa96da006e4b06b0e9c9644033fabc3119e33df46487e584"],
  ["File-Properties","file-properties","properties","e21a02d7a513d66df28182a2555d254e28d5dab371a28a1c020e7b03254a8af4"],
  ["About","about","about","996812685bcccca68f2fb7f264aef02d91afbae5cc772530886db76ce9f30fa8"],
];
const images={};
for(const [name,stem,view,want] of records){
 const n="GhostFTP-Windows-Preview-"+name;
 const png=fs.readFileSync(path.join(dir,n+".png"));
 const text=fs.readFileSync(path.join(dir,n+".txt"),"utf8");
 const got=createHash("sha256").update(png).digest("hex");
 if(got!==want||png.length<20000||png.subarray(0,8).toString("hex")!=="89504e470d0a1a0a"||
 png.readUInt32BE(16)!==1290||png.readUInt32BE(20)!==852||
 !text.includes("View="+view+"\r")&&!text.includes("View="+view+"\n")||
 !text.includes("WindowTitle=Ghost FTP")||!text.includes("CaptureMethod=")||!text.includes("Executable=")){
   throw new Error("Native QA provenance mismatch "+n);
 }
 const output="docs/assets/screenshots/ghostftp-native-"+stem+".png";
 fs.writeFileSync(output,png);
 images[output]=want;
}
fs.writeFileSync("docs/assets/screenshots/README-NATIVE-CAPTURE.json",JSON.stringify({
 version:"0.95.52",platform:"Windows",capture_source_sha:sha,
 github_actions_run_id:run,
 provenance:"Ghost FTP native build, real Windows WebView2 compositor QA, 1290x852",
 images
},null,2)+"\n");
let readme=fs.readFileSync("README.md","utf8");
const old=/<p align="center"><sub>Seven authentic Ghost FTP Windows 0\.95\.51 native WebView2 screenshots[\s\S]*?actual running application pixels, not generated UI mockups\.<\/sub><\/p>/;
const updated='<p align="center"><sub>Seven authentic Windows Ghost FTP <strong>0.95.52</strong> WebView2 screenshots (1290×852), captured in <a href="https://github.com/bren-wp/Ghost-FTP/actions/runs/'+run+'">successful native Windows QA run '+run+'</a> from source SHA <code>'+sha+'</code>. Every original PNG is pinned by SHA-256 in <a href="docs/assets/screenshots/README-NATIVE-CAPTURE.json">the capture manifest</a>. Real running application pixels, not generated marketing images.</sub></p>';
if(old.test(readme))readme=readme.replace(old,updated);
else if(!readme.includes("Seven authentic Windows Ghost FTP <strong>0.95.52</strong>"))throw new Error("Unexpected README provenance");
fs.writeFileSync("README.md",readme);
console.log("Verified 7 genuine v0.95.52 screenshots, current latest public release, from source "+sha);
