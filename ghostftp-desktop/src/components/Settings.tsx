import { useEffect, useRef, useState } from "react";
import {
  ChevronDown,
  Globe2,
  Keyboard,
  Monitor,
  Palette,
  Plug,
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
  resetSettingsToDefaults,
  APP_THEMES,
  useSettings,
} from "@/stores/settingsStore";
import { getLocale, saveLocale } from "@/lib/i18n";
import { ipc } from "@/lib/ipc";
import { useDialog } from "@/hooks/useDialog";
import { requestDesktopNotificationPermission } from "@/lib/notifications";
import { SyncSettings } from "./SyncSettings";
import { toastError } from "@/lib/errors";
import { toast } from "@/stores/toastStore";

interface Props { onClose: () => void; initialSection?: Section }
type Section = "appearance" | "language" | "transfers" | "connection" | "security" | "advanced" | "sync";

const SECTION_DESCRIPTION: Record<Section, string> = {
  appearance: "Theme, density and file-browser layout.",
  language: "Choose the application language used throughout Ghost FTP.",
  transfers: "Transfer concurrency, speed limits, retries and resume behavior.",
  connection: "Reconnect, keep-alive and timeout behavior.",
  security: "Credentials, privacy and local history controls.",
  advanced: "System integrations and keyboard shortcuts.",
  sync: "Create and manage real folder synchronization pairs.",
};

export function Settings({ onClose, initialSection = "appearance" }: Props) {
  const panelRef = useRef<HTMLDivElement>(null);
  const [section, setSection] = useState<Section>(initialSection);
  const syncOnly = initialSection === "sync";
  const [pendingLocale, setPendingLocale] = useState(getLocale());
  const setLocaleNow = (value: any) => {
    setPendingLocale(value);
    saveLocale(value);
  };
  const done = () => onClose();
  const reset = () => {
    resetSettingsToDefaults();
    setPendingLocale("en");
    saveLocale("en");
  };

  useDialog(panelRef, { onClose: done, trapFocus: false });

  return (
    <div className="ghost-workspace-view ghost-standalone-view bg-[#041425]" role="region" aria-label="Ghost FTP Preferences">
      <div ref={panelRef} className="ghost-preferences flex h-full w-full flex-col overflow-hidden bg-bg-panel">
        <div className="ghost-preferences-body flex min-h-0 flex-1 flex-col">
        {!syncOnly && (
          <div className="ghost-preferences-nav ghost-settings-tabs flex shrink-0 items-center gap-1 overflow-x-auto border-b border-border bg-[#061a2d] px-3 py-2">
            <Nav section="appearance" current={section} set={setSection} icon={<Monitor/>} label="Appearance"/>
            <Nav section="language" current={section} set={setSection} icon={<Globe2/>} label="Language"/>
            <Nav section="transfers" current={section} set={setSection} icon={<ArrowDownUp/>} label="Transfers"/>
            <Nav section="connection" current={section} set={setSection} icon={<Wifi/>} label="Connection"/>
            <Nav section="security" current={section} set={setSection} icon={<ShieldCheck/>} label="Security"/>
            <Nav section="advanced" current={section} set={setSection} icon={<SlidersHorizontal/>} label="Advanced"/>
          </div>
        )}

        <main className="flex min-h-0 min-w-0 flex-1 flex-col">
          <div className="ghost-preferences-heading flex h-[78px] shrink-0 items-center gap-3 border-b border-border px-5">
            <Settings2 size={28} className="text-accent"/>
            <div>
              <div className="text-[22px] font-semibold">{section === "sync" ? "Sync & Backup" : section.charAt(0).toUpperCase() + section.slice(1)}</div>
              <div className="text-[12px] text-text-muted">{SECTION_DESCRIPTION[section]}</div>
            </div>
          </div>
          <div className="ghost-preferences-content flex-1 overflow-y-auto p-4">

            {section === "appearance" && <AppearancePanel/>}
            {section === "language" && <LanguagePanel locale={pendingLocale} setLocale={setLocaleNow}/>} 
            {section === "transfers" && <TransfersPanel/>}
            {section === "connection" && <ConnectionPanel/>}
            {section === "security" && <SecurityPanel/>}
            {section === "advanced" && <AdvancedPanel/>}
            {section === "sync" && <div className="max-w-5xl"><SyncSettings/></div>}
          </div>
          <div className="ghost-preferences-actions flex h-[58px] shrink-0 items-center border-t border-border bg-[#061a2d] px-4">
            {!syncOnly && <button className="ghost-mini-button" onClick={reset}><RotateCcw size={14}/> Reset to Defaults</button>}
            <div className="flex-1"/>
            <button className="ghost-primary-button" onClick={done}>Done</button>
          </div>
        </main>
        </div>
      </div>
    </div>
  );
}

function Card({ icon, title, subtitle, children }: { icon: React.ReactNode; title: string; subtitle: string; children: React.ReactNode }) {
  return <section className="rounded-lg border border-border bg-[#071f35] p-4 shadow-[inset_0_1px_0_rgba(255,255,255,.02)]"><div className="mb-3 flex items-center gap-2"><span className="text-accent">{icon}</span><div><div className="font-semibold text-accent">{title}</div><div className="text-[11px] text-text-muted">{subtitle}</div></div></div><div className="space-y-3">{children}</div></section>;
}

function LanguageCard({ locale, setLocale }: { locale: string; setLocale: (value: any) => void }) {
  const options = [['en','English (English)'],['hr','Hrvatski (Croatian)'],['de','Deutsch (German)'],['fr','Français (French)'],['es','Español (Spanish)'],['it','Italiano (Italian)'],['pt','Português (Portuguese)'],['nl','Nederlands (Dutch)'],['pl','Polski (Polish)'],['sl','Slovenščina (Slovenian)'],['sr','Srpski (Serbian)'],['bs','Bosanski (Bosnian)'],['mk','Македонски (Macedonian)'],['sq','Shqip (Albanian)']] as [string,string][];
  return <Card icon={<Globe2 size={20}/>} title="Language" subtitle="Choose your preferred application language."><SelectRow label="Primary Language" value={locale} onChange={setLocale} options={options}/><div className="text-[10px] text-text-dim">Language changes are saved immediately. English remains the primary fallback language.</div></Card>;
}
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
      if (!granted) {
        toast.warning(
          "Desktop notifications remain off",
          "Ghost FTP did not receive notification permission from the operating system."
        );
      }
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
    }).catch((error)=>{
      if(active){
        const detail=error instanceof Error?error.message:String(error);
        setShellDetail(detail);
        toastError(error, "Couldn't read shell integration status");
      }
    });
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
function LanguagePanel({ locale, setLocale }: { locale: string; setLocale: (value: any) => void }){return <div className="max-w-3xl"><LanguageCard locale={locale} setLocale={setLocale}/></div>}
function TransfersPanel(){return <div className="grid max-w-4xl grid-cols-2 gap-4"><PerformanceCard/><TransfersCard/><div className="col-span-2"><TerminalCard/></div></div>}
function ConnectionPanel(){return <div className="max-w-3xl"><ConnectionCard/></div>}
function SecurityPanel(){return <div className="max-w-3xl"><SecurityCard/></div>}
function ShortcutsCard(){return <Card icon={<Keyboard size={20}/>} title="Keyboard Shortcuts" subtitle="Core Ghost FTP shortcuts."><div className="grid grid-cols-[1fr_auto] gap-x-8 gap-y-2 text-[12px]"><span>New connection</span><kbd>Ctrl + N</kbd><span>Settings</span><kbd>Ctrl + ,</kbd><span>Open Transfers</span><kbd>Ctrl + T</kbd><span>Command palette</span><kbd>Ctrl + K</kbd></div></Card>}
function AdvancedPanel(){return <div className="grid max-w-5xl gap-4 xl:grid-cols-2"><IntegrationsCard/><ShortcutsCard/></div>}

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
      className={`flex shrink-0 items-center gap-2 rounded-md border px-3 py-2 text-left ${active ? "border-accent/45 bg-accent/15 text-white" : "border-transparent text-text-muted hover:bg-bg-hover hover:text-white"}`}
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

