import { useRef, useState } from "react";
import {
  ChevronRight,
  FileText,
  Globe2,
  HelpCircle,
  LifeBuoy,
  RefreshCw,
  ShieldCheck,
} from "lucide-react";
import { GhostMark } from "./GhostBrand";
import { useUpdater } from "@/stores/updaterStore";
import {
  PRODUCT_BUILD,
  PRODUCT_EULA_URL,
  PRODUCT_RELEASE_DATE,
  PRODUCT_VERSION_DISPLAY,
} from "@/lib/release";
import { APP_LOCALES, tr } from "@/lib/i18n";
import { ipc } from "@/lib/ipc";
import { useDialog } from "@/hooks/useDialog";

// Canonical public documentation and support live with the open-source release.
// PRODUCT_SITE remains reserved for the separately configured secure update service.
const PROJECT_REPOSITORY_URL = "https://github.com/bren-wp/Ghost-FTP";

type AboutTab = "about" | "updates" | "help" | "privacy";
interface Props { onClose: () => void; initialTab?: Exclude<AboutTab, "privacy"> }

export function AboutDialog({ onClose, initialTab = "about" }: Props) {
  const panelRef = useRef<HTMLDivElement>(null);
  const [tab, setTab] = useState<AboutTab>(initialTab);
  useDialog(panelRef, { onClose, trapFocus: false });

  return (
    <div className="ghost-workspace-view ghost-standalone-view bg-[#041425]" role="region" aria-label="About Ghost FTP">
      <div ref={panelRef} className="ghost-about flex h-full w-full flex-col overflow-hidden bg-bg-panel">
        <div className="ghost-about-body flex min-h-0 flex-1 flex-col">
        <nav className="ghost-about-nav ghost-about-tabs flex shrink-0 items-center gap-1 overflow-x-auto border-b border-border bg-[#061a2d] px-3 py-2" aria-label="Help and About sections">
          <AboutNav active={tab === "about"} icon={<Globe2 size={17}/>} label={tr("About")} onClick={() => setTab("about")} />
          <AboutNav active={tab === "updates"} icon={<RefreshCw size={17}/>} label={tr("Updates")} onClick={() => setTab("updates")} />
          <AboutNav active={tab === "help"} icon={<HelpCircle size={17}/>} label={tr("Help Center")} onClick={() => setTab("help")} />
          <AboutNav active={tab === "privacy"} icon={<ShieldCheck size={17}/>} label={tr("Privacy")} onClick={() => setTab("privacy")} />
        </nav>

        <main className="ghost-about-content min-h-0 min-w-0 flex-1 overflow-y-auto p-4">
          {tab !== "about" && (
            <div className="mb-3">
              <div className="text-lg font-semibold">
                {tab === "updates" ? "Ghost FTP Updates" : tab === "privacy" ? tr("Privacy") : "Ghost FTP Help Center"}
              </div>
              <div className="text-[12px] text-text-muted">More Than Transfer. Total Control.</div>
            </div>
          )}

          {tab === "about" && <AboutContent onNavigate={setTab} />}
          {tab === "updates" && <UpdatesContent />}
          {tab === "help" && <HelpContent />}
          {tab === "privacy" && <PrivacyContent />}
        </main>
        </div>
      </div>
    </div>
  );
}

function AboutContent({ onNavigate }: { onNavigate: (tab: AboutTab) => void }) {
  const check = useUpdater((s) => s.check);
  return (
    <div className="ghost-about-reference-screen">
      <header className="ghost-about-reference-heading">
        <div className="min-w-0">
          <div className="ghost-about-eyebrow">GHOST FTP / {currentPlatform()}</div>
          <h1 className="text-2xl font-semibold tracking-tight">{tr("Help Center")} &amp; {tr("About")}</h1>
          <p className="text-[12px] text-text-muted">Product details, assistance and privacy information.</p>
        </div>
        <button type="button" className="ghost-primary-button shrink-0" onClick={() => { onNavigate("updates"); void check(false); }}>
          <RefreshCw size={15}/> {tr("Check for Updates")}
        </button>
      </header>

      <div className="ghost-about-reference-grid">
        <section className="ghost-about-reference-card" aria-label="Ghost FTP product information">
          <div className="ghost-about-reference-brand">
            <span className="ghost-about-mark" aria-hidden="true"><GhostMark size={86}/></span>
            <div className="min-w-0">
              <h2 className="text-[34px] font-bold tracking-tight">Ghost <span className="text-accent">FTP</span></h2>
              <p className="ghost-about-tagline">MORE THAN TRANSFER. TOTAL CONTROL.</p>
            </div>
          </div>

          <div className="ghost-about-version-pill" aria-label={`Ghost FTP version ${PRODUCT_VERSION_DISPLAY}`}>
            {tr("Version")} {PRODUCT_VERSION_DISPLAY}
          </div>
          <div className="ghost-about-reference-divider"/>
          <h3 className="text-base font-semibold">Designed for complete control</h3>
          <p className="ghost-about-reference-description">Ghost FTP provides file management and FTP, FTPS and SFTP connectivity with control over your own connections. Transfers do not require analytics or a Ghost FTP account.</p>
          <dl className="ghost-about-reference-details">
            <Meta label="Publisher" value="Brendigo"/>
            <Meta label="Platform" value={currentPlatform()}/>
            <Meta label={tr("Version")} value={PRODUCT_VERSION_DISPLAY}/>
            <Meta label={tr("Build")} value={PRODUCT_BUILD}/>
            <Meta label={tr("Release Date")} value={PRODUCT_RELEASE_DATE}/>
            <Meta label={tr("Languages")} value={`${APP_LOCALES.length} interface languages`}/>
          </dl>
        </section>

        <aside className="ghost-about-reference-resources" aria-label="Support and resources">
          <h3 className="ghost-about-resources-heading">{tr("Resources, documentation and support.")}</h3>
          <div className="ghost-about-resources-list">
            <OfficialLinkRow icon={<Globe2 size={18}/>} title={tr("Documentation")} subtitle="Guides and installation instructions" url={`${PROJECT_REPOSITORY_URL}/tree/main/docs`}/>
            <LinkRow icon={<HelpCircle size={18}/>} title={tr("Help Center")} subtitle="Connections and troubleshooting" onClick={() => onNavigate("help")}/>
            <OfficialLinkRow icon={<LifeBuoy size={18}/>} title="Official Support" subtitle="Open the GitHub issue tracker" url={`${PROJECT_REPOSITORY_URL}/issues`}/>
            <OfficialLinkRow icon={<ShieldCheck size={18}/>} title={tr("Privacy Policy")} subtitle="Privacy policy in the project repository" url={`${PROJECT_REPOSITORY_URL}/blob/main/docs/legal/PRIVACY.md`}/>
            <OfficialLinkRow icon={<FileText size={18}/>} title="Terms of use / EULA" subtitle="Software licence" url={PRODUCT_EULA_URL}/>
            <OfficialLinkRow icon={<Globe2 size={18}/>} title={tr("Project repository")} subtitle="Source code and verified releases" url={PROJECT_REPOSITORY_URL}/>
            <LinkRow icon={<RefreshCw size={18}/>} title={tr("Updates")} subtitle="Check the update status" onClick={() => onNavigate("updates")}/>
          </div>
        </aside>
      </div>
    </div>
  );
}

function UpdatesContent() {
  const status = useUpdater((s) => s.status);
  const offered = useUpdater((s) => s.version);
  const error = useUpdater((s) => s.error);
  const check = useUpdater((s) => s.check);
  const download = useUpdater((s) => s.downloadAndInstall);
  const restart = useUpdater((s) => s.restart);
  const label = status === "checking" ? "Checking for updates…"
    : status === "available" ? `Ghost FTP ${offered ?? "update"} is available.`
    : status === "downloading" ? "Downloading and verifying update…"
    : status === "ready" ? "Update is ready. Restart Ghost FTP to finish."
    : status === "error" ? (error ?? "Update check failed.")
    : "No update check has been run in this view yet.";
  return <div className="rounded-lg border border-border bg-[#071f35] p-6"><div className="mb-4 flex items-center gap-3"><RefreshCw size={30} className="text-accent"/><div><div className="text-[18px] font-semibold">Ghost FTP {PRODUCT_VERSION_DISPLAY}</div><div className="text-text-muted">Stable channel · {currentPlatform()}</div></div></div><p className="max-w-2xl text-[13px] leading-6 text-text-muted">Ghost FTP securely checks the official update service and installs only verified packages. Technical delivery details stay hidden from the application interface.</p><p className="mt-3 text-[12px] text-text-muted" aria-live="polite">{label}</p><div className="mt-5 flex gap-2"><button className="ghost-primary-button" disabled={status === "checking" || status === "downloading"} onClick={() => void check(false)}><RefreshCw size={14}/> {tr("Check for Updates")}</button>{status === "available" && <button className="ghost-primary-button" onClick={() => void download()}>{tr("Download & install")}</button>}{status === "ready" && <button className="ghost-primary-button" onClick={() => void restart()}>{tr("Restart now")}</button>}</div></div>;
}

function currentPlatform() {
  const ua = navigator.userAgent.toLowerCase();
  if (ua.includes("windows")) return "Windows";
  if (ua.includes("linux")) return "Linux";
  return "Desktop";
}

function HelpContent() {
  return <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
    <InfoCard icon={<Globe2/>} title={tr("Connections")} text="Create or edit a saved site, then use Test Connection before connecting." />
    <InfoCard icon={<FileText/>} title={tr("Transfers")} text="Use Transfers for queue status, retry, scheduling and transfer history." />
    <InfoCard icon={<LifeBuoy/>} title="Troubleshooting" text="Connection and transfer errors stay visible in the app so you can act on the real failure." />
    <InfoCard icon={<RefreshCw/>} title={tr("Updates")} text="Open the Updates tab here to check the official Ghost FTP update channel." />
  </div>;
}

function PrivacyContent() {
  return <div className="max-w-3xl space-y-4 text-[13px] leading-6 text-text-muted">
    <div className="rounded-lg border border-border bg-[#071f35] p-5"><div className="mb-2 flex items-center gap-2 text-[16px] font-semibold text-text"><ShieldCheck size={20} className="text-accent"/>Privacy-first by default</div><p>Ghost FTP is designed to keep connection data and application settings under your control. The application does not require analytics or telemetry to transfer files.</p></div>
    <div className="rounded-lg border border-border bg-[#071f35] p-5"><div className="font-semibold text-text">Credentials and connections</div><p className="mt-2">Saved credentials are handled by Ghost FTP's local credential storage. Connections go to the server or provider you configure; the file manager does not need a Ghost FTP relay to perform ordinary FTP, FTPS or SFTP transfers.</p></div>
    <div className="rounded-lg border border-border bg-[#071f35] p-5"><div className="font-semibold text-text">Diagnostics</div><p className="mt-2">Operational errors are shown inside the app so failures are visible instead of silently ignored. Review Settings → Security and Settings → Advanced for privacy-sensitive options.</p></div>
  </div>;
}

function AboutNav({ active, icon, label, onClick }: { active: boolean; icon: React.ReactNode; label: string; onClick: () => void }) { return <button onClick={onClick} className={`flex shrink-0 items-center gap-2 rounded-md border px-3 py-2 text-left ${active ? "border-accent/45 bg-accent/15 text-white" : "border-transparent text-text-muted hover:bg-bg-hover hover:text-white"}`}>{icon}<span>{label}</span></button> }
function Meta({ label, value }: { label: string; value: string }) {
  return <div className="ghost-about-reference-detail"><dt>{label}</dt><dd>{value}</dd></div>;
}
function LinkRow({ icon, title, subtitle, onClick }: { icon: React.ReactNode; title: string; subtitle: string; onClick: () => void }) { return <button onClick={onClick} className="flex w-full items-center gap-3 border-t border-border-subtle py-3 text-left first:border-t-0"><span className="text-accent">{icon}</span><span className="min-w-0 flex-1"><span className="block font-medium text-accent">{title}</span><span className="block text-[11px] text-text-muted">{subtitle}</span></span><ChevronRight size={13} className="text-text-dim"/></button> }
function OfficialLinkRow({ icon, title, subtitle, url }: { icon: React.ReactNode; title: string; subtitle: string; url: string }) {
  const [error, setError] = useState<string | null>(null);
  const open = async () => {
    setError(null);
    try {
      await ipc.openExternalUrl(url);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    }
  };
  return <>
    <LinkRow icon={icon} title={title} subtitle={subtitle} onClick={() => void open()} />
    {error && <div className="pb-2 text-[11px] text-danger" role="alert">{error}</div>}
  </>;
}

function InfoCard({ icon, title, text }: { icon: React.ReactNode; title: string; text: string }) { return <div className="rounded-lg border border-border bg-[#071f35] p-5 text-left"><div className="mb-3 text-accent">{icon}</div><div className="text-[16px] font-semibold">{title}</div><div className="mt-1 text-[12px] leading-5 text-text-muted">{text}</div></div> }
