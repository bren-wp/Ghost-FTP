// Command testfixture creates ephemeral, clearly non-production TLS files for
// the Windows SCM integration test on an isolated GitHub-hosted runner.
// Not included in the release artifact or deployed application.
package main

import (
 "crypto/rand"
 "crypto/rsa"
 "crypto/x509"
 "crypto/x509/pkix"
 "encoding/pem"
 "flag"
 "log"
 "math/big"
 "net"
 "os"
 "path/filepath"
 "time"
)

func main() {
 output:=flag.String("out","","Temporary CI output folder")
 flag.Parse()
 if *output==""{log.Fatal("-out required")}
 if err:=os.MkdirAll(filepath.Join(*output,"root"),0700);err!=nil{log.Fatal(err)}
 key,err:=rsa.GenerateKey(rand.Reader,2048);if err!=nil{log.Fatal(err)}
 tmpl:=&x509.Certificate{
  SerialNumber:big.NewInt(1),Subject:pkix.Name{CommonName:"localhost"},
  NotBefore:time.Now().Add(-time.Minute),NotAfter:time.Now().Add(30*time.Minute),
  DNSNames:[]string{"localhost"},IPAddresses:[]net.IP{net.ParseIP("127.0.0.1")},
  KeyUsage:x509.KeyUsageDigitalSignature|x509.KeyUsageKeyEncipherment,
  ExtKeyUsage:[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
 }
 cert,err:=x509.CreateCertificate(rand.Reader,tmpl,tmpl,&key.PublicKey,key);if err!=nil{log.Fatal(err)}
 for _,item:=range []struct{name string;content []byte}{
  {"server-cert.pem",pem.EncodeToMemory(&pem.Block{Type:"CERTIFICATE",Bytes:cert})},
  {"server-key.pem",pem.EncodeToMemory(&pem.Block{Type:"RSA PRIVATE KEY",Bytes:x509.MarshalPKCS1PrivateKey(key)})},
  {"password.txt",[]byte("ci-local-private-test-password-only")},
 } {
  if err:=os.WriteFile(filepath.Join(*output,item.name),item.content,0600);err!=nil{log.Fatal(err)}
 }
}
