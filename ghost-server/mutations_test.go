package ghostserver

import (
 "context"
 "net"
 "os"
 "path/filepath"
 "testing"
 "time"
)

func enabledMutationsServer(t *testing.T)(*Server,string,func()) {
 t.Helper()
 root:=t.TempDir()
 cert,key:=certFiles(t)
 srv,err:=New(Config{
  Root:root,Username:"qa",Password:"long-private-test-password",
  Certificate:cert,PrivateKey:key,
  AllowUploads:true,AllowDestructive:true,MaxUploadBytes:1024,
 })
 if err!=nil{t.Fatal(err)}
 ln,err:=net.Listen("tcp","127.0.0.1:0")
 if err!=nil{t.Fatal(err)}
 ctx,cancel:=context.WithCancel(context.Background())
 done:=make(chan error,1)
 go func(){done<-srv.Serve(ctx,ln)}()
 return srv,ln.Addr().String(),func(){
  cancel()
  select{case <-done:case <-time.After(4*time.Second):t.Error("mutation server shutdown timed out")}
 }
}

func TestMutationsRequireSeparateExplicitOptIn(t *testing.T){
 root:=t.TempDir();crt,key:=certFiles(t)
 if _,err:=New(Config{
  Root:root,Username:"qa",Password:"long-private-test-password",
  Certificate:crt,PrivateKey:key,AllowDestructive:true,
 });err==nil{t.Fatal("destructive mutations enabled without creation privileges")}
}

func TestMutationsRefuseOverwriteAndPreserveBytes(t *testing.T){
 srv,addr,stop:=enabledMutationsServer(t)
 defer stop()
 original:=filepath.Join(srv.root,"from.txt")
 occupied:=filepath.Join(srv.root,"existing.txt")
 if err:=os.WriteFile(original,[]byte("original-byte-stream"),0600);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(occupied,[]byte("occupied-unchanged"),0600);err!=nil{t.Fatal(err)}
 c:=connect(t,addr)
 defer c.conn.Close()
 login(t,c)
 c.command(t,"503 ","RNTO no-source.txt")
 c.command(t,"350 ","RNFR from.txt")
 c.command(t,"550 ","RNTO existing.txt")
 for _,pair:=range [][2]string{{"from.txt","original-byte-stream"},{"existing.txt","occupied-unchanged"}}{
  b,err:=os.ReadFile(filepath.Join(srv.root,pair[0]))
  if err!=nil||string(b)!=pair[1]{t.Fatalf("rename conflict damaged %s: %v %q",pair[0],err,b)}
 }
 c.command(t,"350 ","RNFR from.txt")
 c.command(t,"250 ","RNTO renamed.txt")
 if _,err:=os.Lstat(original);!os.IsNotExist(err){t.Fatalf("source still exists after rename: %v",err)}
 b,err:=os.ReadFile(filepath.Join(srv.root,"renamed.txt"))
 if err!=nil||string(b)!="original-byte-stream"{t.Fatalf("rename byte integrity failed: %v %q",err,b)}
 c.command(t,"250 ","DELE renamed.txt")
 if _,err:=os.Lstat(filepath.Join(srv.root,"renamed.txt"));!os.IsNotExist(err){t.Fatalf("file still exists after DELE: %v",err)}
 c.command(t,"550 ","DELE existing-does-not-exist")
 c.command(t,"550 ","DELE .ghostftp-upload-private-staging")
 b,err=os.ReadFile(occupied)
 if err!=nil||string(b)!="occupied-unchanged"{t.Fatalf("other data unexpectedly changed: %v %q",err,b)}
}

func TestRemoveOnlyEmptyDirectoryAndProtectRoot(t *testing.T){
 srv,addr,stop:=enabledMutationsServer(t)
 defer stop()
 c:=connect(t,addr);defer c.conn.Close()
 login(t,c)
 c.command(t,"257 ","MKD empty")
 c.command(t,"250 ","RMD empty")
 if _,err:=os.Lstat(filepath.Join(srv.root,"empty"));!os.IsNotExist(err){t.Fatalf("empty directory still present: %v",err)}
 if err:=os.Mkdir(filepath.Join(srv.root,"nonempty"),0700);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(srv.root,"nonempty","save"),[]byte("never-delete"),0600);err!=nil{t.Fatal(err)}
 c.command(t,"550 ","RMD nonempty")
 c.command(t,"550 ","RMD /")
 c.command(t,"550 ","DELE ../outside")
 b,err:=os.ReadFile(filepath.Join(srv.root,"nonempty","save"))
 if err!=nil||string(b)!="never-delete"{t.Fatalf("RMD deleted nonempty content: %v %q",err,b)}
}
