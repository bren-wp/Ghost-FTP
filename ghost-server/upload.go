package ghostserver

import (
 "crypto/rand"
 "crypto/tls"
 "encoding/hex"
 "errors"
 "log"
 "io"
 "net"
 "os"
 "path"
 "path/filepath"
 "strings"
 "time"
)

// createDestination deliberately resolves only the EXISTING parent. resolve()
// for reads rejects absent objects; writes must never treat an absent child
// as proof that a different parent has not been reached via a symlink.
func (s *Server) createDestination(cwd, operand string) (string, error) {
 if len(operand)==0 || len(operand)>4096 ||
    strings.HasSuffix(operand,"/") || strings.ContainsAny(operand,"\\\r\n\x00") {
  return "", errors.New("invalid upload destination")
 }
 for _,part:=range strings.Split(operand,"/") {
  if part=="." || part==".." { return "",errors.New("ambiguous upload path") }
 }
 name:=path.Base(operand)
 if name=="" || name=="." || name==".." || len(name)>255 ||
    isReservedInternalName(name) || strings.ContainsAny(name,"\x00\r\n\x1b\x7f") {
  return "",errors.New("invalid upload name")
 }
 parent,_,err:=s.resolve(cwd,path.Dir(operand))
 if err!=nil{return "",err}
 info,err:=os.Lstat(parent)
 if err!=nil || !info.IsDir() || info.Mode()&os.ModeSymlink!=0 {
  return "",errors.New("upload parent unavailable")
 }
 dest:=filepath.Join(parent,filepath.FromSlash(name))
 _,err=os.Lstat(dest)
 if err==nil{return "",errors.New("destination already exists")}
 if !os.IsNotExist(err){return "",errors.New("cannot inspect destination")}
 return dest,nil
}

// createDirectory is an explicitly enabled and root-scoped MKD operation.
// It refuses existing entries and never creates missing parent directories.
func (s *Server) createDirectory(c *client, operand string) {
 if !s.allowUploads{_ = c.reply(550,"Directory creation disabled by server policy");return}
 if !c.private{_ = c.reply(534,"Private TLS data protection must be negotiated");return}
 dest,err:=s.createDestination(c.cwd,operand)
 if err!=nil{_ = c.reply(550,"Directory destination unavailable or already exists");return}
 relative,err:=filepath.Rel(s.root,dest)
 if err!=nil || filepath.IsAbs(relative) || relative==".." || strings.HasPrefix(relative,".."+string(os.PathSeparator)){
  _ = c.reply(550,"Directory outside served root");return
 }
 if err=s.uploadRoot.Mkdir(relative,0700);err!=nil{
  _ = c.reply(550,"Directory already exists or could not be created");return
 }
 _ = c.reply(257,"Directory created securely")
}

// receiveFile implements a bounded, TLS-protected STOR of ONLY new files.
// It stages to a random name in the destination directory and uses a hardlink
// as the atomic create-if-absent commit. On filesystems without same-dir
// hardlinks (or when the destination already exists), it fails closed.
// An os.Root directory handle prevents writes escaping the configured root
// through symlink swaps. It is not a full authorization boundary against a
// privileged local actor that can move the entire served root, and it cannot
// substitute for a dedicated OS identity and filesystem ACLs.
func (s *Server) receiveFile(c *client, operand string) {
 declaredSize,hasDeclaredSize:=c.expectedUploadBytes,c.hasExpectedUpload
 // Never allow a previous ALLO command to authorize a later STOR.
 c.hasExpectedUpload=false
 c.expectedUploadBytes=0
 if !s.allowUploads { _=c.reply(550,"Uploads disabled by server policy");return }
 // A Go TLS reader can report a bare TCP FIN as EOF even without
 // close_notify. A clean read result therefore does NOT prove the source
 // file was complete. Every STOR must carry a single-use authenticated
 // ALLO byte count before any data channel is opened.
 if !hasDeclaredSize { _=c.reply(550,"Exact upload length required via ALLO before STOR");return }
 if !c.private || c.passive==nil { _=c.reply(425,"Private passive data connection required");return }
 dest,err:=s.createDestination(c.cwd,operand)
 if err!=nil {_=c.reply(550,"Upload destination unavailable or already exists");return}
 if s.uploadRoot==nil{_=c.reply(451,"Upload root is unavailable");return}
 parentRel,err:=filepath.Rel(s.root,filepath.Dir(dest))
 if err!=nil{_=c.reply(451,"Cannot resolve upload parent");return}
 parent,err:=s.uploadRoot.OpenRoot(parentRel)
 if err!=nil{_=c.reply(550,"Upload parent is unavailable");return}
 defer parent.Close()
 // Securely create a random staging name relative to an open directory
 // handle, NOT a path that a local attacker can replace with a symlink.
 var staged *os.File
 var tmpName string
 for i:=0;i<8;i++ {
  var randomBytes [16]byte
  if _,err=rand.Read(randomBytes[:]);err!=nil{_=c.reply(451,"Unable to protect upload name");return}
  tmpName=".ghostftp-upload-"+hex.EncodeToString(randomBytes[:])
  staged,err=parent.OpenFile(tmpName,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0600)
  if err==nil{break}
  if !os.IsExist(err){_=c.reply(451,"Unable to stage upload safely");return}
 }
 if staged==nil{_=c.reply(451,"Unable to reserve upload staging");return}
 // Every terminal FTP reply must describe the visible filesystem state.
 // Clean up the random staging file BEFORE sending a failure or success
 // response; otherwise another client can observe a transient leaked file.
 cleanup:=func(){
  _=staged.Close()
  _=parent.Remove(tmpName)
 }
 defer cleanup()
 fail:=func(code int,msg string){cleanup();_=c.reply(code,msg)}
 ln:=c.passive;c.passive=nil
 defer ln.Close()
 if err=c.reply(150,"Opening protected upload connection");err!=nil{return}
 if tcp,ok:=ln.(*net.TCPListener);ok{_ =tcp.SetDeadline(time.Now().Add(15*time.Second))}
 raw,err:=ln.Accept()
 if err!=nil{fail(425,"Data connection timeout");return}
 defer raw.Close()
 _=raw.SetDeadline(time.Now().Add(60*time.Second))
 remote,rok:=raw.RemoteAddr().(*net.TCPAddr)
 control,cok:=c.conn.RemoteAddr().(*net.TCPAddr)
 if !rok || !cok || !remote.IP.Equal(control.IP){fail(425,"Data peer mismatch");return}
 secure:=tls.Server(raw,s.tls)
 if err=secure.Handshake();err!=nil{fail(425,"Data TLS handshake failed");return}
 defer secure.Close()
 n,err:=io.Copy(staged,io.LimitReader(secure,s.maxUploadBytes+1))
 if n>s.maxUploadBytes{fail(552,"Upload exceeds configured size limit");return}
 if hasDeclaredSize && n!=declaredSize {
  // Log only byte counts and a generic error category; never log remote
  // filenames, authenticated usernames, passwords or uploaded contents.
  log.Printf("FTPS integrity reject: declared_bytes=%d received_bytes=%d transport_error=%T",declaredSize,n,err)
  fail(451,"Protected upload length differs from authenticated ALLO");return
 }
 if err!=nil {
  // Some native-TLS FTP clients close the encrypted data TCP stream without
  // sending TLS close_notify. That produces io.ErrUnexpectedEOF in Go.
  // NEVER treat it as successful by itself: the authenticated control
  // channel must have declared an exact byte count with ALLO, and every
  // byte must have arrived in authenticated TLS records. A missing byte,
  // other transport error, or missing ALLO is still a hard failure.
  if !(errors.Is(err,io.ErrUnexpectedEOF) && hasDeclaredSize && n==declaredSize) {
   fail(451,"Protected upload interrupted or TLS truncation detected");return
  }
 }
 if err=staged.Sync();err!=nil{fail(451,"Unable to sync staged upload");return}
 if err=staged.Close();err!=nil{fail(451,"Unable to close staged upload");return}
 // Hard links never replace an already-existing destination. This is the
 // overwrite-prevention commit point even when another client wins the race.
 if err=parent.Link(tmpName,filepath.Base(dest));err!=nil{
  fail(550,"Upload destination exists or filesystem cannot commit safely")
  return
 }
 cleanup()
 _=c.reply(226,"Protected upload committed without overwrite")
}
