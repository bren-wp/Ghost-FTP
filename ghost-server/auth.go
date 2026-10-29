package ghostserver

import (
 "net"
 "time"
)

type authFailures struct {
 count int
 last time.Time
 blockedUntil time.Time
}

// recordAuthAttempt denies a peer after eight failed username/password pairs
// within ten minutes. Unlike a per-socket counter it survives reconnects.
// A known-good login cannot override an active block. This is not a complete
// distributed rate limiter; public exposure still requires trusted ingress.
func (s *Server) recordAuthAttempt(peer net.Addr, valid bool) bool {
 host,_,err:=net.SplitHostPort(peer.String())
 if err!=nil{host=peer.String()}
 now:=time.Now()

 s.authMu.Lock()
 defer s.authMu.Unlock()

 // Bounded in-memory IP history; stale entries expire without a background
 // cleanup worker or an ever-growing map under malicious reconnect bursts.
 if len(s.authFailures)>=1024 {
  for name,entry:=range s.authFailures {
   if now.Sub(entry.last)>10*time.Minute && !now.Before(entry.blockedUntil){
    delete(s.authFailures,name)
   }
  }
 }
 entry:=s.authFailures[host]
 if now.Before(entry.blockedUntil){return false}
 if entry.last.IsZero()||now.Sub(entry.last)>10*time.Minute {
  entry.count=0
  entry.blockedUntil=time.Time{}
 }
 if valid {
  delete(s.authFailures,host)
  return true
 }
 if _,exists:=s.authFailures[host];!exists && len(s.authFailures)>=1024 {
  // Fail closed on per-peer capacity exhaustion instead of forgetting a
  // recently blocked peer. Operators must use network admission controls.
  return false
 }
 entry.count++
 entry.last=now
 if entry.count>=8 {
  entry.blockedUntil=now.Add(time.Minute)
 }
 s.authFailures[host]=entry
 return entry.count<8
}
