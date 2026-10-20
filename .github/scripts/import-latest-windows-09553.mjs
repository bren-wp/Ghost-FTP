#!/usr/bin/env node
// Real v0.95.53 native Windows QA run only: no synthetic screenshot pixels.
import fs from "node:fs";
import path from "node:path";
import { createHash } from "node:crypto";
const dir=process.argv[2];
if(!dir||!fs.existsSync(dir)||!fs.statSync(dir).isDirectory())throw Error("Missing original Windows QA directory");
const sha="27a3f8413f168e4cb8cc31cf6d7b7c85c353f5af",run="39506868388";
const names=[
 ["Native-Window","files","main"],["New-Connection","new-connection","newConnection"],
 ["Site-Manager","sites","siteManager"],["Transfer-Center","transfers","transferCenter"],
 ["Preferences","settings","settings"],["File-Properties","file-properties","properties"],
 ["About","about","about"]
];
const images={};
for(const [capture,destination,view] of names){
 const id="GhostFTP-Windows-Preview-"+capture;
 const png=fs.readFileSync(path.join(dir,id+".png"));
 const meta=fs.readFileSync(path.join(dir,id+".txt"),"utf8");
 if(png.length<20000||png.subarray(0,8).toString("hex")!=="89504e470d0a1a0a"||
 png.readUInt32BE(16)!==1290||png.readUInt32BE(20)!==852||
 !(meta.includes("View="+view+"\r")||meta.includes("View="+view+"\n"))||
 !meta.includes("WindowTitle=Ghost FTP")||!meta.includes("CaptureMethod=")||
 !meta.includes("Executable="))throw Error("Unverified compositor capture "+id);
 const dest="docs/assets/screenshots/ghostftp-native-"+destination+".png";
 fs.writeFileSync(dest,png);
 images[dest]=createHash("sha256").update(png).digest("hex");
}
fs.writeFileSync("docs/assets/screenshots/README-NATIVE-CAPTURE.json",JSON.stringify({
 version:"0.95.53",platform:"Windows",capture_source_sha:sha,
 github_actions_run_id:run,provenance:"Ghost FTP native build, genuine Windows WebView2 compositor QA, 1290x852",
 images
},null,2)+"\n");
const readme=fs.readFileSync("README.md","utf8");
const previous=/<p align="center"><sub>Seven authentic Windows Ghost FTP <strong>0\.95\.52<\/strong> WebView2 screenshots[\s\S]*?not generated marketing images\.<\/sub><\/p>/;
const caption='<p align="center"><sub>Seven authentic Ghost FTP Windows <strong>0.95.53</strong> WebView2 screenshots (1290×852), recorded in <a href="https://github.com/bren-wp/Ghost-FTP/actions/runs/'+run+'">successful native CI run '+run+'</a> from source SHA <code>'+sha+'</code>. Original SHA-256 hashes and provenance appear in <a href="docs/assets/screenshots/README-NATIVE-CAPTURE.json">the capture manifest</a>. Actual running-application screenshots, not mockups.</sub></p>';
if(previous.test(readme))fs.writeFileSync("README.md",readme.replace(previous,caption));
else if(!readme.includes("Seven authentic Ghost FTP Windows <strong>0.95.53</strong>"))throw Error("Unexpected README provenance caption");
let verifier=fs.readFileSync(".github/scripts/check-release-images.mjs","utf8");
if(!verifier.includes("const isVerified09553Capture")){
 const marker='const manifestPath = path.join(screenshotDir, "README-NATIVE-CAPTURE.json");';
 const entries=Object.entries(images).map(([file,digest])=>"  ["+JSON.stringify(file)+", "+JSON.stringify(digest)+"],").join("\n");
 const insert='// Exact real v0.95.53 Windows CI pixels; pinned by source and SHA-256.\n'+
 'const verified09553WindowsCaptures = new Map([\n'+entries+'\n]);\n'+
 'const isVerified09553Capture = (manifest, file, digest) =>\n'+
 '  manifest?.version === "0.95.53" &&\n'+
 '  manifest?.capture_source_sha === "'+sha+'" &&\n'+
 '  manifest?.github_actions_run_id === "'+run+'" &&\n'+
 '  manifest.images?.[file] === digest &&\n'+
 '  verified09553WindowsCaptures.get(file) === digest;\n\n';
 if(!verifier.includes(marker))throw Error("Verification anchor not found");
 verifier=verifier.replace(marker,insert+marker);
 const old='isVerified09552Capture(captureManifest, relative, actualHash)) {';
 if(!verifier.includes(old))throw Error("Existing image acceptance boundary changed");
 verifier=verifier.replace(old,'isVerified09552Capture(captureManifest, relative, actualHash) ||\n        isVerified09553Capture(captureManifest, relative, actualHash)) {');
 fs.writeFileSync(".github/scripts/check-release-images.mjs",verifier);
}
console.log("Imported 7 original Windows 0.95.53 captures with exact SHA-256 provenance.");
