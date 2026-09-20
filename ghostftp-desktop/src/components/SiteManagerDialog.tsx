import { useMemo, useRef, useState } from "react";
import { Download, Edit3, Folder, Search, Server, Star, Tag, Upload, Plus, Link2, RadioTower, Save, Trash2 } from "lucide-react";
import { useConnections } from "@/stores/connectionsStore";
import { useLayout } from "@/stores/layoutStore";
import type { ConnectionProfile, Protocol } from "@/lib/types";
import { PROTOCOL_DEFAULT_PORT } from "@/lib/types";
import { ReferenceMenuRow, ReferenceWindowTitlebar } from "./ReferenceWindowChrome";

interface Props { onClose: () => void }

type View = "all" | "favorites" | "recent" | "bookmarks" | string;

export function SiteManagerDialog({ onClose }: Props) {
  const panelRef = useRef<HTMLDivElement>(null);
  const allProfiles = useConnections((s)=>s.profiles);
  const connect = useConnections((s)=>s.connect);
  const sessions = useConnections((s)=>s.sessions);
  const ephemeralIds = useMemo(() => new Set(sessions.filter((session) => session.ephemeral).map((session) => session.profileId)), [sessions]);
  const profiles = useMemo(() => allProfiles.filter((profile) => !ephemeralIds.has(profile.id)), [allProfiles, ephemeralIds]);
  const saveProfile = useConnections((s)=>s.saveProfile);
  const deleteProfile = useConnections((s)=>s.deleteProfile);
  const disconnect = useConnections((s)=>s.disconnect);
  const openNewConnection = useLayout((s)=>s.openNewConnection);
  const openDialog = useLayout((s)=>s.openDialog);
  const [query,setQuery]=useState("");
  const [selectedId,setSelectedId]=useState<string | null>(profiles[0]?.id ?? null);
  const [view,setView]=useState<View>("all");
  const [editing,setEditing]=useState(false);
  const selected = profiles.find(p=>p.id===selectedId) ?? profiles[0] ?? null;
  const [draft,setDraft]=useState<ConnectionProfile | null>(selected ? {...selected} : null);

  const folders=useMemo(()=>Array.from(new Set(profiles.map(p=>p.group).filter(Boolean) as string[])).sort(),[profiles]);
  const tags=useMemo(()=>Array.from(new Set(profiles.flatMap(p=>p.tags??[]))).sort(),[profiles]);
  const filtered=useMemo(()=>profiles.filter(p=>{
    const q=`${p.name} ${p.host} ${p.protocol} ${p.group??''} ${(p.tags??[]).join(' ')}`.toLowerCase();
    if(query && !q.includes(query.toLowerCase())) return false;
    if(view==="favorites" && !p.favorite) return false;
    if(view==="recent" && !p.lastUsed) return false;
    if(view==="bookmarks" && !p.bookmarked) return false;
    if(view.startsWith("tag:") && !(p.tags??[]).includes(view.slice(4))) return false;
    if(view.startsWith("folder:") && (p.group||"")!==view.slice(7)) return false;
    return true;
  }).sort((a,b)=> view==="recent" ? (b.lastUsed??0)-(a.lastUsed??0) : a.name.localeCompare(b.name)),[profiles,query,view]);
  const isConnected = selected ? sessions.some(s=>s.profileId===selected.id) : false;

  const select = (id:string) => {
    setSelectedId(id);
    const p=profiles.find(x=>x.id===id) ?? null;
    setDraft(p ? {...p} : null);
    setEditing(false);
  };
  const startEdit = () => {
    if(!selected) return;
    setDraft({...selected});
    setEditing(true);
  };
  const save = async() => {
    if(!draft) return;
    await saveProfile(draft);
    setEditing(false);
  };
  const remove = async() => {
    if(!selected) return;
    await deleteProfile(selected.id);
    setSelectedId(null);
    setDraft(null);
  };
  const exportSites = () => {
    const blob=new Blob([JSON.stringify(profiles,null,2)],{type:"application/json"});
    const a=document.createElement("a"); a.href=URL.createObjectURL(blob); a.download="ghostftp-sites.json"; a.click(); URL.revokeObjectURL(a.href);
  };
  const testSelected = async () => {
    if(!selected) return;
    const existing = useConnections.getState().sessions.find((s)=>s.profileId===selected.id);
    if(existing){
      await connect(selected.id);
      return;
    }
    await connect(selected.id);
    const created = useConnections.getState().sessions.find((s)=>s.profileId===selected.id);
    if(created) await disconnect(created.sessionId);
  };

  return <div className="ghost-standalone-view fixed inset-0 z-modal bg-[#041425]" role="dialog" aria-modal="true">
    <div ref={panelRef} className="ghost-site-manager flex h-full w-full flex-col overflow-hidden bg-[#061a2d]">
      <ReferenceWindowTitlebar />
      <ReferenceMenuRow />
      <div className="ghost-site-manager-hero flex items-center gap-3 border-b border-border px-5">
        <div className="ghost-dialog-icon"><Server size={24}/></div>
        <div><div className="text-xl font-semibold">Site Manager</div><div className="text-[12px] text-text-muted">Manage your saved connections, bookmarks and server profiles.</div></div>
        <div className="flex-1"/>
        <div className="relative"><Search size={14} className="absolute left-3 top-2.5 text-text-dim"/><input value={query} onChange={e=>setQuery(e.target.value)} className="ghost-ref-input w-64 pl-9" placeholder="Search sites…"/></div>
        <button className="ghost-mini-button" onClick={()=>openDialog("import")}><Download size={14}/> Import</button>
        <button className="ghost-mini-button" onClick={exportSites}><Upload size={14}/> Export</button>
        <button className="ghost-primary-button" onClick={()=>openNewConnection()}><Plus size={15}/> New Site</button>
      </div>
      <div className="grid min-h-0 flex-1 grid-cols-[242px_minmax(0,1fr)_356px] gap-0">
        <aside className="ghost-site-manager-nav border-r border-border bg-[#051929] p-3">
          <SideItem active={view==="all"} icon={<Server/>} label="All Sites" count={profiles.length} onClick={()=>setView("all")}/>
          <SideItem active={view==="favorites"} icon={<Star/>} label="Favorites" count={profiles.filter((p)=>p.favorite).length} onClick={()=>setView("favorites")}/>
          <SideItem active={view==="recent"} icon={<RadioTower/>} label="Recent Servers" count={profiles.filter((p)=>p.lastUsed).length} onClick={()=>setView("recent")}/>
          <SideItem active={view==="bookmarks"} icon={<Folder/>} label="Bookmarks" count={profiles.filter((p)=>p.bookmarked).length} onClick={()=>setView("bookmarks")}/>
          <div className="my-3 border-t border-border"/>
          <div className="mb-2 flex items-center justify-between px-2 text-[11px] font-semibold text-accent"><span>Tags</span><Tag size={13}/></div>
          {tags.length === 0 && <div className="px-2 py-2 text-[11px] text-text-dim">No tags yet</div>}
          {tags.map((tag)=><SideItem key={tag} active={view===`tag:${tag}`} icon={<Tag/>} label={tag} count={profiles.filter(p=>(p.tags??[]).includes(tag)).length} onClick={()=>setView(`tag:${tag}`)}/>) }
          <div className="my-3 border-t border-border"/>
          <div className="mb-2 flex items-center justify-between px-2 text-[11px] font-semibold text-accent"><span>Folders</span><Folder size={13}/></div>
          <SideItem active={view==="all"} icon={<Folder/>} label="My Sites" count={profiles.length} onClick={()=>setView("all")}/>
          {folders.map((folder)=><SideItem key={folder} active={view===`folder:${folder}`} icon={<Folder/>} label={folder} count={profiles.filter(p=>(p.group??'')===folder).length} onClick={()=>setView(`folder:${folder}`)}/>) }
        </aside>
        <section className="ghost-site-manager-table min-w-0 overflow-auto p-3">
          <div className="grid grid-cols-[minmax(180px,1.4fr)_minmax(150px,1fr)_90px_120px_110px] border-b border-border px-3 py-2 text-[11px] font-semibold text-text-dim"><span>Name</span><span>Host</span><span>Protocol</span><span>Tags</span><span>Last Used</span></div>
          {filtered.length===0 && <div className="p-10 text-center text-text-muted">No saved sites match this view.</div>}
          {filtered.map(p=><SiteRow key={p.id} profile={p} active={selected?.id===p.id} favorite={p.favorite===true} connected={sessions.some(s=>s.profileId===p.id)} onClick={()=>select(p.id)} onFavorite={()=>void saveProfile({...p,favorite:!p.favorite})}/>) }
        </section>
        <aside className="ghost-site-manager-details border-l border-border bg-[#071f35] p-4 overflow-y-auto">
          {selected && draft ? <>
            <div className="mb-4 flex items-center gap-3"><div className="flex h-12 w-12 items-center justify-center rounded-lg bg-accent-strong text-white"><Server size={24}/></div><div className="min-w-0"><div className="truncate text-[17px] font-semibold">{selected.name}</div><div className="text-[11px] text-text-muted">{selected.group || 'Saved connection'}</div></div><div className="flex-1"/><button className="ghost-mini-button" onClick={editing?save:startEdit}>{editing?<Save size={13}/>:<Edit3 size={13}/>} {editing?'Save':'Edit'}</button></div>
            <div className="space-y-3 border-t border-border pt-4">
              <EditField label="Protocol" editing={editing}><select value={draft.protocol} onChange={e=>{const protocol=e.target.value as Protocol;setDraft({...draft,protocol,port:PROTOCOL_DEFAULT_PORT[protocol]});}}><option value="sftp">SFTP</option><option value="ftp">FTP</option><option value="ftps">FTPS</option></select></EditField>
              <EditField label="Host / Address" editing={editing}><input value={draft.host} onChange={e=>setDraft({...draft,host:e.target.value})}/></EditField>
              <EditField label="Port" editing={editing}><input type="number" value={draft.port} onChange={e=>setDraft({...draft,port:Number(e.target.value)||22})}/></EditField>
              <EditField label="Username" editing={editing}><input value={draft.username} onChange={e=>setDraft({...draft,username:e.target.value})}/></EditField>
              <EditField label="Remote Path" editing={editing}><input value={draft.defaultRemotePath||""} onChange={e=>setDraft({...draft,defaultRemotePath:e.target.value})}/></EditField>
              <EditField label="Folder" editing={editing}><input value={draft.group||""} onChange={e=>setDraft({...draft,group:e.target.value.trimStart()||undefined})} placeholder="My Sites"/></EditField>
              <EditField label="Tags" editing={editing}><input value={(draft.tags??[]).join(", ")} onChange={e=>setDraft({...draft,tags:e.target.value.split(",").map(v=>v.trim()).filter(Boolean)})} placeholder="Production, Client"/></EditField>
              <label className="flex items-center justify-between gap-3 text-[11px] text-text-muted"><span>Bookmark</span><input type="checkbox" disabled={!editing} checked={draft.bookmarked===true} onChange={e=>setDraft({...draft,bookmarked:e.target.checked})}/></label>
              <EditField label="Encoding" editing={false}><input value="UTF-8" readOnly/></EditField>
              <EditField label="Connection Mode" editing={false}><input value={draft.protocol==='ftp'||draft.protocol==='ftps'?'Passive':'Secure'} readOnly/></EditField>
            </div>
            <div className="mt-5 flex gap-2"><button disabled={isConnected} className="ghost-primary-button flex-1" onClick={()=>void connect(selected.id)}><Link2 size={15}/>{isConnected?'Connected':'Connect'}</button><button className="ghost-mini-button flex-1" onClick={()=>void testSelected()}><RadioTower size={14}/> Test Connection</button></div>
            <button className="mt-3 flex w-full items-center justify-center gap-2 rounded-md border border-danger/30 bg-danger/10 py-2 text-[11px] text-danger hover:bg-danger/20" onClick={()=>void remove()}><Trash2 size={13}/> Delete Site</button>
          </> : <div className="grid h-full place-items-center text-text-muted"><div className="text-center"><Server size={58} className="mx-auto text-accent"/><div className="mt-2">No site selected</div></div></div>}
        </aside>
      </div>
      <div className="flex h-10 min-h-10 items-center border-t border-border bg-[#051929] px-4 text-[11px] text-text-muted"><span className="h-2 w-2 rounded-full bg-success mr-2"/>Ready<div className="flex-1"/><Server size={13} className="mr-1"/>{profiles.length} sites</div>
    </div>
  </div>
}

function SideItem({icon,label,count,active=false,onClick}:{icon:React.ReactNode;label:string;count:number;active?:boolean;onClick:()=>void}){return <button onClick={onClick} className={`mb-1 flex w-full items-center gap-2 rounded-md border px-2.5 py-2 text-left text-[12px] ${active?'border-accent/35 bg-accent/15 text-white':'border-transparent text-text-muted hover:bg-bg-hover hover:text-white'}`}><span className="[&>svg]:h-4 [&>svg]:w-4 text-accent">{icon}</span><span className="flex-1">{label}</span><span className="rounded bg-[#0a2d4b] px-1.5 py-.5 text-[10px]">{count}</span></button>}
function SiteRow({profile,active,favorite,connected,onClick,onFavorite}:{profile:ConnectionProfile;active:boolean;favorite:boolean;connected:boolean;onClick:()=>void;onFavorite:()=>void}){const tag=profile.tags?.[0]||profile.group||'Sites';return <button onClick={onClick} className={`grid w-full grid-cols-[minmax(180px,1.4fr)_minmax(150px,1fr)_90px_120px_110px] items-center border-b border-border-subtle px-3 py-2.5 text-left text-[12px] ${active?'bg-accent/15 outline outline-1 outline-accent/60':'hover:bg-bg-hover/70'}`}><span className="flex min-w-0 items-center gap-2"><span onClick={e=>{e.stopPropagation();onFavorite();}} onKeyDown={e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();e.stopPropagation();onFavorite();}}} role="button" tabIndex={0} aria-label={favorite?'Remove from favorites':'Add to favorites'}><Star size={14} className={`shrink-0 ${favorite?'fill-warning text-warning':'text-text-dim'}`}/></span><Server size={14} className="shrink-0 text-accent"/><span className="truncate font-medium">{profile.name}</span></span><span className="truncate text-text-muted">{profile.host}</span><span className="text-text-muted">{profile.protocol.toUpperCase()}</span><span><em className="not-italic rounded-full border border-accent/30 bg-accent/10 px-2 py-1 text-[10px] text-accent">{tag}</em></span><span className={connected?'text-success':'text-text-dim'}>{connected?'Connected':formatLastUsed(profile.lastUsed)}</span></button>}
function formatLastUsed(value?:number){if(!value)return '—';const d=new Date(value*1000);const now=new Date();return d.toDateString()===now.toDateString()?d.toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'}):d.toLocaleDateString();}
function EditField({label,editing,children}:{label:string;editing:boolean;children:React.ReactNode}){return <label className="block"><div className="mb-1 text-[10px] text-text-dim">{label}</div><div className={`ghost-ref-field ${editing?'':'opacity-90'}`}>{children}</div></label>}
