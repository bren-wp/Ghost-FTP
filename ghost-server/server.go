// Package ghostserver implements a deliberately read-only explicit FTPS
// server. Unsupported mutations fail closed until an auditable write pipeline
// (atomic no-overwrite, quota, ACL and crash recovery) is implemented.
package ghostserver

import (
 "bufio"
 "context"
 "crypto/sha256"
 "crypto/subtle"
 "crypto/tls"
 "errors"
 "fmt"
 "io"
 "net"
 "net/http"
 "os"
 "path"
 "path/filepath"
 "strconv"
 "strings"
 "sync"
 "sync/atomic"
 "time"
)

type Config struct {
 Listen, Root, Username, Password, Certificate, PrivateKey string
 // Writes are opt-in. The default server remains read-only.
 AllowUploads bool
 // Destructive mutations require a separate explicit opt-in as well as
 // create-only upload permission. The default remains deny-all writes.
 AllowDestructive bool
 MaxUploadBytes int64
 // Admin is disabled unless a strong, separately provisioned token exists.
 AdminToken, AdminListen string
}
type Server struct {
 root, user string
 password [32]byte
 tls *tls.Config
 limit chan struct{}
 allowUploads bool
 allowDestructive bool
 maxUploadBytes int64
 // os.Root confines opt-in mutations within an open directory handle,
 // including against symlink swaps on Linux and Windows.
 uploadRoot *os.Root
 adminEnabled bool
 adminListen string
 adminTokenHash [32]byte
 startedAt time.Time
 activeSessions atomic.Int64
 // Authentication throttling follows the peer IP across short-lived
 // reconnects, rather than resetting after every three failed logins.
 authMu sync.Mutex
 authFailures map[string]authFailures
}
func New(c Config) (*Server,error) {
 if c.Username=="" || len(c.Username)>128 || strings.ContainsAny(c.Username,"\r\n\x00") { return nil,errors.New("valid username required") }
 if len(c.Password)<12 || len(c.Password)>256 {return nil,errors.New("password must be 12-256 bytes")}
 if c.Root=="" {return nil,errors.New("filesystem root required")}
 root,err:=filepath.Abs(c.Root);if err!=nil{return nil,err}
 info,err:=os.Lstat(root);if err!=nil{return nil,err}
 if !info.IsDir() || info.Mode()&os.ModeSymlink!=0{return nil,errors.New("root must be a real directory")}
 root,err=filepath.EvalSymlinks(root);if err!=nil{return nil,err}
 if c.AllowDestructive && !c.AllowUploads {
  return nil,errors.New("destructive operations require explicitly enabled uploads")
 }
 if c.AllowUploads {
  if c.MaxUploadBytes == 0 { c.MaxUploadBytes = 64 << 20 }
  if c.MaxUploadBytes < 1 || c.MaxUploadBytes > 1 << 30 { return nil,errors.New("max upload must be 1 to 1073741824 bytes") }
 }
 if c.AdminToken!="" {
  if len(c.AdminToken)<32||len(c.AdminToken)>256{return nil,errors.New("admin token must contain 32-256 bytes")}
  if c.AdminListen==""{c.AdminListen="127.0.0.1:2821"}
  host,_,err:=net.SplitHostPort(c.AdminListen)
  if err!=nil || net.ParseIP(host)==nil || !net.ParseIP(host).IsLoopback(){
   return nil,errors.New("admin UI must bind to an explicit loopback IP")
  }
 }
 if c.Certificate==""||c.PrivateKey=="" {return nil,errors.New("certificate and private key required; plaintext FTP disabled")}
 cert,err:=tls.LoadX509KeyPair(c.Certificate,c.PrivateKey);if err!=nil{return nil,fmt.Errorf("invalid TLS identity: %w",err)}
 // The same race-resistant handle confines both reads and optional
 // create-only writes, even when local directory entries are swapped.
 uploadRoot,err:=os.OpenRoot(root)
 if err!=nil{return nil,fmt.Errorf("cannot open traversal-resistant server root: %w",err)}
 return &Server{root:root,user:c.Username,password:sha256.Sum256([]byte(c.Password)),tls:&tls.Config{Certificates:[]tls.Certificate{cert},MinVersion:tls.VersionTLS12},limit:make(chan struct{},32),allowUploads:c.AllowUploads,allowDestructive:c.AllowDestructive,maxUploadBytes:c.MaxUploadBytes,uploadRoot:uploadRoot,
 adminEnabled:c.AdminToken!="",adminListen:c.AdminListen,adminTokenHash:sha256.Sum256([]byte(c.AdminToken)),startedAt:time.Now(),authFailures:make(map[string]authFailures)},nil
}
// Close releases filesystem handles when the server is configured but never
// started. Serve also closes these handles after stopping. Callers must not
// use the server after Close.
func (s *Server) Close() error {
 if s.uploadRoot!=nil{return s.uploadRoot.Close()}
 return nil
}
func (s *Server) ListenAndServe(ctx context.Context, address string) error {
 if address=="" {address="127.0.0.1:2121"}
 ln,err:=net.Listen("tcp",address);if err!=nil{return err}
 return s.Serve(ctx,ln)
}
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
 defer ln.Close()
 defer s.Close()
 if s.adminEnabled {
  adminLn,err:=net.Listen("tcp",s.adminListen)
  if err!=nil{return fmt.Errorf("cannot bind local admin dashboard: %w",err)}
  adminHTTP:=&http.Server{Handler:s.adminHandler(),ReadHeaderTimeout:3*time.Second,IdleTimeout:30*time.Second,MaxHeaderBytes:8192}
  go func(){_ = adminHTTP.Serve(adminLn)}()
  defer adminHTTP.Close()
 }
 go func(){<-ctx.Done();_ = ln.Close()}()
 var wg sync.WaitGroup
 defer wg.Wait()
 for {
  conn,err:=ln.Accept()
  if err!=nil {if ctx.Err()!=nil{return nil};return err}
  select {
  case s.limit<-struct{}{}:
   s.activeSessions.Add(1)
   wg.Add(1);go func(){defer wg.Done();defer s.activeSessions.Add(-1);defer func(){<-s.limit}();s.session(ctx,conn)}()
  default:
   _,_ = io.WriteString(conn,"421 Too many connections\r\n");_ = conn.Close()
  }
 }
}
type client struct {
 conn net.Conn
 r *bufio.Reader
 w *bufio.Writer
 secure, logged, pbsz, private bool
 failures int
 user, cwd string
 renameSource string
 passive net.Listener
}
func (c *client) reply(code int, msg string) error {
 _,err:=fmt.Fprintf(c.w,"%d %s\r\n",code,msg)
 if err!=nil{return err};return c.w.Flush()
}
func (c *client) closePassive(){if c.passive!=nil{_ = c.passive.Close();c.passive=nil}}
func (s *Server) session(ctx context.Context,conn net.Conn){
 ended:=make(chan struct{})
 go func(){select{case <-ctx.Done():_ = conn.Close();case <-ended:}}()
 defer close(ended)
 c:=&client{conn:conn,r:bufio.NewReader(conn),w:bufio.NewWriter(conn),cwd:"/"}
 defer func(){c.closePassive();_ = c.conn.Close()}()
 if c.reply(220,"Ghost FTP Server - explicit TLS required")!=nil{return}
 for {
  _ = c.conn.SetDeadline(time.Now().Add(90*time.Second))
  // ReadSlice has a fixed-size buffer. ReadString can allocate without
  // bound when an unauthenticated peer never sends CRLF.
  chunk,err:=c.r.ReadSlice('\n');if err!=nil||len(chunk)>4096{return}
  line:=string(chunk)
  line=strings.TrimSuffix(strings.TrimSuffix(line,"\n"),"\r")
  verb,arg,_:=strings.Cut(line," ")
  verb=strings.ToUpper(verb)
  if verb=="" || strings.ContainsAny(arg,"\r\n\x00"){_ = c.reply(501,"Invalid argument");continue}
  if verb=="QUIT"{_ = c.reply(221,"Goodbye");return}
  if verb=="NOOP"{_ = c.reply(200,"OK");continue}
  if verb=="FEAT"{_,_ = io.WriteString(c.w,"211-Extensions\r\n AUTH TLS\r\n PBSZ\r\n PROT\r\n EPSV\r\n PASV\r\n MLSD\r\n SIZE\r\n UTF8\r\n211 End\r\n");_ = c.w.Flush();continue}
  if verb=="AUTH" {
   if c.secure || strings.ToUpper(strings.TrimSpace(arg))!="TLS"{_ = c.reply(534,"Only one explicit AUTH TLS upgrade is permitted");continue}
   if c.reply(234,"Negotiating TLS")!=nil{return}
   sec:=tls.Server(c.conn,s.tls)
   if sec.Handshake()!=nil{return}
   c.conn=sec;c.r=bufio.NewReader(sec);c.w=bufio.NewWriter(sec);c.secure=true
   continue
  }
  if !c.secure{_ = c.reply(534,"TLS required before authentication");continue}
  if verb=="USER"{c.logged=false;c.pbsz=false;c.private=false;c.renameSource="";c.user=arg;_ = c.reply(331,"Password required");continue}
  if verb=="PASS" {
   digest:=sha256.Sum256([]byte(arg))
   validPassword:=subtle.ConstantTimeCompare(digest[:],s.password[:])==1
   authorized:=c.user==s.user&&validPassword
   // Global anti-bruteforce survives socket reconnects. Do not distinguish
   // non-existent users from invalid passwords in any response.
   allowed:=s.recordAuthAttempt(c.conn.RemoteAddr(),authorized)
   if !allowed{c.logged=false;_=c.reply(421,"Login temporarily restricted");return}
   if authorized{c.logged=true;c.failures=0;_ = c.reply(230,"Authenticated")}else{
    c.logged=false;c.failures++
    if c.failures>=3{_ = c.reply(421,"Too many failed logins");return}
    _ = c.reply(530,"Authentication failed")
   }
   continue
  }
  if !c.logged{_ = c.reply(530,"Login required");continue}
  switch verb {
  case "PBSZ":
   if strings.TrimSpace(arg)!="0"{_ = c.reply(501,"PBSZ must be 0")}else{c.pbsz=true;_ = c.reply(200,"PBSZ 0")}
  case "PROT":
   if !c.pbsz||strings.ToUpper(strings.TrimSpace(arg))!="P"{_ = c.reply(534,"Private data TLS required")}else{c.private=true;_ = c.reply(200,"PROT P")}
  case "SYST": _ = c.reply(215,"UNIX Type: L8")
  case "OPTS":
   if strings.EqualFold(strings.TrimSpace(arg),"UTF8 ON"){_ = c.reply(200,"UTF8 enabled")}else{_ = c.reply(501,"Unsupported option")}
  case "TYPE":
   if strings.EqualFold(strings.TrimSpace(arg),"I"){_ = c.reply(200,"Binary type")}else{_ = c.reply(504,"Binary transfers only")}
  case "PWD": _ = c.reply(257,fmt.Sprintf("%q is current directory",c.cwd))
  case "CWD","CDUP":
   dest:=arg;if verb=="CDUP"{dest=path.Dir(c.cwd)}
   _,virtual,e:=s.resolve(c.cwd,dest)
   if e==nil{
    var handle *os.Root
    handle,e=s.uploadRoot.OpenRoot(rootedPath(virtual))
    if e==nil{_ = handle.Close()}
   }
   if e!=nil{_ = c.reply(550,"Directory unavailable")}else{c.cwd=virtual;_ = c.reply(250,"Directory changed")}
  case "SIZE":
   _,virtual,e:=s.resolve(c.cwd,arg)
   if e==nil{
    file,openErr:=s.uploadRoot.Open(rootedPath(virtual))
    if openErr==nil{
     info,statErr:=file.Stat()
     _ = file.Close()
     if statErr==nil&&info.Mode().IsRegular(){_ = c.reply(213,strconv.FormatInt(info.Size(),10));continue}
    }
   }
   _ = c.reply(550,"File unavailable")
  case "EPSV","PASV":
   if !c.private{_ = c.reply(534,"PROT P required");continue}
   c.closePassive()
   host:=c.conn.LocalAddr().(*net.TCPAddr).IP
   if host.IsUnspecified(){_ = c.reply(425,"Invalid local network address");continue}
   ln,e:=net.ListenTCP("tcp",&net.TCPAddr{IP:host,Port:0})
   if e!=nil{_ = c.reply(425,"Cannot create data listener");continue}
   c.passive=ln
   port:=ln.Addr().(*net.TCPAddr).Port
   if verb=="EPSV" {
    _ = c.reply(229,fmt.Sprintf("Entering Extended Passive Mode (|||%d|)",port))
   } else if ip:=host.To4();ip!=nil {
    _ = c.reply(227,fmt.Sprintf("Entering Passive Mode (%d,%d,%d,%d,%d,%d)",ip[0],ip[1],ip[2],ip[3],port/256,port%256))
   } else {
    c.closePassive();_ = c.reply(522,"Use EPSV for IPv6")
   }
  case "LIST","MLSD","NLST":
   target:=strings.TrimSpace(arg)
   if verb=="LIST" {
    // Older FTP clients use LIST -a when MLSD is unavailable. Treat -a
    // as a listing option, NOT a filesystem path (hidden entries included).
    switch {
    case target=="-a" || target=="-al" || target=="-la" || target=="-l":
     target="."
    case strings.HasPrefix(target,"-a ") || strings.HasPrefix(target,"-al ") || strings.HasPrefix(target,"-la ") || strings.HasPrefix(target,"-l "):
     _,target,_=strings.Cut(target," ")
     target=strings.TrimSpace(target)
    case strings.HasPrefix(target,"-"):
     _ = c.reply(501,"Unsupported LIST option");continue
    }
   }
   if target==""{target="."}
   _,virtual,e:=s.resolve(c.cwd,target)
   if e!=nil{_ = c.reply(550,"Directory unavailable");continue}
   directory,e:=s.uploadRoot.Open(rootedPath(virtual))
   if e!=nil{_ = c.reply(550,"Directory unavailable");continue}
   ents,e:=directory.ReadDir(10001)
   _ = directory.Close()
   if e!=nil&&e!=io.EOF{_ = c.reply(550,"Directory unavailable");continue}
   if len(ents)>10000{_ = c.reply(552,"Directory exceeds Preview entry limit");continue}
   var out strings.Builder
   for _,ent:=range ents {
    // Upload staging names are internal implementation details. Never
    // advertise half-written content in LIST, NLST or MLSD.
    if isReservedInternalName(ent.Name()){continue}
    if len(ent.Name())>255||strings.ContainsAny(ent.Name(),"\\\r\n\x00\x1b\x7f\t") {e=errors.New("unsafe listing entry");break}
    if ent.Type()&os.ModeSymlink!=0 {e=errors.New("symlink in directory");break}
    info,er:=s.uploadRoot.Lstat(filepath.Join(rootedPath(virtual),ent.Name()))
    if er!=nil{e=er;break}
    if !info.IsDir()&&!info.Mode().IsRegular(){e=errors.New("unsupported directory entry");break}
    if verb=="NLST"{out.WriteString(ent.Name()+"\r\n");if out.Len()>8<<20{e=errors.New("listing too large");break};continue}
    if verb=="MLSD"{kind:="file";if info.IsDir(){kind="dir"};fmt.Fprintf(&out,"type=%s;size=%d; %s\r\n",kind,info.Size(),ent.Name());if out.Len()>8<<20{e=errors.New("listing too large");break};continue}
    kind:="-";if info.IsDir(){kind="d"};fmt.Fprintf(&out,"%sr--r--r-- 1 ftp ftp %d Jan 01 00:00 %s\r\n",kind,info.Size(),ent.Name());if out.Len()>8<<20{e=errors.New("listing too large");break}
   }
   if e!=nil{_ = c.reply(550,"Listing unavailable");continue}
   if out.Len()>8<<20{_ = c.reply(552,"Listing too large");continue}
   s.sendData(c,strings.NewReader(out.String()),int64(out.Len()))
  case "RETR":
   _,virtual,e:=s.resolve(c.cwd,arg)
   if e!=nil{_ = c.reply(550,"File unavailable");continue}
   f,e:=s.uploadRoot.Open(rootedPath(virtual))
   if e!=nil{_ = c.reply(550,"File unavailable");continue}
   info,e:=f.Stat()
   if e!=nil||!info.Mode().IsRegular(){_ = f.Close();_ = c.reply(550,"File unavailable");continue}
   if info.Size()>1<<30 { _ = f.Close();_ = c.reply(552,"File exceeds Preview 1 GiB download limit");continue }
   s.sendData(c,io.LimitReader(f,info.Size()),info.Size());_ = f.Close()
  case "STOR":
   s.receiveFile(c,arg)
  case "MKD":
   s.createDirectory(c,arg)
  case "DELE":
   s.deleteFile(c,arg)
  case "RMD":
   s.removeDirectory(c,arg)
  case "RNFR":
   s.startRename(c,arg)
  case "RNTO":
   s.finishRename(c,arg)
  case "APPE","SITE","PORT","EPRT","REST","ALLO":
   _ = c.reply(550,"Mutation or active data mode disabled during read-only server Preview")
  default:_ = c.reply(502,"Command not implemented")
  }
 }
}
func (s *Server) sendData(c *client, source io.Reader, expectedBytes int64){
 if !c.private||c.passive==nil{_ = c.reply(425,"Private passive connection required");return}
 ln:=c.passive;c.passive=nil;defer ln.Close()
 _ = c.reply(150,"Opening protected data connection")
 if tcp,ok:=ln.(*net.TCPListener);ok{_ = tcp.SetDeadline(time.Now().Add(15*time.Second))}
 raw,err:=ln.Accept();if err!=nil{_ = c.reply(425,"Data connection timed out");return}
 defer raw.Close()
 _ = raw.SetDeadline(time.Now().Add(60*time.Second))
 // Bind the data socket to the same client address as the control channel.
 if !raw.RemoteAddr().(*net.TCPAddr).IP.Equal(c.conn.RemoteAddr().(*net.TCPAddr).IP){_ = c.reply(425,"Data peer mismatch");return}
 secure:=tls.Server(raw,s.tls);if err=secure.Handshake();err!=nil{_ = c.reply(425,"Data TLS handshake failed");return}
 defer secure.Close()
 transmitted,err:=io.Copy(secure,source)
 if err!=nil||transmitted!=expectedBytes{_ = c.reply(451,"Data transfer incomplete")}else{_ = c.reply(226,"Transfer complete")}
}
// Reserved staging files must never be addressable by FTP clients while a
// protected upload is being streamed. Comparisons ignore case for NTFS.
func isReservedInternalName(name string) bool {
 return strings.HasPrefix(strings.ToLower(name),".ghostftp-upload-")
}
func (s *Server) resolve(cwd,input string)(string,string,error){
 if len(input)>4096||strings.ContainsAny(input,"\\\x00\r\n"){return "","",errors.New("invalid path")}
 for _,part:=range strings.Split(input,"/"){
  if part==".."||part=="." && input!="."||isReservedInternalName(part){
   return "","",errors.New("traversal, internal staging or ambiguous path")
  }
 }
 virtual:=input
 if !strings.HasPrefix(virtual,"/"){virtual=path.Join(cwd,virtual)}
 virtual=path.Clean("/"+strings.TrimPrefix(virtual,"/"))
 target:=filepath.Join(s.root,filepath.FromSlash(strings.TrimPrefix(virtual,"/")))
 real,err:=filepath.EvalSymlinks(target);if err!=nil{return "","",err}
 rel,err:=filepath.Rel(s.root,real)
 if err!=nil||rel==".."||strings.HasPrefix(rel,".."+string(os.PathSeparator))||filepath.IsAbs(rel){return "","",errors.New("outside root")}
 // Prevent directory entries being followed when they are symlinks.
 p:=s.root
 for _,part:=range strings.Split(strings.TrimPrefix(virtual,"/"),"/"){
  if part==""{continue}
  p=filepath.Join(p,part);info,e:=os.Lstat(p);if e!=nil{return "","",e};if info.Mode()&os.ModeSymlink!=0{return "","",errors.New("symlink forbidden")}
 }
 return real,virtual,nil
}

// rootedPath converts a valid FTP virtual absolute path into an os.Root-
// relative path. Root.Open does not accept absolute OS paths.
func rootedPath(virtual string)string {
 relative:=strings.TrimPrefix(virtual,"/")
 if relative==""{return "."}
 return filepath.FromSlash(relative)
}
