(() => { "use strict";
const $=id=>document.getElementById(id);
let current="files",model=null,activeSite="studio",searchTerm="";
const titles={files:["Files","Explore fictional remote content and simulate transfers without connecting to any real server."],sites:["Sites","Four connected-looking demo profiles with no real credentials."],transfers:["Transfers","Watch simulated upload and download progress against in-memory fixture files."],sync:["Sync & Backup","A read-only additive plan, not a real change to any device."],settings:["Settings","An offline, credential-free preview of Ghost FTP configuration."]};
const text=(id,value)=>$(id).textContent=String(value);
const fmt=n=>n>1024*1024?(n/(1024*1024)).toFixed(1)+" MiB":Math.max(1,Math.round(n/1024))+" KiB";
function view(name){if(!Object.prototype.hasOwnProperty.call(titles,name))return;current=name;for(const el of document.querySelectorAll("[data-view]")){const chosen=el.dataset.view===name;chosen?el.setAttribute("aria-current","page"):el.removeAttribute("aria-current")}for(const key of Object.keys(titles))$("page-"+key).hidden=key!==name;text("crumb-current",titles[name][0]);text("page-heading",titles[name][0]);text("page-description",titles[name][1])}
for(const el of document.querySelectorAll("nav [data-view]"))el.addEventListener("click",()=>view(el.dataset.view));
function render(data){if(!data||data.demo!==true)throw Error("Invalid demo contract");
model=data;text("version","v"+data.version+" · DEMO");text("stat-sites",data.sites.length);text("stat-files",data.files.length);text("stat-transfers",data.transfers.length);
const files=$("file-list");files.replaceChildren();
const active=data.sites.find(s=>s.id===activeSite)||data.sites[0];
text("active-demo-site","Remote browser · "+active.name);
for(const f of data.files.filter(f=>(f.name+" "+f.folder+" "+f.type).toLowerCase().includes(searchTerm))){const tr=document.createElement("tr");
const n=document.createElement("td");n.className="name";n.textContent="▧ "+f.name;
const p=document.createElement("td");p.textContent=f.folder;
const sz=document.createElement("td");sz.textContent=fmt(f.bytes);
const actions=document.createElement("td");
for(const direction of ["download","upload"]){const b=document.createElement("button");b.type="button";b.className="action";b.textContent=direction==="upload"?"Simulate ↑ upload":"Simulate ↓ download";b.addEventListener("click",()=>simulate(f.id,direction));actions.append(b)}
tr.append(n,p,sz,actions);files.append(tr)}
const sites=$("site-list");sites.replaceChildren();
for(const s of data.sites){const item=document.createElement("article");item.className="site";if(s.id===activeSite)item.classList.add("chosen");
for(const [tag,value]of [["strong",s.name],["span",s.protocol+" · SIMULATED"],["small",s.detail]]){const e=document.createElement(tag);e.textContent=value;item.append(e)}
const select=document.createElement("button");select.type="button";select.className="action";select.textContent=s.id===activeSite?"Selected demo account":"Select demo account";select.disabled=s.id===activeSite;select.addEventListener("click",()=>{activeSite=s.id;view("files");render(model);text("status","Switched fictional profile; no network connection occurred.")});item.append(select);sites.append(item)}
const transfers=$("transfer-list");transfers.replaceChildren();
if(!data.transfers.length){const t=document.createElement("p");t.textContent="No simulated transfers yet. Use the Upload/Download controls in Files.";transfers.append(t)}
for(const t of [...data.transfers].reverse()){const box=document.createElement("div");box.className="transfer";
const row=document.createElement("div");row.className="transfer-head";const name=document.createElement("strong");name.textContent=t.file;const state=document.createElement("span");state.textContent=t.direction+" · "+t.state+" · "+t.progress+"%";row.append(name,state);
const meter=document.createElement("div");meter.className="progress";const fill=document.createElement("div");fill.style.width=Math.max(0,Math.min(100,t.progress))+"%";meter.append(fill);
box.append(row,meter);
if(t.state==="simulating"){const stop=document.createElement("button");stop.type="button";stop.className="action";stop.textContent="Cancel simulation";stop.addEventListener("click",()=>act("/api/v1/cancel",{id:t.id}));box.append(stop)}
if(t.state==="cancelled"){const matching=data.files.find(f=>f.name===t.file);if(matching){const retry=document.createElement("button");retry.type="button";retry.className="action";retry.textContent="Retry simulation";retry.addEventListener("click",()=>simulate(matching.id,t.direction));box.append(retry)}}
transfers.append(box)}
const sync=$("sync-list");sync.replaceChildren();for(const s of data.sync_preview){const e=document.createElement("div");e.className="sync-item";e.textContent="✓ "+s;sync.append(e)}
}
async function act(route,data){try{const r=await fetch(route,{method:"POST",cache:"no-store",credentials:"omit",headers:{"Content-Type":"application/json","X-Ghost-Demo":"simulation-only"},body:JSON.stringify(data)});if(!r.ok)throw Error("Demo action blocked: HTTP "+r.status);await poll();text("status","SIMULATED operation, no real FTP/SFTP/network or disk activity.")}catch(e){text("status",e.message)}}
function simulate(file_id,direction){view("transfers");void act("/api/v1/transfer",{file_id,direction})}
async function poll(){try{const r=await fetch("/api/v1/state",{cache:"no-store",credentials:"omit"});if(!r.ok)throw Error("Offline demo disconnected");render(await r.json());text("status","Offline local demo running • all accounts and transfers are fictional.")}catch(e){text("status",e.message)}}
$("demo-search").addEventListener("input",(e)=>{searchTerm=e.target.value.toLowerCase().trim();if(model)render(model)});
view("files");void poll();window.setInterval(()=>{if(!document.hidden)void poll()},300);
})();