package ghostserver

import (
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "path/filepath"
 "strings"
 "testing"
)

func adminTestBackend(t *testing.T) *Server {
 t.Helper()
 crt,key:=certFiles(t)
 srv,err:=New(Config{
  Root:t.TempDir(),Username:"qa",Password:"long-private-test-password",
  Certificate:crt,PrivateKey:key,
  AdminToken:strings.Repeat("Z",48),AdminListen:"127.0.0.1:2821",
 })
 if err!=nil{t.Fatal(err)}
 // os.Root holds a real Windows directory handle. Release it before
 // testing.TempDir cleanup or Windows cannot remove the fixture directory.
 t.Cleanup(func(){ if srv.uploadRoot!=nil{_ = srv.uploadRoot.Close()} })
 return srv
}

func adminRequest(t *testing.T,srv *Server,method,target,token,host string) *httptest.ResponseRecorder {
 t.Helper()
 req:=httptest.NewRequest(method,"http://127.0.0.1:2821"+target,nil)
 if token!=""{req.Header.Set("Authorization","Bearer "+token)}
 if host!=""{req.Host=host}
 result:=httptest.NewRecorder()
 srv.adminHandler().ServeHTTP(result,req)
 return result
}

func TestAdminRequiresSeparateStrongBearerTokenAndLoopbackHost(t *testing.T) {
 srv:=adminTestBackend(t)
 secret:=strings.Repeat("Z",48)
 for _,tc:=range []struct{method,path,token,host string;status int}{
  {"GET","/api/status","","",http.StatusUnauthorized},
  {"GET","/api/status","wrong-token","",http.StatusUnauthorized},
  {"GET","/api/status",secret,"evil.example",http.StatusForbidden},
  {"POST","/api/status",secret,"",http.StatusMethodNotAllowed},
  {"GET","/api/status",secret,"",http.StatusOK},
  {"GET","/api/v1/status",secret,"",http.StatusOK},
  {"GET","/api/v1/status","","",http.StatusUnauthorized},
  {"POST","/api/v1/status",secret,"",http.StatusMethodNotAllowed},
  {"GET","/admin/","","",http.StatusOK},
 } {
  response:=adminRequest(t,srv,tc.method,tc.path,tc.token,tc.host)
  if response.Code!=tc.status{t.Fatalf("%s %s status=%d expected %d: %s",tc.method,tc.path,response.Code,tc.status,response.Body)}
  if response.Header().Get("Content-Security-Policy")=="" && tc.status!=http.StatusForbidden {
   t.Fatal("missing restrictive admin Content Security Policy")
  }
 }
 status:=adminRequest(t,srv,"GET","/api/status",secret,"")
 var payload map[string]any
 if err:=json.Unmarshal(status.Body.Bytes(),&payload);err!=nil{t.Fatal(err)}
 if payload["status"]!="running" || payload["explicit_tls_required"]!=true || payload["create_only_uploads_enabled"]!=false {
  t.Fatalf("inaccurate live admin state: %v",payload)
 }
 if payload["active_connections"]!=float64(0){t.Fatalf("unexpected active sessions: %v",payload)}
 if payload["contract_version"]!=float64(1) || payload["server_version"]!=BuildVersion {t.Fatalf("status API contract out of sync: %v",payload)}
 v1:=adminRequest(t,srv,"GET","/api/v1/status",secret,"")
 var v1payload map[string]any
 if err:=json.Unmarshal(v1.Body.Bytes(),&v1payload);err!=nil{t.Fatal(err)}
 for k,v:=range payload { if v1payload[k]!=v { t.Fatalf("legacy and versioned status diverge: field %s: %v != %v",k,v,v1payload[k]) } }
 for _,untrusted:=range []string{"long-private-test-password",secret,srv.root,filepath.Base(srv.root)}{
  if strings.Contains(status.Body.String(),untrusted){t.Fatalf("leaked private configuration: %q",untrusted)}
 }
}

func TestAdminRejectsPublicListenerAndWeakTokens(t *testing.T) {
 crt,key:=certFiles(t)
 base:=Config{Root:t.TempDir(),Username:"qa",Password:"long-private-test-password",Certificate:crt,PrivateKey:key,AdminToken:strings.Repeat("x",48),AdminListen:"0.0.0.0:2821"}
 if _,err:=New(base);err==nil{t.Fatal("admin exposed to public interfaces")}
 base.AdminListen="127.0.0.1:2821";base.AdminToken="short"
 if _,err:=New(base);err==nil{t.Fatal("weak admin token accepted")}
 base.AdminToken=strings.Repeat("Z",48);base.AdminListen="example.com:2821"
 if _,err:=New(base);err==nil{t.Fatal("hostname-based non-loopback admin listener accepted")}
}

func TestAdminStaticDashboardDoesNotEmbedCredentials(t *testing.T) {
 srv:=adminTestBackend(t)
 response:=adminRequest(t,srv,"GET","/admin/","","")
 if response.Code!=http.StatusOK{t.Fatalf("dashboard status %d",response.Code)}
 body:=response.Body.String()
 if !strings.Contains(body,"Ghost FTP")||!strings.Contains(body,"Administrator token"){
  t.Fatalf("missing actual admin dashboard: %q",body)
 }
 if strings.Contains(body,strings.Repeat("Z",48))||strings.Contains(body,"long-private-test-password"){
  t.Fatal("admin HTML embedded private credentials")
 }
 if !strings.Contains(response.Header().Get("Content-Security-Policy"),"frame-ancestors 'none'"){
  t.Fatal("missing anti-clickjacking policy")
 }
}

func TestAdminPremiumAssetsKeepLocalBrandAndRestrictiveCSP(t *testing.T) {
 srv:=adminTestBackend(t)
 for _,name:=range []string{"style.css","app.js","ghostftp-symbol.svg"}{
  resp:=adminRequest(t,srv,"GET","/admin/"+name,"","")
  if resp.Code!=http.StatusOK {t.Fatalf("%s status=%d",name,resp.Code)}
  if resp.Body.Len()<200 {t.Fatalf("%s empty or placeholder",name)}
  if strings.Contains(resp.Body.String(),strings.Repeat("Z",48)){t.Fatalf("%s leaked admin token",name)}
  csp:=resp.Header().Get("Content-Security-Policy")
  if !strings.Contains(csp,"img-src 'self'") || !strings.Contains(csp,"default-src 'none'") {
   t.Fatalf("%s does not permit local-only logo under strict CSP: %s",name,csp)
  }
 }
 resp:=adminRequest(t,srv,"GET","/admin/","","")
 for _,needle:=range []string{"ghostftp-symbol.svg","data-page=\"overview\"","data-page=\"sessions\"","data-page=\"security\"","data-page=\"limits\"","data-page=\"diagnostics\"","aria-live=\"polite\""} {
  if !strings.Contains(resp.Body.String(),needle){t.Fatalf("server workspace missing %s",needle)}
 }
}
