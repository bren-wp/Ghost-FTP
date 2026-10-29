package ghostserver

import (
 "bufio"
 "context"
 "crypto/rand"
 "crypto/rsa"
 "crypto/tls"
 "crypto/x509"
 "crypto/x509/pkix"
 "encoding/pem"
 "fmt"
 "io"
 "math/big"
 "net"
 "os"
 "path/filepath"
 "strconv"
 "strings"
 "testing"
 "time"
)

func certFiles(t *testing.T)(string,string){
 t.Helper()
 private,err:=rsa.GenerateKey(rand.Reader,2048);if err!=nil{t.Fatal(err)}
 tpl:=&x509.Certificate{SerialNumber:big.NewInt(1),Subject:pkix.Name{CommonName:"localhost"},NotBefore:time.Now().Add(-time.Minute),NotAfter:time.Now().Add(time.Hour),KeyUsage:x509.KeyUsageDigitalSignature|x509.KeyUsageKeyEncipherment,ExtKeyUsage:[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},DNSNames:[]string{"localhost"}}
 cert,err:=x509.CreateCertificate(rand.Reader,tpl,tpl,&private.PublicKey,private);if err!=nil{t.Fatal(err)}
 dir:=t.TempDir()
 crt,key:=filepath.Join(dir,"tls.pem"),filepath.Join(dir,"tls.key")
 if err=os.WriteFile(crt,pem.EncodeToMemory(&pem.Block{Type:"CERTIFICATE",Bytes:cert}),0600);err!=nil{t.Fatal(err)}
 if err=os.WriteFile(key,pem.EncodeToMemory(&pem.Block{Type:"RSA PRIVATE KEY",Bytes:x509.MarshalPKCS1PrivateKey(private)}),0600);err!=nil{t.Fatal(err)}
 return crt,key
}
func testServer(t *testing.T)(*Server,string,func()){
 t.Helper()
 root:=t.TempDir()
 if err:=os.WriteFile(filepath.Join(root,"test.txt"),[]byte("genuine bytes"),0600);err!=nil{t.Fatal(err)}
 crt,key:=certFiles(t)
 srv,err:=New(Config{Root:root,Username:"qa",Password:"long-private-test-password",Certificate:crt,PrivateKey:key})
 if err!=nil{t.Fatal(err)}
 ln,err:=net.Listen("tcp","127.0.0.1:0");if err!=nil{t.Fatal(err)}
 ctx,cancel:=context.WithCancel(context.Background())
 done:=make(chan error,1)
 go func(){done<-srv.Serve(ctx,ln)}()
 return srv,ln.Addr().String(),func(){cancel();select{case <-done:case <-time.After(3*time.Second):t.Error("server did not stop")}}
}
type ftpc struct {conn net.Conn;r *bufio.Reader;w *bufio.Writer}
func (c *ftpc) read(t *testing.T,prefix string)string{
 t.Helper();_ = c.conn.SetDeadline(time.Now().Add(3*time.Second))
 line,err:=c.r.ReadString('\n');if err!=nil{t.Fatal(err)}
 if !strings.HasPrefix(line,prefix){t.Fatalf("expected reply %s, got %q",prefix,line)}
 return line
}
func (c *ftpc) command(t *testing.T,prefix,cmd string) string{
 t.Helper();if _,err:=fmt.Fprintf(c.w,"%s\r\n",cmd);err!=nil{t.Fatal(err)};if err:=c.w.Flush();err!=nil{t.Fatal(err)}
 return c.read(t,prefix)
}
func connect(t *testing.T,addr string)*ftpc{
 t.Helper();conn,err:=net.DialTimeout("tcp",addr,3*time.Second);if err!=nil{t.Fatal(err)}
 c:=&ftpc{conn:conn,r:bufio.NewReader(conn),w:bufio.NewWriter(conn)}
 c.read(t,"220 ")
 return c
}
func secure(t *testing.T,c *ftpc){
 t.Helper();c.command(t,"234 ","AUTH TLS")
 encrypted:=tls.Client(c.conn,&tls.Config{InsecureSkipVerify:true,MinVersion:tls.VersionTLS12}) // only a self-signed local test fixture
 if err:=encrypted.Handshake();err!=nil{t.Fatal(err)}
 c.conn=encrypted;c.r=bufio.NewReader(encrypted);c.w=bufio.NewWriter(encrypted)
}
func login(t *testing.T,c *ftpc){
 t.Helper();secure(t,c);c.command(t,"331 ","USER qa");c.command(t,"230 ","PASS long-private-test-password");c.command(t,"200 ","PBSZ 0");c.command(t,"200 ","PROT P")
}
func TestRejectPlaintextCredentialsAndAuthenticationFailures(t *testing.T){
 _,addr,stop:=testServer(t);defer stop()
 c:=connect(t,addr);defer c.conn.Close()
 c.command(t,"534 ","USER qa")
 c.command(t,"534 ","PASS long-private-test-password")
 secure(t,c)
 c.command(t,"331 ","USER qa")
 c.command(t,"530 ","PASS incorrect-password")
 c.command(t,"530 ","PWD")
 c.command(t,"331 ","USER qa")
 c.command(t,"230 ","PASS long-private-test-password")
 c.command(t,"534 ","PROT C")
 c.command(t,"200 ","PBSZ 0")
 c.command(t,"534 ","PROT C")
 c.command(t,"200 ","PROT P")
}
// RFC 4217 explicitly permits PBSZ/PROT immediately after AUTH TLS and
// before USER/PASS. The real desktop SuppaFTP transport uses this sequence.
// Negotiating transport protection must never grant filesystem permissions.
func TestFTPSProtectionBeforeLoginMatchesRealDesktopClient(t *testing.T) {
 _,addr,stop:=testServer(t)
 defer stop()
 c:=connect(t,addr)
 defer c.conn.Close()

 // Plaintext and unauthenticated transfers must remain impossible.
 c.command(t,"534 ","PBSZ 0")
 c.command(t,"534 ","PROT P")
 secure(t,c)
 c.command(t,"534 ","PROT P")
 c.command(t,"501 ","PBSZ 1")
 c.command(t,"200 ","PBSZ 0")
 c.command(t,"534 ","PROT C")
 c.command(t,"200 ","PROT P")
 c.command(t,"530 ","PWD")
 c.command(t,"530 ","SIZE test.txt")
 c.command(t,"530 ","LIST")
 c.command(t,"331 ","USER qa")
 c.command(t,"230 ","PASS long-private-test-password")
 // USER/PASS does not undo already-negotiated private data protection.
 c.command(t,"229 ","EPSV")
 c.command(t,"213 13","SIZE test.txt")

 // A second USER invalidates authorization, but does not lower PROT P.
 c.command(t,"331 ","USER qa")
 c.command(t,"530 ","PWD")
 c.command(t,"230 ","PASS long-private-test-password")
 c.command(t,"229 ","EPSV")
}
func TestRealFTPSDownloadReadOnlyAndPassiveDataTLS(t *testing.T){
 _,addr,stop:=testServer(t);defer stop()
 c:=connect(t,addr);defer c.conn.Close()
 login(t,c)
 c.command(t,"257 ","PWD")
 c.command(t,"213 13","SIZE test.txt")
 c.command(t,"550 ","CWD ../")
 c.command(t,"550 ","STOR test.txt")
 reply:=c.command(t,"229 ","EPSV")
 pos:=strings.Index(reply,"(|||");if pos<0{t.Fatalf("bad EPSV reply %q",reply)}
 end:=strings.Index(reply[pos+4:],"|");if end<0{t.Fatal("missing EPSV delimiter")}
 port,err:=strconv.Atoi(reply[pos+4:pos+4+end]);if err!=nil{t.Fatal(err)}
 controlAddr,err:=net.ResolveTCPAddr("tcp",addr);if err!=nil{t.Fatal(err)}
 data,err:=net.DialTimeout("tcp",net.JoinHostPort(controlAddr.IP.String(),strconv.Itoa(port)),2*time.Second)
 if err!=nil{t.Fatal(err)}
 defer data.Close()
 c.command(t,"150 ","RETR test.txt")
 encrypted:=tls.Client(data,&tls.Config{InsecureSkipVerify:true,MinVersion:tls.VersionTLS12}) // self-signed fixture only
 if err:=encrypted.Handshake();err!=nil{t.Fatal(err)}
 bytes,err:=io.ReadAll(encrypted);if err!=nil{t.Fatal(err)}
 if string(bytes)!="genuine bytes"{t.Fatalf("wrong bytes %q",bytes)}
 c.read(t,"226 ")
}
func TestOutsideRootAndSymlinksFailClosed(t *testing.T){
 srv,_,stop:=testServer(t);defer stop()
 for _,name:=range []string{"../etc/passwd","/../test.txt","..\\secret","foo/../test.txt"}{
  if _,_,err:=srv.resolve("/",name);err==nil{t.Errorf("accepted %q",name)}
 }
 outside:=t.TempDir()
 if err:=os.WriteFile(filepath.Join(outside,"secret"),[]byte("secret"),0600);err!=nil{t.Fatal(err)}
 link:=filepath.Join(srv.root,"link")
 if err:=os.Symlink(outside,link);err!=nil{t.Skipf("symlink unavailable: %v",err)}
 if _,_,err:=srv.resolve("/","link/secret");err==nil{t.Fatal("followed external symlink")}
}
func TestBadConfigurationFailsClosed(t *testing.T){
 root:=t.TempDir()
 if _,err:=New(Config{Root:root,Username:"qa",Password:"weak"});err==nil{t.Fatal("weak password accepted")}
 if _,err:=New(Config{Root:root,Username:"qa",Password:"strong-password-123"});err==nil{t.Fatal("missing TLS cert accepted")}
}

func TestAuthenticationRateLimitFailsClosed(t *testing.T) {
 _,addr,stop:=testServer(t);defer stop()
 c:=connect(t,addr);defer c.conn.Close()
 secure(t,c)
 c.command(t,"331 ","USER qa")
 c.command(t,"530 ","PASS wrong-one")
 c.command(t,"530 ","PASS wrong-two")
 c.command(t,"421 ","PASS wrong-three")
 _ = c.conn.SetReadDeadline(time.Now().Add(2*time.Second))
 if _,err:=c.r.ReadString('\n');err==nil{t.Fatal("session stayed open after third failed password")}
}

func TestClientWithoutTrustedTLSIdentityRefusesServer(t *testing.T) {
 _,addr,stop:=testServer(t);defer stop()
 c:=connect(t,addr);defer c.conn.Close()
 c.command(t,"234 ","AUTH TLS")
 strict:=tls.Client(c.conn,&tls.Config{ServerName:"localhost",MinVersion:tls.VersionTLS12})
 if err:=strict.Handshake();err==nil{t.Fatal("client accepted self-signed untrusted server identity")}
}

func TestShutdownClosesIdleAuthenticatedControlSessions(t *testing.T) {
 _,addr,stop:=testServer(t)
 c:=connect(t,addr)
 secure(t,c)
 stop()
 defer c.conn.Close()
 _ = c.conn.SetReadDeadline(time.Now().Add(2*time.Second))
 if _,err:=c.r.ReadString('\n');err==nil{t.Fatal("idle FTPS session survived server shutdown")}
}
