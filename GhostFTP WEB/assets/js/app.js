import { api, uploadRequest, submitDownload } from './api.js';
import { $, $$, escapeHtml, formatBytes, formatDate, joinPath, parentPath, fileIcon, naturalCompare } from './utils.js';

const endpoints = {
    download: document.querySelector('meta[name="download-url"]')?.content || 'download/file',
    archive: document.querySelector('meta[name="download-archive-url"]')?.content || 'download/archive',
    preview: document.querySelector('meta[name="preview-url"]')?.content || 'preview/file',
};
const uploadMax = Number(document.querySelector('meta[name="upload-max-bytes"]')?.content || Number.MAX_SAFE_INTEGER);

const state = {
    profiles: [], profileId: '', path: '/', items: [], favorites: [], selected: new Set(),
    sort: { key: 'name', direction: 1 }, showHidden: true, uploadConflict: 'rename',
    editorPath: '', editorEtag: '', activeUpload: null, contextItem: null,
};

const el = {
    welcome: $('#welcome'), fileApp: $('#fileApp'), profiles: $('#profiles'), favorites: $('#favorites'), rows: $('#fileRows'),
    empty: $('#emptyState'), loading: $('#loadingState'), breadcrumbs: $('#breadcrumbs'), filter: $('#filterInput'),
    selectAll: $('#selectAll'), selectionBar: $('#selectionBar'), selectionCount: $('#selectionCount'), status: $('#statusText'),
    statusDot: $('#connectionStatus'), connectionName: $('#connectionName'), connectionMeta: $('#connectionMeta'), itemSummary: $('#itemSummary'),
    pathStatus: $('#pathStatus'), favoriteBtn: $('#favoriteBtn'), showHidden: $('#showHidden'), uploadConflict: $('#uploadConflict'),
    profileModal: $('#profileModal'), profileForm: $('#profileForm'), profileTitle: $('#profileTitle'), editorModal: $('#editorModal'),
    editorContent: $('#editorContent'), editorTitle: $('#editorTitle'), editorInfo: $('#editorInfo'), promptModal: $('#promptModal'),
    promptForm: $('#promptForm'), promptInput: $('#promptInput'), promptTitle: $('#promptTitle'), promptDescription: $('#promptDescription'),
    searchModal: $('#searchModal'), searchResults: $('#searchResults'), searchStatus: $('#searchStatus'), contextMenu: $('#contextMenu'),
    previewModal: $('#previewModal'), previewImage: $('#previewImage'), transferPanel: $('#transferPanel'), transferProgress: $('#transferProgress'),
    transferTitle: $('#transferTitle'), transferDetail: $('#transferDetail'), cancelUpload: $('#cancelUpload'), sidebar: $('#sidebar'),
    sidebarBackdrop: $('#sidebarBackdrop'), userMenu: $('#userMenu'), userMenuBtn: $('#userMenuBtn'),
};

function toast(message, error = false) {
    const node = document.createElement('div');
    node.className = `toast${error ? ' error' : ''}`;
    node.textContent = message;
    $('#toastHost').append(node);
    setTimeout(() => node.remove(), error ? 6500 : 3600);
}
function setStatus(text, online = null) {
    el.status.textContent = text;
    if (online !== null) el.statusDot.classList.toggle('online', online);
}
function currentProfile() { return state.profiles.find((profile) => profile.id === state.profileId) || null; }
function closeSidebar() { el.sidebar.classList.remove('open'); el.sidebarBackdrop.classList.add('hidden'); $('#sidebarToggle')?.setAttribute('aria-expanded', 'false'); }
function openModal(node) { node.classList.remove('hidden'); setTimeout(() => node.querySelector('input,select,textarea,button')?.focus(), 20); }
function closeModal(node) { node.classList.add('hidden'); }
function isImage(name) { return /\.(?:jpe?g|png|gif|webp)$/i.test(name); }
function isEditable(name) { return /\.(?:txt|md|html?|css|js|mjs|ts|tsx|jsx|json|xml|ya?ml|ini|conf|env|php|py|go|java|kt|swift|sh|sql|htaccess)$/i.test(name) || !name.includes('.'); }

async function loadProfiles() {
    const result = await api('profiles');
    state.profiles = result.profiles || [];
    renderProfiles();
    if (!state.profileId && state.profiles.length) await selectProfile(state.profiles[0].id);
    if (!state.profiles.length) {
        el.welcome.classList.remove('hidden');
        el.fileApp.classList.add('hidden');
    }
}
function renderProfiles() {
    el.profiles.innerHTML = '';
    if (!state.profiles.length) {
        el.profiles.innerHTML = '<span class="muted tiny">Još nema spremljenih veza.</span>';
        return;
    }
    for (const profile of state.profiles) {
        const row = document.createElement('div');
        row.className = `profile-item${profile.id === state.profileId ? ' active' : ''}`;
        row.innerHTML = `<span class="profile-dot"></span><button type="button" class="profile-main"><span class="profile-copy"><strong>${escapeHtml(profile.label)}</strong><small>${escapeHtml(profile.protocol.toUpperCase())} · ${escapeHtml(profile.host)}</small></span></button><button class="profile-edit" type="button" aria-label="Uredi vezu">⋯</button>`;
        row.querySelector('.profile-main').addEventListener('click', () => selectProfile(profile.id));
        row.querySelector('.profile-edit').addEventListener('click', (event) => { event.stopPropagation(); showProfileModal(profile); });
        el.profiles.append(row);
    }
}
async function selectProfile(id) {
    if (!id) return;
    state.profileId = id;
    state.path = '/';
    state.selected.clear();
    renderProfiles();
    closeSidebar();
    const profile = currentProfile();
    el.connectionName.textContent = profile?.label || 'Server';
    el.connectionMeta.textContent = profile ? `${profile.protocol.toUpperCase()} · ${profile.host}:${profile.port}` : '';
    el.welcome.classList.add('hidden');
    el.fileApp.classList.remove('hidden');
    await openPath('/');
}
async function openPath(path) {
    if (!state.profileId) return;
    el.loading.classList.remove('hidden');
    el.empty.classList.add('hidden');
    setStatus('Povezivanje…');
    try {
        const result = await api('list', { profile_id: state.profileId, path });
        state.path = result.path || path;
        state.items = result.items || [];
        state.favorites = result.favorites || [];
        state.selected.clear();
        renderAll();
        setStatus('Povezano', true);
    } catch (error) {
        setStatus('Greška veze', false);
        toast(error.message, true);
    } finally {
        el.loading.classList.add('hidden');
    }
}
function visibleItems() {
    const query = (el.filter.value || '').trim().toLocaleLowerCase('hr');
    return state.items.filter((item) => {
        if (!state.showHidden && String(item.name).startsWith('.')) return false;
        if (query && !String(item.name).toLocaleLowerCase('hr').includes(query)) return false;
        return true;
    }).sort((a, b) => {
        if (state.sort.key === 'name' && a.type !== b.type) return a.type === 'dir' ? -1 : 1;
        let result = 0;
        if (state.sort.key === 'name') result = naturalCompare(a.name, b.name);
        else if (state.sort.key === 'size') result = Number(a.size || 0) - Number(b.size || 0);
        else result = String(a.modified || '').localeCompare(String(b.modified || ''));
        return result * state.sort.direction;
    });
}
function renderAll() { renderBreadcrumbs(); renderFiles(); renderFavorites(); updateSelection(); }
function renderBreadcrumbs() {
    const parts = state.path.split('/').filter(Boolean);
    const rows = [{ label: '/', path: '/' }];
    let built = '';
    for (const part of parts) { built += `/${part}`; rows.push({ label: part, path: built }); }
    el.breadcrumbs.innerHTML = '';
    rows.forEach((row, index) => {
        const button = document.createElement('button');
        button.type = 'button'; button.textContent = row.label; button.title = row.path;
        button.addEventListener('click', () => openPath(row.path));
        el.breadcrumbs.append(button);
        if (index < rows.length - 1) el.breadcrumbs.append(document.createTextNode('›'));
    });
    el.pathStatus.textContent = state.path;
    el.favoriteBtn.textContent = state.favorites.includes(state.path) ? '★' : '☆';
}
function renderFiles() {
    const items = visibleItems();
    el.rows.innerHTML = '';
    for (const item of items) {
        const path = joinPath(state.path, item.name);
        const tr = document.createElement('tr');
        tr.dataset.path = path; tr.dataset.type = item.type;
        if (state.selected.has(path)) tr.classList.add('selected');
        tr.innerHTML = `<td><input class="row-check" type="checkbox" ${state.selected.has(path) ? 'checked' : ''} aria-label="Označi ${escapeHtml(item.name)}"></td><td><div class="file-name"><span class="file-icon">${fileIcon(item)}</span><strong title="${escapeHtml(item.name)}">${escapeHtml(item.name)}</strong></div></td><td class="desktop-only">${item.type === 'dir' ? 'Mapa' : 'Datoteka'}</td><td>${item.type === 'dir' ? '—' : formatBytes(item.size)}</td><td class="desktop-only">${escapeHtml(formatDate(item.modified))}</td><td><div class="row-actions"><button class="row-open" type="button" title="Otvori">↗</button><button class="row-more" type="button" title="Više">⋯</button></div></td>`;
        tr.querySelector('.row-check').addEventListener('change', (event) => toggleSelection(path, event.target.checked));
        tr.querySelector('.row-open').addEventListener('click', () => openItem(item));
        tr.querySelector('.row-more').addEventListener('click', (event) => showContext(event.currentTarget, item));
        tr.addEventListener('dblclick', () => openItem(item));
        tr.addEventListener('contextmenu', (event) => { event.preventDefault(); showContextAt(event.clientX, event.clientY, item); });
        el.rows.append(tr);
    }
    el.empty.classList.toggle('hidden', items.length !== 0);
    el.itemSummary.textContent = `${items.length} prikazano · ${state.items.length} ukupno`;
    el.selectAll.checked = items.length > 0 && items.every((item) => state.selected.has(joinPath(state.path, item.name)));
}
function renderFavorites() {
    el.favorites.innerHTML = '';
    if (!state.favorites.length) { el.favorites.innerHTML = '<span class="muted tiny">Nema favorita za ovu vezu.</span>'; return; }
    for (const path of state.favorites) {
        const button = document.createElement('button'); button.type = 'button'; button.className = 'favorite-item';
        button.innerHTML = `<span>☆</span><span class="profile-copy"><strong>${escapeHtml(path === '/' ? '/' : path.split('/').filter(Boolean).pop())}</strong><small>${escapeHtml(path)}</small></span>`;
        button.addEventListener('click', () => openPath(path)); el.favorites.append(button);
    }
}
function toggleSelection(path, checked) { checked ? state.selected.add(path) : state.selected.delete(path); renderFiles(); updateSelection(); }
function updateSelection() {
    const count = state.selected.size;
    el.selectionBar.classList.toggle('hidden', count === 0);
    el.selectionCount.textContent = `${count} označeno`;
}
function selectedItems() { return state.items.filter((item) => state.selected.has(joinPath(state.path, item.name))); }
async function openItem(item) {
    if (item.type === 'dir') return openPath(joinPath(state.path, item.name));
    if (isEditable(item.name) && Number(item.size || 0) <= 4194304) return openEditor(joinPath(state.path, item.name));
    downloadOne(joinPath(state.path, item.name));
}
function downloadOne(path) { submitDownload(endpoints.download, { profile_id: state.profileId, path }); }
async function openEditor(path) {
    try {
        const result = await api('read', { profile_id: state.profileId, path });
        state.editorPath = path; state.editorEtag = result.etag || '';
        el.editorTitle.textContent = path.split('/').pop() || path;
        el.editorInfo.textContent = `${formatBytes(result.bytes)} · ${path}`;
        el.editorContent.value = result.content || '';
        openModal(el.editorModal); updateEditorCursor();
    } catch (error) { toast(error.message, true); }
}
async function saveEditor() {
    try {
        const result = await api('write', { profile_id: state.profileId, path: state.editorPath, content: el.editorContent.value, if_match: state.editorEtag });
        state.editorEtag = result.etag || ''; toast('Datoteka je spremljena.'); await openPath(state.path);
    } catch (error) { toast(error.message, true); }
}
function updateEditorCursor() {
    const pos = el.editorContent.selectionStart || 0;
    const before = el.editorContent.value.slice(0, pos); const lines = before.split('\n');
    $('#editorCursor').textContent = `Red ${lines.length}, stupac ${lines.at(-1).length + 1}`;
}
function showContext(anchor, item) { const rect = anchor.getBoundingClientRect(); showContextAt(Math.min(rect.left, innerWidth - 190), rect.bottom + 4, item); }
function showContextAt(x, y, item) {
    state.contextItem = item;
    el.contextMenu.style.left = `${Math.max(6, Math.min(x, innerWidth - 190))}px`;
    el.contextMenu.style.top = `${Math.max(6, Math.min(y, innerHeight - 380))}px`;
    el.contextMenu.classList.remove('hidden');
    $('[data-action="preview"]', el.contextMenu).classList.toggle('hidden', item.type === 'dir' || !isImage(item.name));
    $('[data-action="edit"]', el.contextMenu).classList.toggle('hidden', item.type === 'dir' || !isEditable(item.name));
    $('[data-action="extract"]', el.contextMenu).classList.toggle('hidden', item.type === 'dir' || !/\.zip$/i.test(item.name));
}
async function contextAction(action) {
    const item = state.contextItem; el.contextMenu.classList.add('hidden'); if (!item) return;
    const path = joinPath(state.path, item.name);
    if (action === 'open') return openItem(item);
    if (action === 'download') return item.type === 'dir' ? submitDownload(endpoints.archive,{profile_id:state.profileId,paths:JSON.stringify([path]),name:`${item.name}.zip`}) : downloadOne(path);
    if (action === 'edit') return openEditor(path);
    if (action === 'preview') return previewImage(path, item.name);
    if (action === 'duplicate') return mutate('duplicate', { source:path }, 'Stavka je duplicirana.');
    if (action === 'delete') return deleteItems([item]);
    if (action === 'rename') {
        const name = await promptValue('Preimenuj', 'Novi naziv', item.name); if (!name) return;
        return mutate('rename', { from:path, name }, 'Stavka je preimenovana.');
    }
    if (action === 'chmod') {
        const mode = await promptValue('Dozvole', 'CHMOD (npr. 644 ili 755)', item.permissions || (item.type === 'dir' ? '755' : '644')); if (!mode) return;
        return mutate('chmod', { path, mode }, 'Dozvole su promijenjene.');
    }
    if (action === 'copy' || action === 'move') {
        const destination = await promptValue(action === 'copy' ? 'Kopiraj u…' : 'Premjesti u…', 'Odredišna putanja', joinPath(state.path,item.name)); if (!destination) return;
        return mutate(action, { source:path, destination, conflict:'rename' }, action === 'copy' ? 'Kopiranje je završeno.' : 'Premještanje je završeno.');
    }
    if (action === 'extract') return mutate('extract', { path, destination:state.path }, 'ZIP je raspakiran.');
}
async function mutate(action, data, message) {
    try { await api(action, { profile_id:state.profileId, ...data }); toast(message); await openPath(state.path); }
    catch (error) { toast(error.message, true); }
}
async function deleteItems(items) {
    if (!items.length || !confirm(`Obrisati ${items.length} odabranih stavki? Radnja može biti nepovratna.`)) return;
    try {
        if (items.length === 1) {
            const item = items[0]; await api('delete',{profile_id:state.profileId,path:joinPath(state.path,item.name),type:item.type,recursive:true});
        } else {
            const payload = items.map((item)=>({path:joinPath(state.path,item.name),type:item.type}));
            await api('bulk_delete',{profile_id:state.profileId,items:JSON.stringify(payload)});
        }
        state.selected.clear(); toast('Brisanje je završeno.'); await openPath(state.path);
    } catch (error) { toast(error.message,true); }
}
async function previewImage(path, name) {
    const body = new FormData();
    body.append('csrf', document.querySelector('meta[name="csrf-token"]').content); body.append('profile_id',state.profileId); body.append('path',path);
    try {
        const response = await fetch(endpoints.preview,{method:'POST',body,credentials:'same-origin',cache:'no-store'});
        if (!response.ok) throw new Error(await response.text());
        const blob = await response.blob(); const old = el.previewImage.dataset.url; if (old) URL.revokeObjectURL(old);
        const url = URL.createObjectURL(blob); el.previewImage.dataset.url = url; el.previewImage.src = url; $('#previewTitle').textContent = name; openModal(el.previewModal);
    } catch (error) { toast(error.message || 'Pregled nije uspio.',true); }
}
function promptValue(title, label, value = '', description = '') {
    return new Promise((resolve) => {
        el.promptTitle.textContent = title; $('#promptInputLabel').childNodes[0].textContent = label; el.promptInput.value = value;
        el.promptDescription.textContent = description; el.promptDescription.classList.toggle('hidden', !description); openModal(el.promptModal); el.promptInput.select();
        const submit = (event) => { event.preventDefault(); cleanup(); closeModal(el.promptModal); resolve(el.promptInput.value); };
        const cancel = () => { cleanup(); closeModal(el.promptModal); resolve(null); };
        const cleanup = () => { el.promptForm.removeEventListener('submit',submit); $$('.close-prompt').forEach((button)=>button.removeEventListener('click',cancel)); };
        el.promptForm.addEventListener('submit',submit); $$('.close-prompt').forEach((button)=>button.addEventListener('click',cancel));
    });
}
async function newFolder() { const name = await promptValue('Nova mapa','Naziv mape'); if (name) mutate('mkdir',{path:state.path,name},'Mapa je izrađena.'); }
async function newFile() { const name = await promptValue('Nova datoteka','Naziv datoteke'); if (name) { await mutate('new_file',{path:state.path,name,content:''},'Datoteka je izrađena.'); openEditor(joinPath(state.path,name)); } }
async function toggleFavorite() {
    try { const result = await api('toggle_favorite',{profile_id:state.profileId,path:state.path}); state.favorites=result.favorites||[]; renderFavorites(); renderBreadcrumbs(); }
    catch(error){toast(error.message,true);}
}
async function searchRemote() {
    const query = ($('#remoteSearch').value || '').trim(); if (!query || !state.profileId) return;
    openModal(el.searchModal); el.searchStatus.textContent='Pretraživanje…'; el.searchResults.innerHTML='';
    try {
        const result=await api('search',{profile_id:state.profileId,path:state.path,query}); const rows=result.results||[]; el.searchStatus.textContent=`${rows.length} rezultata`;
        for(const item of rows){const button=document.createElement('button');button.type='button';button.className='search-result';button.innerHTML=`<span>${fileIcon(item)} ${escapeHtml(item.name)}</span><small>${escapeHtml(item.path)}</small>`;button.addEventListener('click',()=>{closeModal(el.searchModal);item.type==='dir'?openPath(item.path):openPath(parentPath(item.path));});el.searchResults.append(button);}
    } catch(error){el.searchStatus.textContent=error.message;}
}
function showProfileModal(profile=null) {
    el.profileForm.reset(); el.profileForm.elements.passive.checked=true; el.profileForm.elements.utf8.checked=true; el.profileForm.elements.timeout.value='30';
    if(profile){ for(const [key,value] of Object.entries(profile)){const field=el.profileForm.elements[key];if(!field)continue;if(field.type==='checkbox')field.checked=Boolean(value);else if(!['password','public_key','private_key','key_passphrase'].includes(key))field.value=value??'';} }
    else { el.profileForm.elements.protocol.value='sftp'; el.profileForm.elements.port.value='22'; el.profileForm.elements.base_path.value='/'; }
    el.profileTitle.textContent=profile?'Uredi server':'Novi server'; $('#deleteProfileBtn').classList.toggle('hidden',!profile); $('#duplicateProfileBtn').classList.toggle('hidden',!profile); updateProfileFields(); openModal(el.profileModal);
}
function updateProfileFields(){const protocol=el.profileForm.elements.protocol.value;const key=protocol==='sftp'&&el.profileForm.elements.auth_method.value==='key';$$('.sftp-only',el.profileForm).forEach(n=>n.classList.toggle('hidden',protocol!=='sftp'));$('.key-auth',el.profileForm).classList.toggle('hidden',!key);if(!el.profileForm.elements.port.value)el.profileForm.elements.port.value=protocol==='sftp'?'22':'21';}
function profilePayload(){const fd=new FormData(el.profileForm);const out={};for(const [key,value] of fd.entries())out[key]=value;out.passive=el.profileForm.elements.passive.checked?'1':'0';out.utf8=el.profileForm.elements.utf8.checked?'1':'0';return out;}
async function saveProfile(event){event.preventDefault();try{const result=await api('save_profile',profilePayload());closeModal(el.profileModal);toast('Veza je spremljena.');await loadProfiles();if(result.profile?.id)await selectProfile(result.profile.id);}catch(error){toast(error.message,true);}}
async function testProfile(){try{const result=await api('test_profile_draft',profilePayload());toast(`${result.message} ${result.elapsed_ms||''} ms`);}catch(error){toast(error.message,true);}}
async function deleteProfile(){const id=el.profileForm.elements.id.value;if(!id||!confirm('Obrisati ovu spremljenu vezu?'))return;try{await api('delete_profile',{id});if(state.profileId===id){state.profileId='';state.items=[];}closeModal(el.profileModal);await loadProfiles();toast('Veza je obrisana.');}catch(error){toast(error.message,true);}}
async function duplicateProfile(){const id=el.profileForm.elements.id.value;if(!id)return;try{await api('duplicate_profile',{id});closeModal(el.profileModal);await loadProfiles();toast('Veza je duplicirana.');}catch(error){toast(error.message,true);}}
async function uploadFiles(files, folder=false){if(!state.profileId||!files.length)return;const list=[...files];for(const file of list){if(Number.isFinite(uploadMax)&&file.size>uploadMax){toast(`${file.name}: datoteka prelazi PHP upload limit.`,true);return;}}el.transferPanel.classList.remove('hidden');el.cancelUpload.classList.remove('hidden');for(let i=0;i<list.length;i++){const file=list[i];el.transferTitle.textContent=`Upload ${i+1}/${list.length}`;el.transferDetail.textContent=file.name;el.transferProgress.value=0;const relative=folder?(file.webkitRelativePath||file.name):'';const task=uploadRequest({profileId:state.profileId,path:state.path,file,relativePath:relative,conflict:state.uploadConflict,onProgress:(percent,loaded,total)=>{el.transferProgress.value=percent;el.transferDetail.textContent=`${file.name} · ${percent}% · ${formatBytes(loaded)} / ${formatBytes(total)}`;}});state.activeUpload=task.xhr;try{await task.promise;}catch(error){if(error.name!=='AbortError')toast(error.message,true);break;}finally{state.activeUpload=null;}}el.transferPanel.classList.add('hidden');el.cancelUpload.classList.add('hidden');await openPath(state.path);}
function bulkDestination(action){return promptValue(action==='copy'?'Kopiraj označeno':'Premjesti označeno','Odredišna mapa',state.path,'Svaka stavka zadržava svoj naziv. Konflikti se automatski preimenuju.');}
async function bulkCopyMove(action){const items=selectedItems();if(!items.length)return;const root=await bulkDestination(action);if(!root)return;try{for(const item of items){const source=joinPath(state.path,item.name);const destination=joinPath(root,item.name);await api(action,{profile_id:state.profileId,source,destination,conflict:'rename'});}state.selected.clear();toast(action==='copy'?'Kopiranje je završeno.':'Premještanje je završeno.');await openPath(state.path);}catch(error){toast(error.message,true);}}
async function bulkZip(){const paths=[...state.selected];if(!paths.length)return;const name=await promptValue('Stvori ZIP','Naziv arhive','archive.zip');if(!name)return;await mutate('zip',{paths:JSON.stringify(paths),destination:joinPath(state.path,name.endsWith('.zip')?name:`${name}.zip`)},'ZIP je izrađen.');}
function bulkDownload(){const paths=[...state.selected];if(paths.length)submitDownload(endpoints.archive,{profile_id:state.profileId,paths:JSON.stringify(paths),name:'GhostFTP-download.zip'});}

$('#sidebarToggle').addEventListener('click',()=>{const open=el.sidebar.classList.toggle('open');el.sidebarBackdrop.classList.toggle('hidden',!open);$('#sidebarToggle').setAttribute('aria-expanded',String(open));});
el.sidebarBackdrop.addEventListener('click',closeSidebar);
el.userMenuBtn.addEventListener('click',()=>{const open=el.userMenu.classList.toggle('hidden');el.userMenuBtn.setAttribute('aria-expanded',String(!open));});
document.addEventListener('click',(event)=>{if(!event.target.closest('.user-menu-wrap'))el.userMenu.classList.add('hidden');if(!event.target.closest('#contextMenu')&&!event.target.closest('.row-more'))el.contextMenu.classList.add('hidden');});
$('#addProfile').addEventListener('click',()=>showProfileModal());$('#welcomeAdd').addEventListener('click',()=>showProfileModal());
$$('.close-profile').forEach(b=>b.addEventListener('click',()=>closeModal(el.profileModal)));el.profileForm.addEventListener('submit',saveProfile);el.profileForm.elements.protocol.addEventListener('change',()=>{el.profileForm.elements.port.value=el.profileForm.elements.protocol.value==='sftp'?'22':'21';updateProfileFields();});el.profileForm.elements.auth_method.addEventListener('change',updateProfileFields);$('#testProfileBtn').addEventListener('click',testProfile);$('#deleteProfileBtn').addEventListener('click',deleteProfile);$('#duplicateProfileBtn').addEventListener('click',duplicateProfile);
$('#upBtn').addEventListener('click',()=>openPath(parentPath(state.path)));$('#refreshBtn').addEventListener('click',()=>openPath(state.path));el.favoriteBtn.addEventListener('click',toggleFavorite);el.filter.addEventListener('input',renderFiles);el.showHidden.addEventListener('change',()=>{state.showHidden=el.showHidden.checked;renderFiles();});el.uploadConflict.addEventListener('change',()=>state.uploadConflict=el.uploadConflict.value);el.selectAll.addEventListener('change',()=>{for(const item of visibleItems()){const path=joinPath(state.path,item.name);el.selectAll.checked?state.selected.add(path):state.selected.delete(path);}renderFiles();updateSelection();});$$('.sort-button').forEach(button=>button.addEventListener('click',()=>{const key=button.dataset.sort;if(state.sort.key===key)state.sort.direction*=-1;else state.sort={key,direction:1};renderFiles();}));
$('#newFolderBtn').addEventListener('click',newFolder);$('#newFileBtn').addEventListener('click',newFile);$('#uploadInput').addEventListener('change',(event)=>{uploadFiles(event.target.files,false);event.target.value='';});$('#folderUploadInput').addEventListener('change',(event)=>{uploadFiles(event.target.files,true);event.target.value='';});el.cancelUpload.addEventListener('click',()=>state.activeUpload?.abort());
$('#clearSelection').addEventListener('click',()=>{state.selected.clear();renderFiles();updateSelection();});$('#bulkDelete').addEventListener('click',()=>deleteItems(selectedItems()));$('#bulkCopy').addEventListener('click',()=>bulkCopyMove('copy'));$('#bulkMove').addEventListener('click',()=>bulkCopyMove('move'));$('#bulkZip').addEventListener('click',bulkZip);$('#bulkDownload').addEventListener('click',bulkDownload);
$('#searchBtn').addEventListener('click',searchRemote);$('#remoteSearch').addEventListener('keydown',(event)=>{if(event.key==='Enter'){event.preventDefault();searchRemote();}});$$('.close-search').forEach(b=>b.addEventListener('click',()=>closeModal(el.searchModal)));
$$('.close-editor').forEach(b=>b.addEventListener('click',()=>closeModal(el.editorModal)));$('#saveEditor').addEventListener('click',saveEditor);el.editorContent.addEventListener('keyup',updateEditorCursor);el.editorContent.addEventListener('click',updateEditorCursor);el.editorContent.addEventListener('keydown',(event)=>{if((event.ctrlKey||event.metaKey)&&event.key.toLowerCase()==='s'){event.preventDefault();saveEditor();}if(event.key==='Tab'){event.preventDefault();const start=el.editorContent.selectionStart,end=el.editorContent.selectionEnd;el.editorContent.setRangeText('    ',start,end,'end');}});
$$('.close-preview').forEach(b=>b.addEventListener('click',()=>{const url=el.previewImage.dataset.url;if(url)URL.revokeObjectURL(url);el.previewImage.removeAttribute('src');delete el.previewImage.dataset.url;closeModal(el.previewModal);}));
el.contextMenu.addEventListener('click',(event)=>{const button=event.target.closest('[data-action]');if(button)contextAction(button.dataset.action);});
const dropZone=$('#dropZone');['dragenter','dragover'].forEach(type=>dropZone.addEventListener(type,(event)=>{event.preventDefault();dropZone.classList.add('dragging');}));['dragleave','drop'].forEach(type=>dropZone.addEventListener(type,(event)=>{event.preventDefault();dropZone.classList.remove('dragging');}));dropZone.addEventListener('drop',(event)=>{if(event.dataTransfer?.files?.length)uploadFiles(event.dataTransfer.files,false);});
document.addEventListener('keydown',(event)=>{if(event.key==='Escape'){$$('.modal:not(.hidden)').forEach(closeModal);el.contextMenu.classList.add('hidden');closeSidebar();}if(event.key==='Delete'&&state.selected.size&&!event.target.matches('input,textarea'))deleteItems(selectedItems());if(event.key==='F2'&&state.selected.size===1&&!event.target.matches('input,textarea')){event.preventDefault();const item=selectedItems()[0];if(item){state.contextItem=item;contextAction('rename');}}});

loadProfiles().catch((error)=>toast(error.message,true));
