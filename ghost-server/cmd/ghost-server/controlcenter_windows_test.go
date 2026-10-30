//go:build windows

package main

import (
 "encoding/base64"
 "strings"
 "testing"
 "unicode/utf16"
)
func TestNativeControlCenterEmbedsRealWinFormsNotWebView(t *testing.T){
 if !strings.Contains(nativeControlCenter,"System.Windows.Forms") {t.Fatal("embedded native GUI missing")}
 if !strings.Contains(nativeControlCenter,"Start-Service") || !strings.Contains(nativeControlCenter,"Stop-Service") {t.Fatal("service buttons do not call actual SCM")}
 if !strings.Contains(nativeControlCenter,"Get-Service") {t.Fatal("real service state not read")}
 if !strings.Contains(nativeControlCenter,"MinimumSize") || !strings.Contains(nativeControlCenter,"WorkingArea"){t.Fatal("native GUI not bounded/resizable")}
 for _,v:=range []string{"WebBrowser","WebView2","http://","https://","Invoke-WebRequest","DownloadString"}{
  if strings.Contains(nativeControlCenter,v){t.Fatalf("forbidden embedded browser/network API %s",v)}
 }
 if len(nativeControlCenter)>11500 {t.Fatal("embedded script may exceed Windows command-line size")}
}
func TestControlCenterPowerShellEncoding(t *testing.T){
 raw,err:=base64.StdEncoding.DecodeString(encodeWindowsPowerShell("Hello 🚀"))
 if err!=nil {t.Fatal(err)}
 if len(raw)%2!=0 {t.Fatal("odd UTF16 byte length")}
 w:=make([]uint16,0,len(raw)/2)
 for i:=0;i<len(raw);i+=2 {w=append(w,uint16(raw[i])|uint16(raw[i+1])<<8)}
 if string(utf16.Decode(w))!="Hello 🚀" {t.Fatal("PowerShell encoding corrupts Unicode")}
}
