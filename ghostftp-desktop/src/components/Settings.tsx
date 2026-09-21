import { useEffect, useRef, useState } from "react";
import {
  Bell,
  ChevronDown,
  Globe2,
  Keyboard,
  Languages,
  Monitor,
  Palette,
  Plug,
  RefreshCw,
  RotateCcw,
  Settings2,
  ShieldCheck,
  SlidersHorizontal,
  Sparkles,
  TerminalSquare,
  Wifi,
  ArrowDownUp,
} from "lucide-react";
import {
  captureSettingsSnapshot,
  resetSettingsToDefaults,
  restoreSettingsSnapshot,
  APP_THEMES,
  useSettings,
} from "@/stores/settingsStore";
import { getLocale, saveLocale } from "@/lib/i18n";
import { ipc } from "@/lib/ipc";
import { useUpdater } from "@/stores/updaterStore";
import { PRODUCT_VERSION_DISPLAY } from "@/lib/release";
import { GhostMark } from "./GhostBrand";
import { ReferenceWindowTitlebar } from "./ReferenceWindowChrome";
import { useDialog } from "@/hooks/useDialog";
import { requestDesktopNotificationPermission } from "@/lib/notifications";
import { SyncSettings } from "./SyncSettings";

interface Props { onClose: () => void; initialSection?: Section }
type Section = "general" | "appearance" | "transfers" | "connection" | "security" | "updates" | "integrations" | "shortcuts" | "language" | "sync";

export function Settings({ onClose, initialSection = "general" }: Props) {
  const panelRef = useRef<HTMLDivElement>(null);
  const [section, setSection] = useState<Section>(initialSection);
  const [pendingLocale, setPendingLocale] = useState(getLocale());
  const initialSettings = useRef(captureSettingsSnapshot());
  const initialLocale = useRef(getLocale());

  const cancel = () => {
    restoreSettingsSnapshot(initialSettings.current);
    saveLocale(initialLocale.current);
    onClose();
  };
  const apply = () => {
    saveLocale(pendingLocale);
    onClose();
  };
  const reset = () => {
    resetSettingsToDefaults();
    setPendingLocale("en");
  };

  useDialog(panelRef, { onClose: cancel });

  return (
    <div className="ghost-workspace-view ghost-standalone-view bg-[#041425]" role="region" aria-label="Ghost FTP Preferences">
      <div ref={panelRef} className="ghost-preferences flex h-full w-full flex-col overflow-hidden bg-bg-panel">
        <ReferenceWindowTitlebar suffix="Preferences" onClose={cancel} />
        <div className="ghost-preferences-body flex min-h-0 flex-1">
        <aside className="ghost-preferences-nav flex w-[214px] shrink-0 flex-col border-r border-border bg-[#061a2d] p-3">
          <Nav section="general" current={section} set={setSection} icon={<Settings2/>} label="General"/>
          <Nav section="appearance" current={section} set={setSection} icon={<Monitor/>} label="Appearance"/>
          <Nav section="transfers" current={section} set={setSection} icon={<ArrowDownUp/>} label="Transfers"/>
          <Nav section="connection" current={section} set={setSection} icon={<Wifi/>} label="Connection"/>
          <Nav section="security" current={section} set={setSection} icon={<ShieldCheck/>} label="Security"/>
          <Nav section="updates" current={section} set={setSection} icon={<RefreshCw/>} label="Updates"/>
          <Nav section="integrations" current={section} set={setSection} icon={<Plug/>} label="Integrations"/>
          <Nav section="shortcuts" current={section} set={setSection} icon={<Keyboard/>} label="Shortcuts"/>
          <Nav section="language" current={section} set={setSection} icon={<Globe2/>} label="Language"/>
          <div className="mt-auto border-t border-border pt-4 px-2 text-[11px] text-text-muted">
            <div className="mb-2 flex items-center gap-2"><GhostMark size={20}/><strong className="text-text">Ghost FTP</strong></div>
            <div>v{PRODUCT_VERSION_DISPLAY}</div><div>Built for Windows & Linux</div>
          </div>
        </aside>

        <main className="flex min-w-0 flex-1 flex-col">
          <div className="ghost-preferences-heading flex h-[78px] shrink-0 items-center gap-3 border-b border-border px-5">
            <Settings2 size={28} className="text-accent"/>
            <div>
              <div className="text-[22px] font-semibold">{section === "sync" ? "Sync & Backup" : section.charAt(0).toUpperCase() + section.slice(1)}</div>
              <div className="text-[12px] text-text-muted">{section === "sync" ? "Create and manage real folder synchronization pairs." : "Configure how Ghost FTP looks, behaves and keeps your data safe."}</div>
            </div>
          </div>
          <div className="ghost-preferences-content flex-1 overflow-y-auto p-4">
            {section === "general" && <GeneralGrid locale={pendingLocale} setLocale={setPendingLocale}/>}
            {section === "appearance" && <AppearancePanel/>}
            {section === "transfers" && <TransfersPanel/>}
            {section === "connection" && <ConnectionPanel/>}
            {section === "security" && <SecurityPanel/>}
            {section === "updates" && <UpdatesPanel/>}
            {section === "integrations" && <IntegrationsPanel/>}
            {section === "shortcuts" && <ShortcutsPanel/>}
            {section === "language" && <LanguagePanel locale={pendingLocale} setLocale={setPendingLocale}/>} 
            {section === "sync" && <div className="max-w-5xl"><SyncSettings/></div>}
          </div>
          <div className="ghost-preferences-actions flex h-[58px] shrink-0 items-center border-t border-border bg-[#061a2d] px-4">
            <button className="ghost-mini-button" onClick={reset}><RotateCcw size={14}/> Reset to Defaults</button>
            <div className="flex-1"/>
            <button className="ghost-mini-button" onClick={cancel}>Cancel</button>
            <button className="ghost-primary-button ml-2" onClick={apply}>Apply</button>
          </div>
        </main>
        </div>
      </div>
    </div>
  );
}

function GeneralGrid({ locale, setLocale }: { locale: string; setLocale: (value: any) => void }) {
  const s = useSettings();
  const themeOptions = APP_THEMES.map((theme) => [theme.value, theme.label] as [string, string]);

  return (
    <div className="ghost-general-grid grid grid-cols-2 gap-3">
      <GeneralCard icon={<Globe2 size={20}/>} title="Language" subtitle="Choose your preferred application language.">
        <SelectRow
          label="Primary Language"
          value={locale}
          onChange={setLocale}
          options={[
            ["en","English (English)"],
            ["hr","Hrvatski (Croatian)"],
            ["de","Deutsch (German)"],
            ["fr","Français (French)"],
            ["es","Español (Spanish)"],
            ["it","Italiano (Italian)"],
            ["pt","Português (Portuguese)"],
            ["nl","Nederlands (Dutch)"],
            ["pl","Polski (Polish)"],
            ["sl","Slovenščina (Slovenian)"],
            ["sr","Srpski (Serbian)"],
            ["bs","Bosanski (Bosnian)"],
            ["mk","Македонски (Macedonian)"],
            ["sq","Shqip (Albanian)"],
          ]}
        />
      </GeneralCard>

      <GeneralCard icon={<Monitor size={20}/>} title="Appearance" subtitle="Personalize the look and feel of Ghost FTP.">
        <SelectRow label="Theme" value={s.appTheme} onChange={(v)=>s.setAppTheme(v as typeof s.appTheme)} options={themeOptions}/>
        <AccentRow value={s.accentColor} onChange={s.setAccentColor}/>
        <SelectRow label="Interface Density" value={s.paneDensity} onChange={(v)=>s.setPaneDensity(v as "comfortable"|"compact")} options={[["comfortable","Comfortable"],["compact","Compact"]]}/>
        <ToggleRow label="Image previews" checked={s.remoteImagePreviews === "on"} onChange={(v)=>s.setRemoteImagePreviews(v ? "on" : "off")}/>
      </GeneralCard>

      <GeneralCard icon={<ArrowDownUp size={20}/>} title="Transfers" subtitle="Set default options for file transfers.">
        <SelectRow label="Overwrite Behavior" value={s.overwritePolicy} onChange={(v)=>s.setOverwritePolicy(v as "overwrite"|"skip"|"rename")} options={[["overwrite","Overwrite"],["rename","Rename duplicate"],["skip","Skip existing"]]}/>
        <ToggleRow label="Prompt before overwrite" checked={s.promptOnOverwrite} onChange={s.setPromptOnOverwrite}/>
        <ToggleRow label="Open transfer queue" checked={s.autoOpenTransferPanel} onChange={s.setAutoOpenTransferPanel}/>
      </GeneralCard>

      <GeneralCard icon={<Wifi size={20}/>} title="Connection" subtitle="Configure connection behavior and reliability.">
        <NumberRow label="Default SFTP Port" value={s.defaultPort} min={1} max={65535} fallback={22} onChange={s.setDefaultPort}/>
        <DesktopNotificationsToggle/>
        <ToggleRow label="Notify only when unfocused" checked={s.notifications.unfocusedOnly} onChange={(v)=>s.setNotifications({...s.notifications,unfocusedOnly:v})}/>
      </GeneralCard>

      <GeneralCard icon={<ShieldCheck size={20}/>} title="Security & Privacy" subtitle="Protect your data and control your privacy.">
        <ToggleRow label="No tracking" checked locked/>
        <ToggleRow label="No analytics or telemetry" checked locked/>
        <ToggleRow label="OS keychain credentials" checked locked/>
      </GeneralCard>

      <GeneralUpdatesCard/>

      <GeneralIntegrationsCard/>

      <GeneralCard icon={<Keyboard size={20}/>} title="Shortcuts" subtitle="Keyboard shortcuts for common actions.">
        <div className="grid grid-cols-[1fr_auto] gap-x-5 gap-y-2 text-[11px]">
          <span>Open Quick Connect</span><kbd>Ctrl + Q</kbd>
          <span>Start Transfer</span><kbd>Ctrl + Enter</kbd>
          <span>Command Palette</span><kbd>Ctrl + K</kbd>
        </div>
      </GeneralCard>
    </div>
  );
}

function GeneralCard({ icon, title, subtitle, children }: { icon: React.ReactNode; title: string; subtitle: string; children: React.ReactNode }) {
  return (
    <section className="ghost-general-card rounded-lg border border-border bg-[#071f35] p-3">
      <div className="mb-2 flex items-center gap-2">
        <span className="text-accent">{icon}</span>
        <div className="min-w-0">
          <div className="font-semibold text-accent">{title}</div>
          <div className="truncate text-[10.5px] text-text-muted">{subtitle}</div>
        </div>
      </div>
      <div className="space-y-2">{children}</div>
    </section>
  );
}

function GeneralIntegrationsCard() {
  const s=useSettings();
  const [busy,setBusy]=useState(false);

  useEffect(()=>{
    let active=true;
    void ipc.pathStatus().then((status)=>{
      if(active) s.setShellIntegration(status.managed);
    }).catch(()=>{});
    return()=>{ active=false; };
  },[]);

  const setShell=async(enabled:boolean)=>{
    if(busy)return;
    setBusy(true);
    try{
      const status=enabled?await ipc.pathAdd():await ipc.pathRemove();
      s.setShellIntegration(status.managed);
    }finally{
      setBusy(false);
    }
  };

  return (
    <GeneralCard icon={<Plug size={20}/>} title="Integrations" subtitle="Extend Ghost FTP with system integrations.">
      <ToggleRow label="Shell integration" checked={s.shellIntegration} onChange={(v)=>void setShell(v)} locked={busy}/>
      <ToggleRow label="File associations" checked={s.fileAssociations} onChange={s.setFileAssociations} locked/>
      <DesktopNotificationsToggle/>
    </GeneralCard>
  );
}

function GeneralUpdatesCard() {
  const status=useUpdater((x)=>x.status);
  const version=useUpdater((x)=>x.version);
  const check=useUpdater((x)=>x.check);
  const busy=status==="checking"||status==="downloading";
  return (
    <GeneralCard icon={<RefreshCw size={20}/>} title="Updates" subtitle="Keep Ghost FTP current.">
      <div className="grid grid-cols-[150px_1fr] items-center gap-3 text-[12px]">
        <span className="text-text-muted">Update Channel</span>
        <div className="rounded-md border border-border bg-[#051929] px-3 py-2 text-text">Stable (Recommended)</div>
      </div>
      <div className="flex items-center justify-between gap-3">
        <span className="min-w-0 truncate text-[10.5px] text-text-dim">
          {status==="available" ? `Ghost FTP ${version ?? "update"} available` : status==="checking" ? "Checking…" : "Automatic update checks supported"}
        </span>
        <button className="ghost-mini-button" disabled={busy} onClick={()=>void check(false)}>
          <RefreshCw size={13}/> Check
        </button>
      </div>
    </GeneralCard>
  );
}

function Card({ icon, title, subtitle, children }: { icon: React.ReactNode; title: string; subtitle: string; children: React.ReactNode }) {
  return <section className="rounded-lg border border-border bg-[#071f35] p-4 shadow-[inset_0_1px_0_rgba(255,255,255,.02)]"><div className="mb-3 flex items-center gap-2"><span className="text-accent">{icon}</span><div><div className="font-semibold text-accent">{title}</div><div className="text-[11px] text-text-muted">{subtitle}</div></div></div><div className="space-y-3">{children}</div></section>;
}

function LanguageCard({ locale, setLocale }: { locale: string; setLocale: (value: any) => void }) { const options = [['en','English (English)'],['hr','Hrvatski (Croatian)'],['de','Deutsch (German)'],['fr','Français (French)'],['es','Español (Spanish)'],['it','Italiano (Italian)'],['pt','Português (Portuguese)'],['nl','Nederlands (Dutch)'],['pl','Polski (Polish)'],['sl','Slovenščina (Slovenian)'],['sr','Srpski (Serbian)'],['bs','Bosanski (Bosnian)'],['mk','Македонски (Macedonian)'],['sq','Shqip (Albanian)']] as [string,string][]; return <Card icon={<Globe2 size={20}/>} title="Language" subtitle="Choose your preferred application language."><SelectRow label="Primary Language" value={locale} onChange={setLocale} options={options}/><div className="text-[10px] text-text-dim">English is the primary language. Changes take effect after restart.</div></Card> }
function AppearanceCard() {
  const s = useSettings();
  const themeOptions = APP_THEMES.map(
    (theme) => [theme.value, theme.label] as [string, string]
  );

  return (
    <Card icon={<Monitor size={20}/>} title="Appearance" subtitle="Personalize the look and feel of Ghost FTP.">
      <SelectRow
        label="Theme"
        value={s.appTheme}
        onChange={(value) => s.setAppTheme(value as typeof s.appTheme)}
        options={themeOptions}
      />
      <AccentRow value={s.accentColor} onChange={s.setAccentColor} />
      <SelectRow
        label="Interface Density"
        value={s.paneDensity}
        onChange={(value) => s.setPaneDensity(value as "comfortable" | "compact")}
        options={[
          ["comfortable", "Comfortable"],
          ["compact", "Compact"],
        ]}
      />
      <SelectRow
        label="File Browser"
        value={s.browserLayout}
        onChange={(value) => s.setBrowserLayout(value as "single" | "dual")}
        options={[
          ["dual", "Local + Remote (Dual Pane)"],
          ["single", "Server Focused"],
        ]}
      />
      <SelectRow
        label="File View"
        value={s.paneViewMode}
        onChange={(value) => s.setPaneViewMode(value as "list" | "details" | "grid")}
        options={[
          ["details", "Details"],
          ["list", "List"],
          ["grid", "Grid"],
        ]}
      />
      <ToggleRow
        label="Show hidden files"
        checked={s.showHiddenFiles}
        onChange={s.setShowHiddenFiles}
      />
      <ToggleRow
        label="Remote image previews"
        checked={s.remoteImagePreviews === "on"}
        onChange={(value) => s.setRemoteImagePreviews(value ? "on" : "off")}
      />
    </Card>
  );
}
function PerformanceCard() {
  const s = useSettings();
  return (
    <Card icon={<Sparkles size={20}/>} title="Performance" subtitle="Adjust transfer performance and resilience.">
      <RangeRow
        label="Concurrent Transfers"
        value={s.transferConcurrency}
        min={1}
        max={32}
        onChange={s.setTransferConcurrency}
      />
      <RangeRow
        label="Max Retry Attempts"
        value={s.maxRetryAttempts}
        min={0}
        max={8}
        onChange={s.setMaxRetryAttempts}
      />
      <NumberRow
        label="Speed Limit (KiB/s)"
        value={s.transferThrottleKbps}
        min={0}
        fallback={0}
        onChange={s.setTransferThrottleKbps}
      />
      <div className="-mt-1 text-[10px] text-text-dim">Use 0 for unlimited bandwidth.</div>
      <ToggleRow
        label="Delta synchronization"
        checked={s.deltaSync}
        onChange={s.setDeltaSync}
      />
    </Card>
  );
}
function TransfersCard() {
  const s = useSettings();
  return (
    <Card icon={<ArrowDownUp size={20}/>} title="Transfers" subtitle="Set default file-transfer behavior.">
      <SelectRow
        label="Overwrite Behavior"
        value={s.overwritePolicy}
        onChange={(v) => s.setOverwritePolicy(v as "overwrite" | "skip" | "rename")}
        options={[
          ["overwrite", "Overwrite"],
          ["rename", "Rename duplicate"],
          ["skip", "Skip existing"],
        ]}
      />
      <ToggleRow
        label="Prompt before overwrite"
        checked={s.promptOnOverwrite}
        onChange={s.setPromptOnOverwrite}
      />
      <ToggleRow
        label="Open transfer queue automatically"
        checked={s.autoOpenTransferPanel}
        onChange={s.setAutoOpenTransferPanel}
      />
      <TextRow
        label="Download Folder"
        value={s.defaultDownloadFolder}
        placeholder="System Downloads folder"
        onChange={s.setDefaultDownloadFolder}
      />
      <TextRow
        label="Default Editor"
        value={s.defaultEditor}
        placeholder="System default application"
        onChange={s.setDefaultEditor}
      />
    </Card>
  );
}
function TerminalCard() {
  const s = useSettings();
  return (
    <Card
      icon={<TerminalSquare size={20}/>}
      title="Terminal"
      subtitle="Tune the integrated SSH terminal for daily server work."
    >
      <SelectRow
        label="Terminal Theme"
        value={s.terminalTheme}
        onChange={(value) => s.setTerminalTheme(value as typeof s.terminalTheme)}
        options={[
          ["dark", "Dark"],
          ["light", "Light"],
          ["dracula", "Dracula"],
          ["solarized-dark", "Solarized Dark"],
          ["gruvbox-dark", "Gruvbox Dark"],
          ["onedark", "One Dark"],
          ["rosepine", "Rosé Pine"],
          ["everforest", "Everforest"],
        ]}
      />
      <NumberRow
        label="Font Size"
        value={s.terminalFontSize}
        min={8}
        max={32}
        fallback={13}
        onChange={s.setTerminalFontSize}
      />
      <TextRow
        label="Font Family"
        value={s.terminalFontFamily}
        placeholder="JetBrains Mono, Cascadia Code, monospace"
        onChange={s.setTerminalFontFamily}
      />
      <NumberRow
        label="Scrollback Lines"
        value={s.terminalScrollback}
        min={100}
        max={100000}
        fallback={5000}
        onChange={s.setTerminalScrollback}
      />
      <ToggleRow
        label="Copy selection automatically"
        checked={s.terminalCopyOnSelect}
        onChange={s.setTerminalCopyOnSelect}
      />
      <ToggleRow
        label="Inline command suggestions"
        checked={s.terminalSuggestions}
        onChange={s.setTerminalSuggestions}
      />
    </Card>
  );
}

function DesktopNotificationsToggle() {
  const s = useSettings();
  const [busy, setBusy] = useState(false);
  const setEnabled = async (enabled: boolean) => {
    if (!enabled) {
      s.setNotifications({ ...s.notifications, enabled: false });
      return;
    }
    if (busy) return;
    setBusy(true);
    try {
      const granted = await requestDesktopNotificationPermission();
      s.setNotifications({ ...s.notifications, enabled: granted });
    } finally {
      setBusy(false);
    }
  };
  return <ToggleRow label={busy ? "Waiting for notification permission…" : "Desktop notifications"} checked={s.notifications.enabled} onChange={(v)=>void setEnabled(v)} locked={busy}/>;
}

function ConnectionCard() {
  const s = useSettings();
  return (
    <Card icon={<Wifi size={20}/>} title="Connection" subtitle="Configure connection behavior and notifications.">
      <NumberRow
        label="Default SFTP Port"
        value={s.defaultPort}
        min={1}
        max={65535}
        fallback={22}
        onChange={s.setDefaultPort}
      />
      <DesktopNotificationsToggle />
      <ToggleRow
        label="Notify only when unfocused"
        checked={s.notifications.unfocusedOnly}
        onChange={(v) => s.setNotifications({ ...s.notifications, unfocusedOnly: v })}
      />
    </Card>
  );
}
function SecurityCard() { return <Card icon={<ShieldCheck size={20}/>} title="Security & Privacy" subtitle="Protect your data and control your privacy."><ToggleRow label="No tracking" checked locked/><ToggleRow label="No analytics or telemetry" checked locked/><ToggleRow label="Store credentials in OS keychain" checked locked/></Card> }
function UpdatesCard() { const status=useUpdater((x)=>x.status); const version=useUpdater((x)=>x.version); const error=useUpdater((x)=>x.error); const check=useUpdater((x)=>x.check); const download=useUpdater((x)=>x.downloadAndInstall); const restart=useUpdater((x)=>x.restart); const busy=status==='checking'||status==='downloading'; const label=status==='checking'?'Checking for updates…':status==='available'?`Ghost FTP ${version ?? 'update'} is available.`:status==='downloading'?'Downloading and verifying update…':status==='ready'?'Update ready — restart to finish.':status==='error'?(error??'Update check failed.'):'Ready to check the official Ghost FTP update service.'; return <Card icon={<RefreshCw size={20}/>} title="Updates" subtitle="Choose how Ghost FTP updates itself."><div className="grid grid-cols-[150px_1fr] items-center gap-3 text-[12px]"><span className="text-text-muted">Update Channel</span><div className="rounded-md border border-border bg-[#051929] px-3 py-2 text-text">Stable (Recommended)</div></div><div className="text-[11px] text-text-muted" aria-live="polite">{label}</div><div className="flex flex-wrap gap-2"><button className="ghost-mini-button" disabled={busy} onClick={()=>void check(false)}><RefreshCw size={13}/> Check for Updates</button>{status==='available'&&<button className="ghost-mini-button" onClick={()=>void download()}>Download &amp; install</button>}{status==='ready'&&<button className="ghost-mini-button" onClick={()=>void restart()}>Restart now</button>}</div></Card> }
function IntegrationsCard() {
  const s=useSettings();
  const [shellBusy,setShellBusy]=useState(false);
  const [shellDetail,setShellDetail]=useState<string>("Adds Ghost FTP's managed CLI directory to your user PATH.");
  useEffect(()=>{
    let active=true;
    void ipc.pathStatus().then((status)=>{
      if(!active)return;
      s.setShellIntegration(status.managed);
      if(status.detail)setShellDetail(status.detail);
    }).catch(()=>{});
    return()=>{active=false};
  },[]);
  const setShell=async(enabled:boolean)=>{
    if(shellBusy)return;
    setShellBusy(true);
    try{
      const status=enabled?await ipc.pathAdd():await ipc.pathRemove();
      s.setShellIntegration(status.managed);
      setShellDetail(status.detail ?? (status.managed?'Shell integration enabled.':'Shell integration disabled.'));
    }catch(error){
      setShellDetail(error instanceof Error?error.message:String(error));
    }finally{setShellBusy(false)}
  };
  return <Card icon={<Plug size={20}/>} title="Integrations" subtitle="Extend Ghost FTP with system integrations.">
    <DesktopNotificationsToggle/>
    <ToggleRow label="Shell integration" checked={s.shellIntegration} onChange={(v)=>void setShell(v)} locked={shellBusy}/>
    <div className="-mt-1 text-[10px] leading-4 text-text-dim">{shellDetail}</div>
    <ToggleRow label="File associations" checked={s.fileAssociations} onChange={s.setFileAssociations} locked/>
    <div className="-mt-1 text-[10px] leading-4 text-text-dim">File associations are installed only by signed production packages; this control stays locked in source/dev builds.</div>
  </Card>
}

function AppearancePanel(){return <div className="max-w-3xl"><AppearanceCard/></div>}
function TransfersPanel(){return <div className="grid max-w-4xl grid-cols-2 gap-4"><PerformanceCard/><TransfersCard/><div className="col-span-2"><TerminalCard/></div></div>}
function ConnectionPanel(){return <div className="max-w-3xl"><ConnectionCard/></div>}
function SecurityPanel(){return <div className="max-w-3xl"><SecurityCard/></div>}
function UpdatesPanel(){return <div className="max-w-3xl"><UpdatesCard/></div>}
function IntegrationsPanel(){return <div className="max-w-3xl"><IntegrationsCard/></div>}
function LanguagePanel({ locale, setLocale }: { locale: string; setLocale: (value: any) => void }){return <div className="max-w-3xl"><LanguageCard locale={locale} setLocale={setLocale}/></div>}
function ShortcutsPanel(){return <Card icon={<Keyboard size={20}/>} title="Keyboard Shortcuts" subtitle="Core Ghost FTP shortcuts."><div className="grid grid-cols-[1fr_auto] gap-x-8 gap-y-2 text-[12px]"><span>New connection</span><kbd>Ctrl + N</kbd><span>Settings</span><kbd>Ctrl + ,</kbd><span>Transfer queue</span><kbd>Ctrl + Shift + T</kbd><span>Command palette</span><kbd>Ctrl + K</kbd></div></Card>}

function Nav({
  section,
  current,
  set,
  icon,
  label,
}: {
  section: Section;
  current: Section;
  set: (v: Section) => void;
  icon: React.ReactNode;
  label: string;
}) {
  const active = current === section;
  return (
    <button
      type="button"
      aria-current={active ? "page" : undefined}
      onClick={() => set(section)}
      className={`mb-1 flex items-center gap-3 rounded-md border px-3 py-2.5 text-left ${active ? "border-accent/45 bg-accent/15 text-white" : "border-transparent text-text-muted hover:bg-bg-hover hover:text-white"}`}
    >
      <span className="[&>svg]:h-[18px] [&>svg]:w-[18px]">{icon}</span>
      <span>{label}</span>
    </button>
  );
}

function SelectRow({
  label,
  value,
  onChange,
  options,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  options: [string, string][];
}) {
  return (
    <label className="grid grid-cols-[150px_1fr] items-center gap-3 text-[12px]">
      <span className="text-text-muted">{label}</span>
      <span className="relative">
        <select
          className="w-full appearance-none rounded-md border border-border bg-[#051929] px-3 py-2 pr-8"
          value={value}
          onChange={(e) => onChange(e.target.value)}
        >
          {options.map(([optionValue, optionLabel]) => (
            <option key={optionValue} value={optionValue}>{optionLabel}</option>
          ))}
        </select>
        <ChevronDown size={13} className="pointer-events-none absolute right-2.5 top-2.5 text-text-dim"/>
      </span>
    </label>
  );
}

function AccentRow({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  const color = /^#[0-9a-f]{6}$/i.test(value) ? value : "#3b82f6";
  return (
    <div className="grid grid-cols-[150px_1fr] items-center gap-3 text-[12px]">
      <span className="text-text-muted">Accent Color</span>
      <div className="flex min-w-0 items-center gap-2">
        <input
          type="color"
          value={color}
          aria-label="Accent color"
          onChange={(event) => onChange(event.target.value)}
          className="h-8 w-12 cursor-pointer rounded border border-border bg-[#051929] p-1"
        />
        <span className="min-w-0 flex-1 truncate font-mono text-[11px] text-text-dim">
          {value || "Theme default"}
        </span>
        <button
          type="button"
          className="ghost-mini-button"
          disabled={!value}
          onClick={() => onChange("")}
        >
          Reset
        </button>
      </div>
    </div>
  );
}

function ToggleRow({
  label,
  checked,
  onChange,
  locked = false,
}: {
  label: string;
  checked: boolean;
  onChange?: (v: boolean) => void;
  locked?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-4 text-[12px]">
      <span className="text-text-muted">{label}</span>
      <button
        type="button"
        disabled={locked || !onChange}
        aria-label={label}
        aria-pressed={checked}
        aria-readonly={locked || !onChange ? true : undefined}
        onClick={() => onChange?.(!checked)}
        className={`relative h-5 w-9 rounded-full border ${checked ? "border-accent bg-accent-strong" : "border-border bg-[#041522]"}`}
      >
        <span className={`absolute top-[2px] h-3.5 w-3.5 rounded-full bg-white transition-all ${checked ? "left-[18px]" : "left-[2px]"}`}/>
      </button>
    </div>
  );
}

function RangeRow({
  label,
  value,
  min,
  max,
  onChange,
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (v: number) => void;
}) {
  return (
    <label className="grid grid-cols-[150px_1fr_28px] items-center gap-3 text-[12px]">
      <span className="text-text-muted">{label}</span>
      <input type="range" min={min} max={max} value={value} onChange={(e) => onChange(Number(e.target.value))}/>
      <strong>{value}</strong>
    </label>
  );
}

function NumberRow({
  label,
  value,
  onChange,
  min,
  max,
  fallback,
}: {
  label: string;
  value: number;
  onChange: (v: number) => void;
  min?: number;
  max?: number;
  fallback: number;
}) {
  return (
    <label className="grid grid-cols-[150px_1fr] items-center gap-3 text-[12px]">
      <span className="text-text-muted">{label}</span>
      <input
        className="rounded-md border border-border bg-[#051929] px-3 py-2"
        type="number"
        min={min}
        max={max}
        value={value}
        onChange={(e) => {
          const parsed = Number(e.target.value);
          onChange(Number.isFinite(parsed) ? parsed : fallback);
        }}
      />
    </label>
  );
}

function TextRow({
  label,
  value,
  placeholder,
  onChange,
}: {
  label: string;
  value: string;
  placeholder?: string;
  onChange: (v: string) => void;
}) {
  return (
    <label className="grid grid-cols-[150px_1fr] items-center gap-3 text-[12px]">
      <span className="text-text-muted">{label}</span>
      <input
        className="min-w-0 rounded-md border border-border bg-[#051929] px-3 py-2"
        type="text"
        value={value}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
      />
    </label>
  );
}

