import { useCallback, useEffect, useMemo, useRef, useState, type KeyboardEvent, type UIEvent } from "react";
import { createPortal } from "react-dom";
import { Code2, Save, Search, X, ChevronUp, ChevronDown, Loader2 } from "lucide-react";
import { ipc } from "@/lib/ipc";
import { useEditor } from "@/stores/editorStore";
import { useDialog } from "@/hooks/useDialog";
import { messageOf } from "@/lib/errors";
import { languageForFile, tokenizeCode, type TokenKind } from "@/lib/codeSyntax";

const MAX_HIGHLIGHT=120*1024;
const MAX_EDIT_BYTES=2*1024*1024;
const colors:Record<TokenKind,string>={
  plain:"text-[#dbeafe]",comment:"text-[#8194ab]",string:"text-[#a4dfb3]",
  keyword:"text-[#b5a2ff]",number:"text-[#f8c47d]",tag:"text-[#83d6f9]",selector:"text-[#8cd6db]"
};
export function BuiltinCodeEditor(){
  const id=useEditor(s=>s.builtinEditId);
  const active=useEditor(s=>id?s.edits[id]:undefined);
  const stop=useEditor(s=>s.stopEditing);
  const [content,setContent]=useState("");
  const [saved,setSaved]=useState("");
  const [loading,setLoading]=useState(false);
  const [loaded,setLoaded]=useState(false);
  const [error,setError]=useState<string|null>(null);
  const [query,setQuery]=useState("");
  const [pending,setPending]=useState<{content:string;at:number}|null>(null);
  const [writing,setWriting]=useState(false);
  const area=useRef<HTMLTextAreaElement>(null);
  const backdrop=useRef<HTMLPreElement>(null);
  const finder=useRef<HTMLInputElement>(null);
  const panel=useRef<HTMLDivElement>(null);
  const lang=languageForFile(active?.remotePath??"");
  const dirty=content!==saved;
  const tokens=useMemo(()=>content.length<=MAX_HIGHLIGHT?tokenizeCode(content,lang):[],[content,lang]);

  useEffect(()=>{
    if(!id)return;
    let canceled=false;
    setLoading(true);setLoaded(false);setError(null);setContent("");setSaved("");setPending(null);setQuery("");
    void ipc.editorReadText(id).then(text=>{
      if(!canceled){setContent(text);setSaved(text);setLoaded(true);}
    }).catch(err=>{if(!canceled)setError(messageOf(err));})
      .finally(()=>{if(!canceled)setLoading(false);});
    return ()=>{canceled=true;};
  },[id]);

  useEffect(()=>{
    if(pending&&active&&active.lastSavedAt>=pending.at&&active.lastSavedAt>0){
      setSaved(pending.content);setPending(null);
    }
  },[pending,active?.lastSavedAt,active]);

  const close=useCallback(()=>{
    if(!id)return;
    if((dirty||pending||writing)&&!window.confirm("Some changes are unsaved or still uploading. Discard this editing session?"))return;
    void stop(id);
  },[id,dirty,pending,writing,stop]);
  useDialog(panel,{onClose:close});

  const save=useCallback(async()=>{
    if(!id||writing||loading||!loaded||!dirty)return;
    if(new TextEncoder().encode(content).length>MAX_EDIT_BYTES){setError("2 MiB maximum for built-in editing.");return;}
    setWriting(true);setError(null);
    try{
      const at=Date.now();
      await ipc.editorWriteText(id,content);
      setPending({content,at}); // not a confirmed remote save!
    }catch(err){setError(messageOf(err));}finally{setWriting(false);}
  },[id,writing,loading,dirty,content]);

  const find=useCallback((reverse:boolean)=>{
    const el=area.current;if(!el||!query)return;
    const hay=content.toLowerCase(),needle=query.toLowerCase();
    const from=reverse?Math.max(0,el.selectionStart-1):el.selectionEnd;
    let pos=reverse?hay.lastIndexOf(needle,from-1):hay.indexOf(needle,from);
    if(pos<0)pos=reverse?hay.lastIndexOf(needle):hay.indexOf(needle);
    if(pos>=0){el.focus();el.setSelectionRange(pos,pos+query.length);}
  },[content,query]);

  const keydown=useCallback((e:KeyboardEvent<HTMLTextAreaElement>)=>{
    if((e.ctrlKey||e.metaKey)&&e.key.toLowerCase()==="s"){
      e.preventDefault();e.stopPropagation();void save();return;
    }
    if((e.ctrlKey||e.metaKey)&&e.key.toLowerCase()==="f"){
      e.preventDefault();e.stopPropagation();finder.current?.focus();return;
    }
    if(e.ctrlKey||e.metaKey||e.altKey||e.nativeEvent.isComposing)return;
    const el=e.currentTarget,start=el.selectionStart,end=el.selectionEnd;
    const pairs:Record<string,string>={"(":")","[":"]","{":"}",'"':'"',"'":"'"};let insert:string|null=null,offset=0;
    if(e.key==="Tab"){insert="  ";offset=2;}
    else if(Object.prototype.hasOwnProperty.call(pairs,e.key)){
      insert=e.key+content.slice(start,end)+pairs[e.key];offset=1;
    }else if(e.key==="Enter"){
      const prev=content.slice(0,start),indent=/^[ \t]*/.exec(prev.slice(prev.lastIndexOf("\n")+1))?.[0]??"";
      insert="\n"+indent;offset=insert.length;
    }
    if(insert===null)return;
    e.preventDefault();setContent(content.slice(0,start)+insert+content.slice(end));
    requestAnimationFrame(()=>{el.focus();el.setSelectionRange(start+offset,start+offset);});
  },[content,save]);

  const scroll=(e:UIEvent<HTMLTextAreaElement>)=>{
    if(backdrop.current){
      backdrop.current.scrollTop=e.currentTarget.scrollTop;
      backdrop.current.scrollLeft=e.currentTarget.scrollLeft;
    }
  };
  if(!id)return null;
  return createPortal(
    <div className="fixed inset-0 z-[120] flex items-center justify-center bg-black/75 p-2 backdrop-blur-sm sm:p-5">
      <div ref={panel} role="dialog" aria-modal="true" aria-label="Ghost FTP code editor"
        className="flex h-[min(90vh,900px)] w-full max-w-6xl flex-col overflow-hidden rounded-xl border border-[#315277] bg-[#0c192b] shadow-2xl">
        <div className="flex flex-wrap items-center gap-3 border-b border-[#274162] bg-[#102440] px-4 py-3">
          <Code2 size={18} className="text-[#67c5ff]"/>
          <div className="min-w-0 flex-1">
            <div className="truncate text-sm font-semibold text-[#e9f4ff]">{active?.remotePath.split(/[/\\]/).pop()??"Editor"}</div>
            <div className="truncate text-xs text-[#92abc5]">{active?.remotePath}</div>
          </div>
          <span className="rounded bg-[#1c3b59] px-2 py-1 font-mono text-xs text-[#8ed7ff]">{lang.toUpperCase()}</span>
          <button type="button" onClick={()=>void save()} disabled={!loaded||!dirty||writing||loading}
            className="flex items-center gap-1.5 rounded-md bg-[#237bbb] px-3 py-1.5 text-xs font-semibold text-white hover:bg-[#3290d5] disabled:opacity-40">
            {writing?<Loader2 size={14} className="animate-spin"/>:<Save size={14}/>} Save
          </button>
          <button type="button" onClick={close} aria-label="Close code editor" className="rounded p-1.5 text-[#a9c0d6] hover:bg-[#1c3b59]"><X size={18}/></button>
        </div>
        <div className="flex flex-wrap items-center gap-2 border-b border-[#274162] px-3 py-2">
          <Search size={15} className="text-[#8da6c0]"/>
          <input ref={finder} value={query} onChange={e=>setQuery(e.target.value)}
            onKeyDown={e=>{if(e.key==="Enter"){e.preventDefault();find(e.shiftKey);}}}
            placeholder="Find (Ctrl+F)" aria-label="Find in code"
            className="min-w-[130px] max-w-sm flex-1 rounded border border-[#345475] bg-[#152b45] px-2 py-1 text-xs text-[#e6f4ff] focus:border-[#53baff]"/>
          <button type="button" onClick={()=>find(true)} aria-label="Find previous" className="rounded p-1 text-[#b8d2e5] hover:bg-[#1c3b59]"><ChevronUp size={16}/></button>
          <button type="button" onClick={()=>find(false)} aria-label="Find next" className="rounded p-1 text-[#b8d2e5] hover:bg-[#1c3b59]"><ChevronDown size={16}/></button>
          <span className="text-[11px] text-[#8da6c0]">Auto brackets · indentation · Ctrl+S</span>
        </div>
        {error&&<div role="alert" className="bg-red-950/30 px-4 py-2 text-xs text-red-300">{error}</div>}
        {active?.lastError&&<div role="alert" className="bg-amber-950/30 px-4 py-2 text-xs text-amber-200">Remote upload will retry: {active.lastError}</div>}
        <div className="relative min-h-0 flex-1 overflow-hidden bg-[#0a1728]">
          {loading?<div className="p-5 text-sm text-[#9bb9d5]">Loading text…</div>:<>
            <pre ref={backdrop} aria-hidden="true"
              className="pointer-events-none absolute inset-0 m-0 overflow-hidden whitespace-pre p-4 font-mono text-[13px] leading-[22px] text-[#dbeafe]">
              {tokens.length ? tokens.map((token,i)=><span className={colors[token.kind]} key={i}>{token.text}</span>):content}{"\n"}
            </pre>
            <textarea ref={area} value={content} wrap="off" spellCheck={false} disabled={!loaded}
              onChange={e=>setContent(e.target.value)} onKeyDown={keydown} onScroll={scroll}
              aria-label="Code text editor"
              className="absolute inset-0 h-full w-full resize-none overflow-auto whitespace-pre bg-transparent p-4 font-mono text-[13px] leading-[22px] text-transparent caret-[#f6fbff] outline-none selection:bg-[#41698b]/70"
              style={{tabSize:2}}/>
          </>}
        </div>
        <div className="flex items-center justify-between gap-2 border-t border-[#274162] bg-[#102440] px-4 py-2 text-[11px] text-[#9fb9d0]">
          <span>{content.split("\n").length.toLocaleString()} lines · {new TextEncoder().encode(content).length.toLocaleString()} bytes {content.length>MAX_HIGHLIGHT?"· highlight disabled for large files":""}</span>
          <span aria-live="polite">{writing?"Writing local copy…":pending?"Awaiting remote upload…":dirty?"Unsaved local changes":"Saved"}</span>
        </div>
      </div>
    </div>,document.body
  );
}
