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
 "os"
 "path"
 "path/filepath"
 "strconv"
 "strings"
 "sync"
 "time"
)

type Config struct {
 Listen, Root, Username, Password, Certificate, PrivateKey string
}
type Server struct {
 root, user string
 password [32]byte
 tls *tls.Config
 limit chan struct{}
}
func New(c Config) (*Server,error) {
 if c.Username=="" || len(c.Username)>128 || strings.ContainsAny(c.Username,"\r\n\x00") { return nil,errors.New("valid username required") }
 if len(c.Password)<12 || len(c.Password)>256 {return nil,errors.New("password must be 12-256 bytes")}
 if c.Root=="" {return nil,errors.New("filesystem root required")}
 root,err:=filepath.Abs(c.Root);if err!=nil{return nil,err}
 info,err:=os.Lstat(root);if err!=nil{return nil,err}
 if !info.IsDir() || info.Mode()&os.ModeSymlink!=0{return nil,errors.New("root must be a real directory")}
 root,err=filepath.EvalSymlinks(root);if err!=nil{return nil,err}
 if c.Certificate==""||c.PrivateKey=="" {return nil,errors.New("certificate and private key required; plaintext FTP disabled")}
 cert,err:=tls.LoadX509KeyPair(c.Certificate,c.PrivateKey);if err!=nil{return nil,fmt.Errorf("invalid TLS identity: %w",err)}
 return &Server{root:root,user:c.Username,password:sha256.Sum256([]byte(c.Password)),tls:&tls.Config{Certificates:[]tls.Certificate{cert},MinVersion:tls.VersionTLS12},limit:make(chan struct{},32)},nil
}
func (s *Server) ListenAndServe(ctx context.Context, address string) error {
 if address=="" {address="127.0.0.1:2121"}
 ln,err:=net.Listen("tcp",address);if err!=nil{return err}
 return s.Serve(ctx,ln)
}
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
 defer ln.Close()
 go func(){<-ctx.Done();_ = ln.Close()}()
 var wg sync.WaitGroup
 defer wg.Wait()
 for {
  conn,err:=ln.Accept()
  if err!=nil {if ctx.Err()!=nil{return nil};return err}
  select {
  case s.limit<-struct{}{}:
   wg.Add(1);go func(){defer wg.Done();defer func(){<-s.limit}();s.session(conn)}()
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
 user, cwd string
 passive net.Listener
}
func (c *client) reply(code int, msg string) error {
 _,err:=fmt.Fprintf(c.w,"%d %s\r\n",code,msg)
 if err!=nil{return err};return c.w.Flush()
}
func (c *client) closePassive(){if c.passive!=nil{_ = c.passive.Close();c.passive=nil}}
func (s *Server) session(conn net.Conn){
 c:=&client{conn:conn,r:bufio.NewReader(conn),w:bufio.NewWriter(conn),cwd:"/"}
 defer func(){c.closePassive();_ = c.conn.Close()}()
 if c.reply(220,"Ghost FTP Server - explicit TLS required")!=nil{return}
 for {
  _ = c.conn.SetDeadline(time.Now().Add(90*time.Second))
  line,err:=c.r.ReadString('\n');if err!=nil{return}
  if len(line)>4096{return}
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
  if verb=="USER"{c.logged=false;c.pbsz=false;c.private=false;c.user=arg;_ = c.reply(331,"Password required");continue}
  if verb=="PASS" {
   digest:=sha256.Sum256([]byte(arg))
   if c.user==s.user&&subtle.ConstantTimeCompare(digest[:],s.password[:])==1{c.logged=true;_ = c.reply(230,"Authenticated")}else{c.logged=false;_ = c.reply(530,"Authentication failed")}
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
   resolved,virtual,e:=s.resolve(c.cwd,dest)
   if e==nil {var fi os.FileInfo;fi,e=os.Stat(resolved);if e==nil&&!fi.IsDir(){e=errors.New("not a directory")}}
   if e!=nil{_ = c.reply(550,"Directory unavailable")}else{c.cwd=virtual;_ = c.reply(250,"Directory changed")}
  case "SIZE":
   filename,_,e:=s.resolve(c.cwd,arg)
   if e==nil{var f os.FileInfo;f,e=os.Stat(filename);if e==nil&&!f.Mode().IsRegular(){e=errors.New("not regular")};if e==nil{_ = c.reply(213,strconv.FormatInt(f.Size(),10));continue}}
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
   target:=strings.TrimSpace(arg);if target==""{target="."}
   directory,_,e:=s.resolve(c.cwd,target)
   if e!=nil{_ = c.reply(550,"Directory unavailable");continue}
   ents,e:=os.ReadDir(directory)
   if e!=nil{_ = c.reply(550,"Directory unavailable");continue}
   var out strings.Builder
   for _,ent:=range ents {
    if ent.Type()&os.ModeSymlink!=0 {e=errors.New("symlink in directory");break}
    info,er:=ent.Info();if er!=nil{e=er;break}
    if verb=="NLST"{out.WriteString(ent.Name()+"\r\n");continue}
    if verb=="MLSD"{kind:="file";if info.IsDir(){kind="dir"};fmt.Fprintf(&out,"type=%s;size=%d; %s\r\n",kind,info.Size(),ent.Name());continue}
    kind:="-";if info.IsDir(){kind="d"};fmt.Fprintf(&out,"%sr--r--r-- 1 ftp ftp %d Jan 01 00:00 %s\r\n",kind,info.Size(),ent.Name())
   }
   if e!=nil{_ = c.reply(550,"Listing unavailable");continue}
   if out.Len()>8<<20{_ = c.reply(552,"Listing too large");continue}
   s.sendData(c,strings.NewReader(out.String()))
  case "RETR":
   filename,_,e:=s.resolve(c.cwd,arg)
   if e!=nil{_ = c.reply(550,"File unavailable");continue}
   f,e:=os.Open(filename)
   if e!=nil{_ = c.reply(550,"File unavailable");continue}
   info,e:=f.Stat()
   if e!=nil||!info.Mode().IsRegular(){_ = f.Close();_ = c.reply(550,"File unavailable");continue}
   if info.Size()>1<<30 { _ = f.Close();_ = c.reply(552,"File exceeds Preview 1 GiB download limit");continue }
   s.sendData(c,io.LimitReader(f,info.Size()));_ = f.Close()
  case "STOR","APPE","DELE","RMD","MKD","RNFR","RNTO","SITE","PORT","EPRT","REST","ALLO":
   _ = c.reply(550,"Mutation or active data mode disabled during read-only server Preview")
  default:_ = c.reply(502,"Command not implemented")
  }
 }
}
func (s *Server) sendData(c *client, source io.Reader){
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
 _,err=io.Copy(secure,source)
 if err!=nil{_ = c.reply(451,"Data transfer failed")}else{_ = c.reply(226,"Transfer complete")}
}
func (s *Server) resolve(cwd,input string)(string,string,error){
 if len(input)>4096||strings.ContainsAny(input,"\\\x00\r\n"){return "","",errors.New("invalid path")}
 for _,part:=range strings.Split(input,"/"){if part==".."||part=="." && input!="."{return "","",errors.New("traversal or ambiguous path")}}
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
