(() => {
  "use strict";
  // No cookies, storage, analytics, external requests or URL-based secrets.
  // The only shared state is the real /api/v1/status response from this instance.
  const byId = (id) => document.getElementById(id);
  const pages = {
    overview: ["Overview", "Live local status for this Ghost FTP Server instance."],
    sessions: ["Sessions", "Authenticated FTP and FTPS connection telemetry."],
    security: ["Security", "Effective encryption and mutation policy from the server."],
    limits: ["Limits", "Active transfer limits and upload mode."],
    diagnostics: ["Diagnostics", "Versioned read-only diagnostics and provenance."]
  };
  let token = "";
  let authenticated = false;
  let timer = 0;
  let activeRequest = null;
  let liveData = null;
  let generation = 0;
  const safeText = (id, value) => { byId(id).textContent = String(value); };
  const displayBool = (value) => value === true ? "Enabled" : value === false ? "Disabled" : "—";
  const formatDuration = (seconds) => {
    if (!Number.isSafeInteger(seconds) || seconds < 0) return "—";
    const d = Math.floor(seconds / 86400);
    const h = Math.floor(seconds % 86400 / 3600);
    const m = Math.floor(seconds % 3600 / 60);
    const s = seconds % 60;
    return d ? d + "d " + h + "h" : h ? h + "h " + m + "m" : m ? m + "m " + s + "s" : s + "s";
  };
  const formatBytes = (bytes) => {
    if (!Number.isSafeInteger(bytes) || bytes < 0) return "—";
    if (bytes < 1024) return bytes + " B";
    const units = ["KiB","MiB","GiB","TiB"];
    let size = bytes, i = -1;
    do { size /= 1024; i++; } while (size >= 1024 && i < units.length - 1);
    return size.toLocaleString("en-US", {maximumFractionDigits: 1}) + " " + units[i];
  };
  const setMessage = (message, error = false) => {
    safeText("message", message);
    byId("message").classList.toggle("error", error);
  };
  const switchPage = (name) => {
    if (!Object.prototype.hasOwnProperty.call(pages, name)) return;
    document.querySelectorAll(".nav-item[data-page]").forEach((button) => {
      const selected = button.dataset.page === name;
      button.classList.toggle("selected", selected);
      if (selected) button.setAttribute("aria-current", "page");
      else button.removeAttribute("aria-current");
    });
    document.querySelectorAll(".page[data-view]").forEach((section) => {
      const selected = section.dataset.view === name;
      section.hidden = !selected;
      section.classList.toggle("active", selected);
    });
    safeText("breadcrumb", pages[name][0]);
    safeText("page-title", pages[name][0]);
    safeText("page-description", pages[name][1]);
  };
  const clearDisplayedState = () => {
    for (const id of ["clients","uploads","uptime","overview-tls","overview-destructive","overview-contract",
      "session-count","security-tls","security-uploads","security-destructive",
      "limit-upload","limit-mode","diagnostic-contract","diagnostic-version","diagnostic-refresh"]) safeText(id, "—");
    safeText("status","Locked");
    safeText("top-status","LOCKED");
    safeText("footer-state","Local status · Locked");
    byId("top-status").classList.remove("live");
  };
  const lock = (message = "Local session locked.") => {
    generation++;
    if (activeRequest) activeRequest.abort();
    activeRequest = null;
    if (timer) window.clearInterval(timer);
    timer = 0;
    token = "";
    authenticated = false;
    liveData = null;
    byId("token").value = "";
    byId("refresh").disabled = true;
    byId("lock").disabled = true;
    byId("unlock").disabled = false;
    clearDisplayedState();
    setMessage(message);
  };
  const render = (data) => {
    safeText("status", data.status === "running" ? "Running · FTPS" : "Unavailable");
    safeText("top-status", data.status === "running" ? "RUNNING" : "UNAVAILABLE");
    byId("top-status").classList.toggle("live", data.status === "running");
    safeText("clients", data.active_connections);
    safeText("uploads", displayBool(data.create_only_uploads_enabled));
    safeText("uptime", formatDuration(data.uptime_seconds));
    safeText("overview-tls", displayBool(data.explicit_tls_required));
    safeText("overview-destructive", displayBool(data.destructive_mutations_enabled));
    safeText("overview-contract", "v" + data.contract_version);
    safeText("session-count", data.active_connections);
    safeText("security-tls", data.explicit_tls_required ? "Required" : "Not verified");
    safeText("security-uploads", displayBool(data.create_only_uploads_enabled));
    safeText("security-destructive", displayBool(data.destructive_mutations_enabled));
    safeText("limit-upload", formatBytes(data.max_upload_bytes));
    safeText("limit-mode", data.create_only_uploads_enabled ? "Create-only" : "Disabled");
    safeText("diagnostic-contract", "v" + data.contract_version);
    safeText("diagnostic-version", data.server_version);
    safeText("diagnostic-refresh", new Date().toLocaleTimeString());
    safeText("footer-state", "Live server · " + data.active_connections + " active");
  };
  const isValid = (data) => {
    return data && typeof data === "object" && data.contract_version === 1 &&
      typeof data.server_version === "string" && data.server_version.length > 0 && data.server_version.length < 80 &&
      (data.status === "running" || data.status === "stopped") &&
      typeof data.explicit_tls_required === "boolean" &&
      typeof data.create_only_uploads_enabled === "boolean" &&
      typeof data.destructive_mutations_enabled === "boolean" &&
      Number.isSafeInteger(data.max_upload_bytes) && data.max_upload_bytes >= 0 &&
      Number.isSafeInteger(data.active_connections) && data.active_connections >= 0 &&
      Number.isSafeInteger(data.uptime_seconds) && data.uptime_seconds >= 0;
  };
  const refresh = async () => {
    if (!authenticated || activeRequest) return;
    const requestId = generation;
    const controller = new AbortController();
    activeRequest = controller;
    byId("refresh").disabled = true;
    try {
      const response = await fetch("/api/v1/status", {
        method: "GET", cache: "no-store", credentials: "omit", signal: controller.signal,
        headers: {"Authorization":"Bearer " + token, "Accept":"application/json"}
      });
      if (requestId !== generation) return;
      if (response.status === 401 || response.status === 403) {
        lock("Authentication rejected. Re-enter your private token.");
        byId("token").focus();
        return;
      }
      if (!response.ok) throw new Error("Local server temporarily unavailable (HTTP " + response.status + ")");
      const result = await response.json();
      if (!isValid(result)) throw new Error("Unsupported or malformed server status contract");
      liveData = result;
      render(liveData);
      setMessage("Authenticated live status updated from the local server.");
    } catch (error) {
      if (requestId !== generation || controller.signal.aborted) return;
      // Fail visibly: stale values must never masquerade as live telemetry.
      clearDisplayedState();
      byId("top-status").textContent = "OFFLINE";
      byId("footer-state").textContent = "Local status unavailable";
      setMessage(error instanceof Error ? error.message : "Could not load live status", true);
    } finally {
      if (requestId === generation) {
        activeRequest = null;
        byId("refresh").disabled = !authenticated;
      }
    }
  };
  const unlock = () => {
    const candidate = byId("token").value;
    byId("token").value = "";
    if (candidate.length < 32) { setMessage("Enter the separate 32+ character administrator token.", true); return; }
    lock("Authenticating…");
    token = candidate;
    authenticated = true;
    byId("lock").disabled = false;
    byId("unlock").disabled = true;
    void refresh();
    timer = window.setInterval(() => { if (authenticated && !document.hidden) void refresh(); }, 15000);
  };
  document.querySelectorAll(".nav-item[data-page]").forEach((button) => {
    button.addEventListener("click", () => switchPage(button.dataset.page));
  });
  byId("unlock").addEventListener("click", unlock);
  byId("token").addEventListener("keydown", (event) => { if (event.key === "Enter") unlock(); });
  byId("refresh").addEventListener("click", () => { if (authenticated) void refresh(); });
  byId("lock").addEventListener("click", () => lock());
  document.addEventListener("visibilitychange", () => { if (!document.hidden && authenticated) void refresh(); });
  window.addEventListener("pagehide", () => lock("Local session closed."));
  switchPage("overview");
  clearDisplayedState();
})();
