//go:build windows

package main

import (
 "encoding/base64"
 _ "embed"
 "fmt"
 "os"
 "os/exec"
 "syscall"
 "unicode/utf16"
)

//go:embed control_center.ps1
var nativeControlCenter string

// runNativeControlCenter displays actual Windows Forms controls, not a browser,
// a loopback HTTP page, a WebView, or a fake running server. The secure FTPS
// daemon continues to require explicitly provisioned TLS credentials.
// PowerShell receives an embedded constant program and only the executable path
// through an environment variable; no password is passed to its arguments.
func runNativeControlCenter() bool {
 exe,err:=os.Executable()
 if err!=nil {fmt.Fprintln(os.Stderr,"Ghost FTP Server could not locate its executable:",err);return false}
 encoded:=encodeWindowsPowerShell(nativeControlCenter)
 cmd:=exec.Command("powershell.exe","-NoLogo","-NoProfile","-NonInteractive","-EncodedCommand",encoded)
 cmd.Env=append(os.Environ(),"GHOSTFTP_SERVER_EXE="+exe)
 // CREATE_NO_WINDOW suppresses only an unwanted PowerShell console.
 // HideWindow:true would set STARTF_USESHOWWINDOW/SW_HIDE and could also hide
 // the first WinForms window, making the Control Center look broken.
 const createNoWindow=0x08000000
 cmd.SysProcAttr=&syscall.SysProcAttr{CreationFlags:createNoWindow}
 cmd.Stdout=os.Stdout
 cmd.Stderr=os.Stderr
 if err:=cmd.Run();err!=nil{
  fmt.Fprintln(os.Stderr,"Ghost FTP Server native control center could not start:",err)
  return false
 }
 return true
}
func encodeWindowsPowerShell(script string) string {
 units:=utf16.Encode([]rune(script))
 bytes:=make([]byte,len(units)*2)
 for i,v:=range units {bytes[2*i]=byte(v);bytes[2*i+1]=byte(v>>8)}
 return base64.StdEncoding.EncodeToString(bytes)
}
