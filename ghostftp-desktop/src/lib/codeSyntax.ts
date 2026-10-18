// Tokenize untrusted source without HTML injection. React escapes text nodes.
export type Language = "php" | "javascript" | "html" | "css" | "text";
export type TokenKind = "plain"|"comment"|"string"|"keyword"|"number"|"tag"|"selector";
export interface Token { text:string; kind:TokenKind; }
export function languageForFile(file:string):Language {
  const ext=file.split(/[\\/]/).pop()?.split(".").pop()?.toLowerCase();
  if(ext==="php"||ext==="phtml")return "php";
  if(["js","jsx","ts","tsx","mjs","cjs","json"].includes(ext??""))return "javascript";
  if(["html","htm","xml","svg"].includes(ext??""))return "html";
  if(["css","scss","sass"].includes(ext??""))return "css";
  return "text";
}
const code=/\/\*[\s\S]*?\*\/|\/\/[^\n]*|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|\x60(?:\\.|[^\x60\\])*\x60|\b(?:function|class|const|let|var|return|if|else|while|for|foreach|switch|case|try|catch|throw|new|async|await|import|export|default|public|private|protected|static|interface|type|extends|echo|require|include|namespace|use|true|false|null|undefined)\b|\b\d+(?:\.\d+)?\b/g;
const css=/\/\*[\s\S]*?\*\/|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|@[a-zA-Z-]+|#[0-9a-fA-F]{3,8}\b|[.#][a-zA-Z_][\w-]*|\b\d+(?:\.\d+)?\b/g;
const html=/<!--[\s\S]*?-->|<\/?[\w:-]+|\/?>|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'/g;
export function tokenizeCode(text:string,lang:Language):Token[]{
  if(lang==="text")return [{text,kind:"plain"}];
  const regex=new RegExp((lang==="css"?css:lang==="html"?html:code).source,"g");
  const out:Token[]=[];let start=0;
  for(const match of text.matchAll(regex)){
    const at=match.index;
    if(at>start)out.push({text:text.slice(start,at),kind:"plain"});
    const word=match[0];let kind:TokenKind="keyword";
    if(word.startsWith("/*")||word.startsWith("//")||word.startsWith("<!--"))kind="comment";
    else if(/^(?:["'\x60])/.test(word))kind="string";
    else if(lang==="html")kind="tag";
    else if(lang==="css"&&/^[.#@]/.test(word))kind="selector";
    else if(/^\d/.test(word))kind="number";
    out.push({text:word,kind});start=at+word.length;
  }
  if(start<text.length)out.push({text:text.slice(start),kind:"plain"});
  return out.length?out:[{text,kind:"plain"}];
}
