import { GhostMark } from "./GhostBrand";
import { PRODUCT_VERSION_BADGE } from "@/lib/release";

/** Native Windows/Linux desktop launch branding. No fake progress, data or telemetry. */
export function StartupSplash() {
  return (
    <div className="ghost-startup-splash" role="status" aria-label="Opening Ghost FTP">
      <div className="ghost-startup-card">
        <div className="ghost-startup-brand">
          <GhostMark size={108} />
          <div className="ghost-startup-wordmark">
            <div className="ghost-startup-name">Ghost<span>FTP</span></div>
            <div className="ghost-startup-tagline">MORE THAN TRANSFER. TOTAL CONTROL.</div>
          </div>
        </div>
        <div className="ghost-startup-rule" aria-hidden="true" />
        <div className="ghost-startup-footer">
          <span>YOUR FILES. YOUR SERVERS.</span>
          <span className="ghost-startup-version">PREVIEW {PRODUCT_VERSION_BADGE}</span>
        </div>
      </div>
      <span className="sr-only">Ghost FTP is opening the desktop workspace.</span>
    </div>
  );
}
