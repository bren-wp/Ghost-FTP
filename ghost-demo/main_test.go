package main

import (
 "bytes"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
 "time"
)

func req(t *testing.T,h http.Handler,method,path,host,body,origin,action string) *httptest.ResponseRecorder {
 t.Helper()
 r:=httptest.NewRequest(method,"http://127.0.0.1:28291"+path,strings.NewReader(body))
 if host!="" {r.Host=host}
 if method=="POST" {r.Header.Set("Content-Type","application/json");r.Header.Set("X-Ghost-Demo",action)}
 if origin!="" {r.Header.Set("Origin",origin)}
 w:=httptest.NewRecorder();h.ServeHTTP(w,r);return w
}
func fixture(t *testing.T)*demo{
 t.Helper()
 d:=&demo{start:time.Now()}
 if _,err:=d.handler("127.0.0.1:28291");err!=nil{t.Fatal(err)}
 return d
}
func TestOfflineDemoIsRealEmbeddedSelfTest(t *testing.T){
 if err:=selfTest();err!=nil {t.Fatal(err)}
}
func TestDemoFixturesStayExplicitlyFictionalAndSafe(t *testing.T){
 d:=fixture(t);h,_:=d.handler("127.0.0.1:28291")
 w:=req(t,h,"GET","/api/v1/state","","","","")
 if w.Code!=200 {t.Fatal(w.Code)}
 var state state
 if err:=json.Unmarshal(w.Body.Bytes(),&state);err!=nil{t.Fatal(err)}
 if !state.Demo||len(state.Sites)<3||len(state.Files)<10{t.Fatalf("not a rich in-memory demo: %+v",state)}
 for _,site:=range state.Sites {
  if !strings.Contains(site.Detail,".invalid") || !strings.Contains(site.Detail,"SIMULATED"){t.Errorf("real-looking endpoint found: %+v",site)}
 }
 if strings.Contains(w.Body.String(),"password")||strings.Contains(w.Body.String(),"127.0.0.1:2121"){t.Fatal("secret or real service URL embedded")}
 if w.Header().Get("Access-Control-Allow-Origin")!="" {t.Fatal("unsafe CORS")}
 if !strings.Contains(w.Header().Get("Content-Security-Policy"),"default-src 'none'"){t.Fatal("weak CSP")}
 if req(t,h,"GET","/api/v1/state","attacker.invalid","","","").Code!=403{t.Fatal("DNS rebinding host accepted")}
}
func TestDemoUploadDownloadCancellationAndNoActualFiles(t *testing.T){
 d:=fixture(t);h,_:=d.handler("127.0.0.1:28291")
 input:=`{"file_id":"f1","direction":"upload"}`
 for _,tc:=range []struct{action,origin string; want int}{
  {"","",403},{"simulation-only","https://attacker.invalid",403},
  {"simulation-only","http://127.0.0.1:28291",202},
 }{
  w:=req(t,h,"POST","/api/v1/transfer","",input,tc.origin,tc.action)
  if w.Code!=tc.want{t.Fatalf("unexpected status for %q %q: %d",tc.action,tc.origin,w.Code)}
 }
 w:=req(t,h,"POST","/api/v1/transfer","",`{"file_id":"f2","direction":"download"}`,"","simulation-only")
 if w.Code!=202{t.Fatal(w.Code,w.Body)}
 if n:=len(d.snapshot().Transfers);n!=2{t.Fatalf("expected two synthetic transfers; got %d",n)}
 w=req(t,h,"POST","/api/v1/cancel","",`{"id":1}`,"","simulation-only")
 if w.Code!=200 || !bytes.Contains(w.Body.Bytes(),[]byte("cancelled")){t.Fatal(w.Code,w.Body)}
 state:=d.snapshot()
 if state.Transfers[0].State!="cancelled"||state.Transfers[1].State!="simulating"{t.Fatalf("incorrect simulated statuses %+v",state.Transfers)}
 if !strings.Contains(w.Header().Get("Content-Security-Policy"),"script-src 'self'"){t.Fatal("demo CSP missing")}
}
func TestDemoRejectsUnknownMutationsAndMalformedPayload(t *testing.T){
 d:=fixture(t);h,_:=d.handler("127.0.0.1:28291")
 for _,payload:=range []string{
  `{"file_id":"not-a-file","direction":"upload"}`,
  `{"file_id":"f1","direction":"delete"}`,
  `{"file_id":"f1","direction":"upload","unexpected":"x"}`,
  `{"file_id":"f1","direction":"upload"}{"file_id":"f1"}`,
 }{
  w:=req(t,h,"POST","/api/v1/transfer","",payload,"","simulation-only")
  if w.Code!=400{t.Errorf("%q: got %d",payload,w.Code)}
 }
 if len(d.snapshot().Transfers)!=0{t.Fatal("invalid action created transfer")}
}
