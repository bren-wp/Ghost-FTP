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
func main(){
 listen:=flag.String("listen","127.0.0.1:2121","FTP control listener (loopback by default)")
 root:=flag.String("root","","Existing directory to serve read-only")
 cert:=flag.String("cert","","TLS server certificate PEM")
 key:=flag.String("key","","TLS private key PEM")
 username:=flag.String("user","","FTP username")
 passfile:=flag.String("password-file","","Private file holding FTP password, never a CLI argument")
 flag.Parse()
 if *passfile==""{log.Fatal("password-file is required")}
 info,err:=os.Lstat(*passfile);if err!=nil{log.Fatal(err)}
 if !info.Mode().IsRegular()||info.Mode()&os.ModeSymlink!=0{log.Fatal("password file must be regular, never symlink")}
 if runtime.GOOS!="windows"&&info.Mode().Perm()&0077!=0{log.Fatal("password file must be owner-only (chmod 600)")}
 raw,err:=os.ReadFile(*passfile);if err!=nil{log.Fatal(err)}
 if len(raw)>257{log.Fatal("password file too large")}
 secret:=strings.TrimSuffix(string(raw),"\n");secret=strings.TrimSuffix(secret,"\r")
 cfg:=server.Config{Listen:*listen,Root:*root,Username:*username,Password:secret,Certificate:*cert,PrivateKey:*key}
 srv,err:=server.New(cfg);if err!=nil{log.Fatal(err)}
 for i:=range raw{raw[i]=0}
 ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer stop()
 fmt.Printf("Ghost FTP Server Preview: explicit FTPS, read-only at %s\n",*listen)
 if err:=srv.ListenAndServe(ctx,*listen);err!=nil{log.Fatal(err)}
}
