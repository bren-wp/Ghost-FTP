(() => {
  "use strict";
  const byId = (id) => document.getElementById(id);
  let token = "";
  async function refresh() {
    byId("message").textContent = "Reading local server state...";
    try {
      const response = await fetch("/api/status", {
        method: "GET", cache: "no-store",
        headers: {"Authorization": "Bearer " + token, "Accept": "application/json"}
      });
      if (!response.ok) throw new Error("Authentication failed or server unavailable (" + response.status + ")");
      const value = await response.json();
      byId("status").textContent = value.status === "running" ? "Running · TLS" : "Unavailable";
      byId("clients").textContent = String(value.active_connections);
      byId("uploads").textContent = value.create_only_uploads_enabled ? "Enabled" : "Disabled";
      byId("uptime").textContent = String(value.uptime_seconds) + " s";
      byId("refresh").disabled = false;
      byId("message").textContent = "Live status retrieved securely from this server.";
    } catch (error) {
      byId("message").textContent = error instanceof Error ? error.message : "Could not load status";
      byId("refresh").disabled = true;
    }
  }
  byId("unlock").addEventListener("click", () => {
    token = byId("token").value;
    byId("token").value = "";
    if (token.length < 32) { byId("message").textContent = "Enter the separate 32+ character admin token."; return; }
    void refresh();
  });
  byId("refresh").addEventListener("click", () => { if (token.length >= 32) void refresh(); });
  window.addEventListener("pagehide", () => { token = ""; });
})();
