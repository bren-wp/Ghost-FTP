package main

import (
 "context"
 "flag"
 "fmt"
 "log"
 "os"
 "os/signal"
 "runtime"
 "strings"
 "syscall"
 server "ghostftp/server"
)
var version = "dev" // set only by audited CI -ldflags; never changes protocol policy

func main(){
 // A double-click without configuration must no longer immediately close
 // after a cryptic missing-secret error. On Windows, launch the real native
 // (non-WebView) Control Center. Linux remains an explicit daemon/CLI service.
 if len(os.Args)==1 {
  if runNativeControlCenter(){return}
  fmt.Println("Ghost FTP Server requires a configured TLS identity and password file.")
  fmt.Println("Use -help for secure daemon options; on Linux use systemctl for service management.")
  return
 }
 controlCenter:=flag.Bool("control-center",false,"Open native local Windows Control Center (no browser/WebView)")
 listen:=flag.String("listen","127.0.0.1:2121","FTP control listener (loopback by default)")
 root:=flag.String("root","","Existing directory to serve read-only")
 cert:=flag.String("cert","","TLS server certificate PEM")
 key:=flag.String("key","","TLS private key PEM")
 username:=flag.String("user","","FTP username")
 passfile:=flag.String("password-file","","Private file holding FTP password, never a CLI argument")
 serviceMode:=flag.Bool("service",false,"Run under Windows Service Control Manager (Windows only)")
 adminTokenFile:=flag.String("admin-token-file","","Private file holding a separate random admin dashboard token; omitted disables admin HTTP")
 adminListen:=flag.String("admin-listen","127.0.0.1:2821","Local-only admin HTTP address when enabled")
 allowUploads:=flag.Bool("allow-uploads",false,"Opt-in creation of new remote files (never overwrite); requires a sandboxed service account")
 allowDestructive:=flag.Bool("allow-destructive",false,"Additional opt-in to DELE, empty RMD and no-overwrite file rename; requires allow-uploads and isolated OS identity")
 maxUpload:=flag.Int64("max-upload-bytes",64<<20,"Per-transfer upload limit when uploads are explicitly enabled")
 showVersion:=flag.Bool("version",false,"Print the compiled Ghost FTP Server version and exit")
 flag.Parse()
 if *showVersion { fmt.Println("Ghost FTP Server "+version+" Preview"); return }
 if *controlCenter {
  if !runNativeControlCenter(){log.Fatal("native Control Center unavailable; on Linux manage Ghost FTP Server with systemd or CLI")}
  return
 }
 if *passfile==""{log.Fatal("password-file is required")}
 info,err:=os.Lstat(*passfile);if err!=nil{log.Fatal(err)}
 if !info.Mode().IsRegular()||info.Mode()&os.ModeSymlink!=0{log.Fatal("password file must be regular, never symlink")}
 if runtime.GOOS!="windows"&&info.Mode().Perm()&0077!=0{log.Fatal("password file must be owner-only (chmod 600)")}
 raw,err:=os.ReadFile(*passfile);if err!=nil{log.Fatal(err)}
 if len(raw)>257{log.Fatal("password file too large")}
 secret:=strings.TrimSuffix(string(raw),"\n");secret=strings.TrimSuffix(secret,"\r")
 adminToken:=""
 if *adminTokenFile!="" {
  tokenInfo,e:=os.Lstat(*adminTokenFile);if e!=nil{log.Fatal(e)}
  if !tokenInfo.Mode().IsRegular()||tokenInfo.Mode()&os.ModeSymlink!=0{log.Fatal("admin token file must be regular and not symlinked")}
  if runtime.GOOS!="windows"&&tokenInfo.Mode().Perm()&0077!=0{log.Fatal("admin token file must be owner-only (chmod 600)")}
  tokenBytes,e:=os.ReadFile(*adminTokenFile);if e!=nil{log.Fatal(e)}
  if len(tokenBytes)>257{log.Fatal("admin token file too large")}
  adminToken=strings.TrimSuffix(strings.TrimSuffix(string(tokenBytes),"\n"),"\r")
  for i:=range tokenBytes{tokenBytes[i]=0}
 }
 cfg:=server.Config{Listen:*listen,Root:*root,Username:*username,Password:secret,Certificate:*cert,PrivateKey:*key,AllowUploads:*allowUploads,AllowDestructive:*allowDestructive,MaxUploadBytes:*maxUpload,AdminToken:adminToken,AdminListen:*adminListen}
 srv,err:=server.New(cfg);if err!=nil{log.Fatal(err)}
 for i:=range raw{raw[i]=0}
 if *serviceMode {
  if err:=runAsWindowsService(srv,*listen);err!=nil{log.Fatal(err)}
  return
 }
 ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer stop()
 if *allowUploads {fmt.Printf("Ghost FTP Server Preview: explicit FTPS, restricted create-only uploads at %s\n",*listen)} else {fmt.Printf("Ghost FTP Server Preview: explicit FTPS, read-only at %s\n",*listen)}
 if err:=srv.ListenAndServe(ctx,*listen);err!=nil{log.Fatal(err)}
}
