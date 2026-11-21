// Ghost FTP by Brendigo
import { Cloud, Download, Plus, Server } from "lucide-react";

interface Props {
  disabled: boolean;
  onNewSite: () => void;
  onConnectCloud: () => void;
  onImportSites: () => void;
}

export function SiteManagerEmptyState({
  disabled,
  onNewSite,
  onConnectCloud,
  onImportSites,
}: Props) {
  return (
    <div className="ghost-site-manager-workspace ghost-site-manager-workspace--empty">
      <section className="ghost-site-manager-table ghost-site-manager-table--empty" aria-label="Saved sites">
        <div className="ghost-sites-empty">
          <div className="ghost-sites-empty-card">
            <div className="ghost-sites-empty-icon" aria-hidden="true">
              <Server size={46} />
            </div>
            <p className="ghost-sites-empty-eyebrow">0 saved sites</p>
            <h2 className="ghost-sites-empty-heading">Add your first site</h2>
            <p className="ghost-sites-empty-description">
              Keep your FTP, FTPS, SFTP and compatible cloud connections organized.
              Save a site once and reconnect without entering its details again.
            </p>
            <div className="ghost-sites-empty-actions">
              <button type="button" className="ghost-primary-button" disabled={disabled} onClick={onNewSite}>
                <Plus size={16} aria-hidden="true" /> New Site
              </button>
              <button type="button" className="ghost-mini-button" disabled={disabled} onClick={onConnectCloud}>
                <Cloud size={16} aria-hidden="true" /> Connect Cloud
              </button>
              <button type="button" className="ghost-mini-button" disabled={disabled} onClick={onImportSites}>
                <Download size={16} aria-hidden="true" /> Import Sites
              </button>
            </div>
            <p className="ghost-sites-empty-hint">
              Already have connection profiles? Use Import Sites to bring them into Ghost FTP.
            </p>
          </div>
        </div>
      </section>
    </div>
  );
}
