import assert from "node:assert/strict";
import "./check-code-search.mjs";
import fs from "node:fs";
import ts from "typescript";
const raw=fs.readFileSync(new URL("../src/lib/codeSyntax.ts",import.meta.url),"utf8");
const js=ts.transpileModule(raw,{compilerOptions:{module:ts.ModuleKind.ESNext,target:ts.ScriptTarget.ES2022}}).outputText;
const {languageForFile,tokenizeCode}=await import("data:text/javascript;base64,"+Buffer.from(js).toString("base64"));
for(const [path,lang] of [["code.php","php"],["code.js","javascript"],["index.html","html"],["site.css","css"],["README.txt","text"]])assert.equal(languageForFile(path),lang);
for(const language of ["php","javascript","html","css","text"]){
 const s='<script> alert("&"); </script> // comment';
 assert.equal(tokenizeCode(s,language).map(t=>t.text).join(""),s);
}
console.log("PHP, JS, HTML and CSS highlighting preserves untrusted source text.");
