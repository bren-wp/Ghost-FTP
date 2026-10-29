package main

import (
 "context"
 "embed"
 "encoding/json"
  "flag"
 "fmt"
 "io"
 "io/fs"
 "net"
 "net/http"
 "net/http/httptest"
 "os"
 "os/exec"
 "os/signal"
 "runtime"
 "strings"
 "sync"
 "syscall"
 "time"
)

// CI supplies the same version as the Ghost FTP clients/server.
var version = "dev"

//go:embed assets/*
var frontend embed.FS

type demoFile struct { ID string `json:"id"`; Name string `json:"name"`; Folder string `json:"folder"`; Bytes int64 `json:"bytes"`; Type string `json:"type"` }
type demoSite struct { ID string `json:"id"`; Name string `json:"name"`; Protocol string `json:"protocol"`; Detail string `json:"detail"` }
type transfer struct { ID int64 `json:"id"`; File string `json:"file"`; Direction string `json:"direction"`; Bytes int64 `json:"bytes"`; Progress int `json:"progress"`; State string `json:"state"`; Started time.Time `json:"-"` }
type state struct {
 Demo bool `json:"demo"`
 Version string `json:"version"`
 Sites []demoSite `json:"sites"`
 Files []demoFile `json:"files"`
 Transfers []transfer `json:"transfers"`
 SyncPreview []string `json:"sync_preview"`
}
type demo struct { mu sync.Mutex; seq int64; transfers []transfer; start time.Time }
var sites=[]demoSite{
 {"studio","Creative Studio","FTPS","demo.studio.invalid · SIMULATED"},
 {"backup","Backup Vault","SFTP","demo.backup.invalid · SIMULATED"},
 {"archive","Project Archive","WebDAV","demo.archive.invalid · SIMULATED"},
 {"cloud","Team Cloud","S3","demo.cloud.invalid · SIMULATED"},
}
var sampleFiles=[]demoFile{
 {"f1","Brand-Guidelines.pdf","/Design",4732134,"PDF"},{"f2","Homepage-Mockup.fig","/Design",13872455,"Design"},
 {"f3","Product-Photos.zip","/Media",52274560,"Archive"},{"f4","launch-hero.webp","/Media",1973420,"Image"},
 {"f5","Project-Overview.md","/Projects",15921,"Document"},{"f6","roadmap.xlsx","/Projects",287544,"Spreadsheet"},
 {"f7","campaign-report.pdf","/Reports",1164573,"PDF"},{"f8","server-backup.tar.gz","/Backups",78361422,"Archive"},
 {"f9","customers-demo.csv","/Reports",12487,"Sample data"},{"f10","banner-promo.png","/Media",1957600,"Image"},
 {"f11","notes.txt","/Projects",8462,"Document"},{"f12","website-release.zip","/Projects",26843547,"Archive"},
}

func (d *demo) snapshot() state {
 d.mu.Lock(); defer d.mu.Unlock()
 result:=state{Demo:true,Version:version,Sites:sites,Files:sampleFiles,
   SyncPreview:[]string{"3 files queued for simulated upload","2 remote files would be downloaded","0 destructive changes (additive preview only)"}}
 result.Transfers=make([]transfer,len(d.transfers))
 copy(result.Transfers,d.transfers)
 for i:=range result.Transfers {
  t:=&result.Transfers[i]
  if t.State=="cancelled" {continue}
  p:=int(time.Since(t.Started)/(100*time.Millisecond))
  if p>100 {p=100}
  if p<0 {p=0}
  t.Progress=p
  if p==100 {t.State="completed"} else {t.State="simulating"}
 }
 return result
}
func setHeaders(w http.ResponseWriter) {
 h:=w.Header()
 h.Set("Content-Security-Policy","default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
 h.Set("Cache-Control","no-store")
 h.Set("X-Content-Type-Options","nosniff")
 h.Set("X-Frame-Options","DENY")
 h.Set("Referrer-Policy","no-referrer")
}
func (d *demo) handler(host string) (http.Handler,error) {
 filesys,err:=fs.Sub(frontend,"assets")
 if err!=nil{return nil,err}
 mux:=http.NewServeMux()
 mux.Handle("/app/",http.StripPrefix("/app/",http.FileServer(http.FS(filesys))))
 mux.HandleFunc("/",func(w http.ResponseWriter,r *http.Request) {
  if r.URL.Path!="/" {http.NotFound(w,r);return}
  http.Redirect(w,r,"/app/",http.StatusSeeOther)
 })
 mux.HandleFunc("/api/v1/state",func(w http.ResponseWriter,r *http.Request) {
  if r.Method!=http.MethodGet {w.Header().Set("Allow","GET");http.Error(w,"Method not allowed",405);return}
  w.Header().Set("Content-Type","application/json")
  _=json.NewEncoder(w).Encode(d.snapshot())
 })
 mux.HandleFunc("/api/v1/transfer",func(w http.ResponseWriter,r *http.Request) {
  if !allowedDemoAction(w,r,host){return}
  var input struct {FileID string `json:"file_id"`; Direction string `json:"direction"`}
  if !decodeAction(w,r,&input){return}
  if input.Direction!="upload"&&input.Direction!="download" {http.Error(w,"Invalid demo direction",400);return}
  var found *demoFile
  for i:=range sampleFiles {if sampleFiles[i].ID==input.FileID {found=&sampleFiles[i];break}}
  if found==nil {http.Error(w,"Unknown demo file",400);return}
  d.mu.Lock()
  if len(d.transfers)>=100 {d.mu.Unlock();http.Error(w,"Demo queue full",429);return}
  d.seq++
  id:=d.seq
  d.transfers=append(d.transfers,transfer{ID:id,File:found.Name,Direction:input.Direction,Bytes:found.Bytes,State:"simulating",Started:time.Now()})
  d.mu.Unlock()
  w.Header().Set("Content-Type","application/json")
  w.WriteHeader(http.StatusAccepted)
  _=json.NewEncoder(w).Encode(map[string]any{"simulated":true,"id":id})
 })
 mux.HandleFunc("/api/v1/cancel",func(w http.ResponseWriter,r *http.Request) {
  if !allowedDemoAction(w,r,host){return}
  var input struct{ ID int64 `json:"id"` }
  if !decodeAction(w,r,&input){return}
  d.mu.Lock(); found:=false
  for i:=range d.transfers {if d.transfers[i].ID==input.ID{
   if d.transfers[i].State!="cancelled"{d.transfers[i].State="cancelled"}
   found=true; break
  }}
  d.mu.Unlock()
  if !found {http.Error(w,"Unknown demo transfer",404);return}
  w.Header().Set("Content-Type","application/json")
  _,_=w.Write([]byte(`{"simulated":true,"cancelled":true}`))
 })
 return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  setHeaders(w)
  if r.Host!=host {http.Error(w,"Invalid local demo host",403);return}
  mux.ServeHTTP(w,r)
 }),nil
}
func allowedDemoAction(w http.ResponseWriter,r *http.Request,host string) bool {
 if r.Method!=http.MethodPost {w.Header().Set("Allow","POST");http.Error(w,"Method not allowed",405);return false}
 if r.Header.Get("X-Ghost-Demo")!="simulation-only" {http.Error(w,"Missing local demo action header",403);return false}
 if o:=r.Header.Get("Origin");o!=""&&o!="http://"+host {http.Error(w,"Cross-origin demo action denied",403);return false}
 if r.Header.Get("Content-Type")!="application/json" {http.Error(w,"JSON required",415);return false}
 return true
}
func decodeAction(w http.ResponseWriter,r *http.Request,d any) bool {
 dec:=json.NewDecoder(http.MaxBytesReader(w,r.Body,2048));dec.DisallowUnknownFields()
 if err:=dec.Decode(d);err!=nil {http.Error(w,"Invalid demo action",400);return false}
 var extra any
 if err:=dec.Decode(&extra);err!=io.EOF {http.Error(w,"Extra JSON values rejected",400);return false}
 return true
}
func selfTest() error {
 d:=&demo{start:time.Now()}
 h,err:=d.handler("127.0.0.1:28291");if err!=nil{return err}
 req:=httptest.NewRequest("GET","http://127.0.0.1:28291/api/v1/state",nil)
 w:=httptest.NewRecorder();h.ServeHTTP(w,req)
 if w.Code!=200||!strings.Contains(w.Body.String(),`"demo":true`)||!strings.Contains(w.Body.String(),"Brand-Guidelines.pdf"){return fmt.Errorf("offline demo fixture failure: %d",w.Code)}
 req=httptest.NewRequest("GET","http://127.0.0.1:28291/app/index.html",nil)
 w=httptest.NewRecorder();h.ServeHTTP(w,req)
 if w.Code!=200||!strings.Contains(w.Body.String(),"Ghost FTP Demo"){return fmt.Errorf("offline demo UI failure: %d",w.Code)}
 req=httptest.NewRequest("GET","http://malicious.test/api/v1/state",nil)
 req.Host="malicious.test";w=httptest.NewRecorder();h.ServeHTTP(w,req)
 if w.Code!=403{return fmt.Errorf("host-boundary failure: %d",w.Code)}
 return nil
}
func main(){
 showVersion:=flag.Bool("version",false,"Print canonical product version")
 check:=flag.Bool("self-test",false,"Verify embedded UI and offline simulation without launching a browser")
 flag.Parse()
 if *showVersion {fmt.Println("Ghost FTP Demo "+version+" Preview");return}
 if *check {if err:=selfTest();err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)};fmt.Println("Ghost FTP Demo offline/self-test PASS");return}
 ln,err:=net.Listen("tcp","127.0.0.1:0")
 if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}
 defer ln.Close()
 host:=ln.Addr().String(); d:=&demo{start:time.Now()}
 h,err:=d.handler(host)
 if err!=nil{fmt.Fprintln(os.Stderr,err);os.Exit(1)}
 srv:=&http.Server{Handler:h,ReadHeaderTimeout:5*time.Second,ReadTimeout:15*time.Second,WriteTimeout:15*time.Second,IdleTimeout:30*time.Second}
 url:="http://"+host+"/app/"
 fmt.Println("Ghost FTP Demo — SIMULATION ONLY; all files/accounts and transfers are fictional.")
 fmt.Println("Open "+url)
 if runtime.GOOS=="windows"&&os.Getenv("GHOSTFTP_DEMO_HEADLESS")!="1" { _=exec.Command("rundll32","url.dll,FileProtocolHandler",url).Start() }
 ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer stop()
 go func(){<-ctx.Done();c,cancel:=context.WithTimeout(context.Background(),5*time.Second);defer cancel();_=srv.Shutdown(c)}()
 if err:=srv.Serve(ln);err!=nil&&err!=http.ErrServerClosed {fmt.Fprintln(os.Stderr,err);os.Exit(1)}
}
