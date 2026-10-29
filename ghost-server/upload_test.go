package ghostserver

import (
 "context"
 "crypto/tls"
 "io"
 "net"
 "os"
 "path/filepath"
 "strconv"
 "strings"
 "testing"
 "time"
)

func writableServer(t *testing.T, maxBytes int64) (*Server,string,func()) {
 t.Helper()
 root:=t.TempDir()
 crt,key:=certFiles(t)
 srv,err:=New(Config{
  Root:root,Username:"qa",Password:"long-private-test-password",
  Certificate:crt,PrivateKey:key,AllowUploads:true,MaxUploadBytes:maxBytes,
 })
 if err!=nil{t.Fatal(err)}
 ln,err:=net.Listen("tcp","127.0.0.1:0")
 if err!=nil{t.Fatal(err)}
 ctx,cancel:=context.WithCancel(context.Background())
 done:=make(chan error,1)
 go func(){done<-srv.Serve(ctx,ln)}()
 return srv,ln.Addr().String(),func(){
  cancel()
  select{case <-done:case <-time.After(3*time.Second):t.Error("server did not shut down")}
 }
}

func passivePort(t *testing.T,c *ftpc) int {
 t.Helper()
 reply:=c.command(t,"229 ","EPSV")
 start:=strings.Index(reply,"(|||")
 if start<0{t.Fatalf("bad passive reply: %q",reply)}
 tail:=reply[start+4:]
 end:=strings.IndexByte(tail,'|')
 if end<0{t.Fatalf("invalid passive delimiter %q",reply)}
 port,err:=strconv.Atoi(tail[:end])
 if err!=nil||port<1||port>65535{t.Fatalf("bad passive port: %q (%v)",reply,err)}
 return port
}

func protectedUpload(t *testing.T,c *ftpc,addr,name string,data []byte,expectedReply string) {
 t.Helper()
 port:=passivePort(t,c)
 control,err:=net.ResolveTCPAddr("tcp",addr)
 if err!=nil{t.Fatal(err)}
 raw,err:=net.DialTimeout("tcp",net.JoinHostPort(control.IP.String(),strconv.Itoa(port)),3*time.Second)
 if err!=nil{t.Fatal(err)}
 defer raw.Close()
 c.command(t,"150 ","STOR "+name)
 conn:=tls.Client(raw,&tls.Config{InsecureSkipVerify:true,MinVersion:tls.VersionTLS12}) // test fixture only
 if err:=conn.Handshake();err!=nil{t.Fatal(err)}
 if _,err:=conn.Write(data);err!=nil{t.Fatal(err)}
 if err:=conn.Close();err!=nil{t.Fatal(err)}
 c.read(t,expectedReply)
}

func TestCreateOnlyUploadPreservesExistingFileBytes(t *testing.T) {
 srv,addr,stop:=writableServer(t,1<<20)
 defer stop()
 original:=filepath.Join(srv.root,"existing.bin")
 if err:=os.WriteFile(original,[]byte("original-do-not-touch"),0600);err!=nil{t.Fatal(err)}
 c:=connect(t,addr)
 defer c.conn.Close()
 login(t,c)
 // A conflict must fail before TLS data is opened and leave bytes unchanged.
 _=passivePort(t,c)
 c.command(t,"550 ","STOR existing.bin")
 after,err:=os.ReadFile(original)
 if err!=nil||string(after)!="original-do-not-touch"{t.Fatalf("existing file changed %q: %v",after,err)}
 protectedUpload(t,c,addr,"new.bin",[]byte("secure-created-file"),"226 ")
 data,err:=os.ReadFile(filepath.Join(srv.root,"new.bin"))
 if err!=nil||string(data)!="secure-created-file"{t.Fatalf("upload mismatch %q: %v",data,err)}
 if info,err:=os.Lstat(filepath.Join(srv.root,"new.bin"));err!=nil||!info.Mode().IsRegular(){t.Fatalf("invalid output mode: %v %v",info,err)}
 _=passivePort(t,c)
 c.command(t,"550 ","STOR new.bin")
 after,err=os.ReadFile(filepath.Join(srv.root,"new.bin"))
 if err!=nil||string(after)!="secure-created-file"{t.Fatalf("second STOR overwrote file: %q %v",after,err)}
 leftovers,err:=filepath.Glob(filepath.Join(srv.root,".ghostftp-upload-*"))
 if err!=nil||len(leftovers)!=0{t.Fatalf("staged upload leaked: %v, %v",leftovers,err)}
}

func TestUploadQuotaRefusesOversizedPayloadAndCleansStaging(t *testing.T) {
 srv,addr,stop:=writableServer(t,8)
 defer stop()
 c:=connect(t,addr)
 defer c.conn.Close()
 login(t,c)
 protectedUpload(t,c,addr,"too-big.txt",[]byte("123456789"),"552 ")
 if _,err:=os.Lstat(filepath.Join(srv.root,"too-big.txt"));!os.IsNotExist(err){t.Fatalf("oversized file committed: %v",err)}
 leftovers,_:=filepath.Glob(filepath.Join(srv.root,".ghostftp-upload-*"))
 if len(leftovers)!=0{t.Fatalf("oversized stage leaked: %v",leftovers)}
}

func TestUploadRejectsUnsafeParentsAndDoesNotEnableDestructiveCommands(t *testing.T) {
 srv,addr,stop:=writableServer(t,1024)
 defer stop()
 outside:=t.TempDir()
 if err:=os.Symlink(outside,filepath.Join(srv.root,"outside"));err!=nil{t.Skipf("symlink unsupported: %v",err)}
 for _,name:=range []string{"../escape",".","child/../escape","outside/file","existing/../../x","/../x","name/",".ghostftp-upload-other",".GHOSTFTP-UPLOAD-case-alias"} {
  if _,err:=srv.createDestination("/",name);err==nil{t.Errorf("accepted unsafe upload %q",name)}
 }
 c:=connect(t,addr);defer c.conn.Close()
 login(t,c)
 for _,cmd:=range []string{"DELE test.txt","RNFR test.txt","RNTO renamed.txt","MKD folder","APPE test.txt","PORT 1,1,1,1,1,1"} {
  c.command(t,"550 ",cmd)
 }
}

func TestListDashAUsesRealInventoryWithoutLosingHiddenEntries(t *testing.T) {
 srv,addr,stop:=testServer(t)
 defer stop()
 if err:=os.WriteFile(filepath.Join(srv.root,".env"),[]byte("hidden"),0600);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(srv.root,".ghostftp-upload-private-stage"),[]byte("incomplete-secret-data"),0600);err!=nil{t.Fatal(err)}
 if _,_,err:=srv.resolve("/",".ghostftp-upload-private-stage");err==nil{t.Fatal("private staging data exposed by FTP read path")}
 c:=connect(t,addr);defer c.conn.Close()
 login(t,c)
 port:=passivePort(t,c)
 remote,err:=net.ResolveTCPAddr("tcp",addr);if err!=nil{t.Fatal(err)}
 raw,err:=net.DialTimeout("tcp",net.JoinHostPort(remote.IP.String(),strconv.Itoa(port)),3*time.Second)
 if err!=nil{t.Fatal(err)}
 defer raw.Close()
 c.command(t,"150 ","LIST -a")
 secure:=tls.Client(raw,&tls.Config{InsecureSkipVerify:true,MinVersion:tls.VersionTLS12}) // local fixture only
 if err:=secure.Handshake();err!=nil{t.Fatal(err)}
 listing,err:=io.ReadAll(secure)
 if err!=nil{t.Fatal(err)}
 c.read(t,"226 ")
 if !strings.Contains(string(listing),".env")||!strings.Contains(string(listing),"test.txt"){t.Fatalf("LIST -a lost real entries: %s",listing)}
 if strings.Contains(string(listing),".ghostftp-upload-") || strings.Contains(string(listing),"incomplete-secret-data"){
  t.Fatalf("in-flight staged upload exposed in FTP directory listing: %s",listing)
 }
}

