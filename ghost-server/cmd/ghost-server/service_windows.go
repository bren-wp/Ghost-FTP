//go:build windows

package main

import (
 "context"
 "time"

 server "ghostftp/server"
 "golang.org/x/sys/windows/svc"
)

const windowsServiceName = "GhostFTPServer"

// Windows SCM reports real service start/running/stopping states. The server
// runs with the identity configured by SCM; it must be a restricted account.
type windowsFTPSService struct {
 backend *server.Server
 listen string
}

func (s *windowsFTPSService) Execute(_ []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool,uint32) {
 changes<-svc.Status{State:svc.StartPending}
 ctx,cancel:=context.WithCancel(context.Background())
 defer cancel()
 done:=make(chan error,1)
 go func(){done<-s.backend.ListenAndServe(ctx,s.listen)}()
 select {
 case err:=<-done:
  changes<-svc.Status{State:svc.StopPending}
  if err!=nil{return false,1};return false,0
 case <-time.After(150*time.Millisecond):
 }
 changes<-svc.Status{State:svc.Running,Accepts:svc.AcceptStop|svc.AcceptShutdown}
 for {
  select {
  case err:=<-done:
   changes<-svc.Status{State:svc.StopPending}
   if err!=nil{return false,1};return false,0
  case req:=<-requests:
   switch req.Cmd {
   case svc.Interrogate:
    changes<-req.CurrentStatus
   case svc.Stop,svc.Shutdown:
    changes<-svc.Status{State:svc.StopPending}
    cancel()
    select {
    case err:=<-done:
     if err!=nil{return false,1};return false,0
    case <-time.After(30*time.Second):
     return false,2
    }
   }
  }
 }
}
func runAsWindowsService(srv *server.Server,listen string)error {
 return svc.Run(windowsServiceName,&windowsFTPSService{backend:srv,listen:listen})
}
