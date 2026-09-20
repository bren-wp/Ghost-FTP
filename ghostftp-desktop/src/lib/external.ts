import { ipc } from "./ipc";
import { toast } from "../stores/toastStore";

const OFFICIAL_ORIGIN = "https://ghostftp.com/";

/** Open a Ghost FTP website resource in the OS default browser, never in a WebView popup. */
export function openOfficialUrl(path = "/"): void {
  try {
    const url = new URL(path, OFFICIAL_ORIGIN);
    if (url.protocol !== "https:" || (url.hostname !== "ghostftp.com" && !url.hostname.endsWith(".ghostftp.com"))) {
      throw new Error("Blocked non-Ghost FTP external URL");
    }
    void ipc.openExternalUrl(url.toString()).catch((error) => {
      toast.error("Could not open link", error instanceof Error ? error.message : String(error));
    });
  } catch (error) {
    toast.error("Could not open link", error instanceof Error ? error.message : String(error));
  }
}
