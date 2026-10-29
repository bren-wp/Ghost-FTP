package ghostserver

import (
 "crypto/sha256"
 "crypto/subtle"
 "embed"
 "encoding/json"
 "io/fs"
 "net/http"
 "strings"
 "time"
)

//go:embed admin-assets/*
var adminAssets embed.FS

// The admin dashboard is independently token-gated and *loopback-only*.
// No cookies, URL credentials, permissive CORS or user file paths are exposed.
func (s *Server) adminHandler() http.Handler {
 assets,err:=fs.Sub(adminAssets,"admin-assets")
 if err!=nil{panic("embedded admin assets unavailable")}
 mux:=http.NewServeMux()
 mux.Handle("/admin/",http.StripPrefix("/admin/",http.FileServer(http.FS(assets))))
 mux.HandleFunc("/api/status",func(w http.ResponseWriter,r *http.Request){
  if r.Method!=http.MethodGet{w.Header().Set("Allow","GET");http.Error(w,"Method not allowed",http.StatusMethodNotAllowed);return}
  authorization:=r.Header.Get("Authorization")
  if !strings.HasPrefix(authorization,"Bearer ") {
   w.Header().Set("WWW-Authenticate","Bearer")
   http.Error(w,"Authentication required",http.StatusUnauthorized);return
  }
  digest:=sha256.Sum256([]byte(strings.TrimPrefix(authorization,"Bearer ")))
  if subtle.ConstantTimeCompare(digest[:],s.adminTokenHash[:])!=1{
   w.Header().Set("WWW-Authenticate","Bearer")
   http.Error(w,"Authentication required",http.StatusUnauthorized);return
  }
  w.Header().Set("Content-Type","application/json; charset=utf-8")
  _=json.NewEncoder(w).Encode(struct {
   Status string `json:"status"`
   ExplicitTLS bool `json:"explicit_tls_required"`
   WriteEnabled bool `json:"create_only_uploads_enabled"`
   MaxUploadBytes int64 `json:"max_upload_bytes"`
   ActiveConnections int64 `json:"active_connections"`
   UptimeSeconds int64 `json:"uptime_seconds"`
  }{
   Status:"running",ExplicitTLS:true,WriteEnabled:s.allowUploads,
   MaxUploadBytes:s.maxUploadBytes,ActiveConnections:s.activeSessions.Load(),
   UptimeSeconds:int64(time.Since(s.startedAt).Seconds()),
  })
 })
 mux.HandleFunc("/",func(w http.ResponseWriter,r *http.Request){
  if r.URL.Path!="/" {http.NotFound(w,r);return}
  http.Redirect(w,r,"/admin/",http.StatusFound)
 })
 return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  // Check Host as well as the bound loopback address to block DNS rebinding.
  if r.Host!=s.adminListen{http.Error(w,"Unknown administration host",http.StatusForbidden);return}
  w.Header().Set("Cache-Control","no-store")
  w.Header().Set("X-Content-Type-Options","nosniff")
  w.Header().Set("X-Frame-Options","DENY")
  w.Header().Set("Referrer-Policy","no-referrer")
  w.Header().Set("Content-Security-Policy","default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
  mux.ServeHTTP(w,r)
 })
}
