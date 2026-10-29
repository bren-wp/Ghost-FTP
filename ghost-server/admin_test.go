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
