//go:build linux

package ghostserver

import (
 "context"
 "encoding/pem"
 "net"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "testing"
 "time"
)

// An independent real FTP/FTPS client verifies interoperability rather than
// only testing the server with its own test-only FTP wire helper.
func TestCurlIndependentFTPSClientDownloadAndCreateOnlyUpload(t *testing.T) {
 if _,err:=exec.LookPath("curl");err!=nil{t.Skip("curl not installed")}
 srv,addr,stop:=writableServer(t,1024*1024)
 defer stop()
 if err:=os.WriteFile(filepath.Join(srv.root,"existing.txt"),[]byte("baseline-existing-file"),0600);err!=nil{t.Fatal(err)}
 certFile:=filepath.Join(t.TempDir(),"trusted-local-test-cert.pem")
 leaf:=srv.tls.Certificates[0].Certificate[0]
 if err:=os.WriteFile(certFile,pem.EncodeToMemory(&pem.Block{Type:"CERTIFICATE",Bytes:leaf}),0600);err!=nil{t.Fatal(err)}
 _,port,err:=net.SplitHostPort(addr)
 if err!=nil{t.Fatal(err)}
 url:="ftp://localhost:"+port+"/"
 run:=func(args ...string)(string,error){
  t.Helper()
  ctx,cancel:=context.WithTimeout(context.Background(),15*time.Second)
  defer cancel()
  base:=[]string{"--silent","--show-error","--fail","--ipv4","--noproxy","*",
   "--ssl-reqd","--cacert",certFile,"--user","qa:long-private-test-password",
   "--connect-timeout","3","--max-time","12"}
  cmd:=exec.CommandContext(ctx,"curl",append(base,args...)...)
  output,err:=cmd.CombinedOutput()
  if ctx.Err()!=nil{t.Fatalf("curl timed out: %v",ctx.Err())}
  return string(output),err
 }
 data,err:=run(url+"existing.txt")
 if err!=nil||data!="baseline-existing-file"{t.Fatalf("curl real FTPS download failed: %v, %q",err,data)}
 source:=filepath.Join(t.TempDir(),"source.txt")
 if err:=os.WriteFile(source,[]byte("verified-new-upload"),0600);err!=nil{t.Fatal(err)}
 out,err:=run("--quote","ALLO 19","--upload-file",source,url+"uploaded.txt")
 if err!=nil{t.Fatalf("curl real protected FTPS STOR failed: %v, %q",err,out)}
 saved,err:=os.ReadFile(filepath.Join(srv.root,"uploaded.txt"))
 if err!=nil||string(saved)!="verified-new-upload"{t.Fatalf("curl upload did not preserve bytes: %v %q",err,saved)}
 out,err=run("--quote","ALLO 19","--upload-file",source,url+"existing.txt")
 if err==nil{t.Fatalf("curl unexpectedly overwrote existing destination: %q",out)}
 bytes,err:=os.ReadFile(filepath.Join(srv.root,"existing.txt"))
 if err!=nil||string(bytes)!="baseline-existing-file"{t.Fatalf("curl conflict changed original bytes: %v %q",err,bytes)}
 if strings.Contains(string(out),"verified-new-upload") {t.Fatal("server response contained uploaded secret bytes")}
}
