package ghostserver

import (
 "crypto/tls"
 "errors"
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
    strings.ContainsAny(name,"\x00\r\n\x1b\x7f") {
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

// receiveFile implements a bounded, TLS-protected STOR of ONLY new files.
// It stages to a random name in the destination directory and uses a hardlink
// as the atomic create-if-absent commit. On filesystems without same-dir
// hardlinks (or when the destination already exists), it fails closed.
// This does NOT eliminate parent-directory replacement races by privileged
// local processes. Deploy with a dedicated OS identity and a private root.
func (s *Server) receiveFile(c *client, operand string) {
 if !s.allowUploads { _=c.reply(550,"Uploads disabled by server policy");return }
 if !c.private || c.passive==nil { _=c.reply(425,"Private passive data connection required");return }
 dest,err:=s.createDestination(c.cwd,operand)
 if err!=nil {_=c.reply(550,"Upload destination unavailable or already exists");return}
 staged,err:=os.CreateTemp(filepath.Dir(dest),".ghostftp-upload-*")
 if err!=nil{_=c.reply(451,"Unable to stage upload");return}
 defer os.Remove(staged.Name())
 defer staged.Close()
 if err=staged.Chmod(0600);err!=nil {_=c.reply(451,"Cannot protect staged upload");return}
 ln:=c.passive;c.passive=nil
 defer ln.Close()
 if err=c.reply(150,"Opening protected upload connection");err!=nil{return}
 if tcp,ok:=ln.(*net.TCPListener);ok{_ =tcp.SetDeadline(time.Now().Add(15*time.Second))}
 raw,err:=ln.Accept()
 if err!=nil{_ = c.reply(425,"Data connection timeout");return}
 defer raw.Close()
 _=raw.SetDeadline(time.Now().Add(60*time.Second))
 remote,rok:=raw.RemoteAddr().(*net.TCPAddr)
 control,cok:=c.conn.RemoteAddr().(*net.TCPAddr)
 if !rok || !cok || !remote.IP.Equal(control.IP){_=c.reply(425,"Data peer mismatch");return}
 secure:=tls.Server(raw,s.tls)
 if err=secure.Handshake();err!=nil{_=c.reply(425,"Data TLS handshake failed");return}
 defer secure.Close()
 n,err:=io.Copy(staged,io.LimitReader(secure,s.maxUploadBytes+1))
 if err!=nil{_=c.reply(451,"Protected upload interrupted");return}
 if n>s.maxUploadBytes{_=c.reply(552,"Upload exceeds configured size limit");return}
 if err=staged.Sync();err!=nil{_=c.reply(451,"Unable to sync staged upload");return}
 if err=staged.Close();err!=nil{_=c.reply(451,"Unable to close staged upload");return}
 // Hard links never replace an already-existing destination. This is the
 // overwrite-prevention commit point even when another client wins the race.
 if err=os.Link(staged.Name(),dest);err!=nil{
  _=c.reply(550,"Upload destination exists or filesystem cannot commit safely")
  return
 }
 _=c.reply(226,"Protected upload committed without overwrite")
}
