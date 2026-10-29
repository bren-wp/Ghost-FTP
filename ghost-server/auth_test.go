package ghostserver

import (
 "net"
 "testing"
 "time"
)

func TestPeerAuthThrottleSurvivesNewConnectionsAndRejectsCorrectPasswordDuringLockout(t *testing.T) {
 srv,_,stop:=testServer(t)
 defer stop()
 peerA:=&net.TCPAddr{IP:net.ParseIP("127.0.0.1"),Port:11001}
 peerAReconnected:=&net.TCPAddr{IP:net.ParseIP("127.0.0.1"),Port:11002}
 peerB:=&net.TCPAddr{IP:net.ParseIP("127.0.0.2"),Port:11999}
 for attempt:=1;attempt<8;attempt++ {
  if !srv.recordAuthAttempt(peerA,false) {t.Fatalf("blocked too early at attempt %d",attempt)}
 }
 if srv.recordAuthAttempt(peerAReconnected,false) {t.Fatal("eighth failure accepted after reconnect")}
 if srv.recordAuthAttempt(peerAReconnected,true) {t.Fatal("correct password bypassed existing lockout")}
 if !srv.recordAuthAttempt(peerB,true) {t.Fatal("other loopback peer incorrectly blocked")}
 // Reset a time-expired entry without actually sleeping in the test.
 srv.authMu.Lock()
 srv.authFailures["127.0.0.1"]=authFailures{count:8,
  last:time.Now().Add(-11*time.Minute),blockedUntil:time.Now().Add(-time.Second)}
 srv.authMu.Unlock()
 if !srv.recordAuthAttempt(peerAReconnected,true) {t.Fatal("expired lockout did not recover")}
 srv.authMu.Lock()
 _,stillBlocked:=srv.authFailures["127.0.0.1"]
 srv.authMu.Unlock()
 if stillBlocked {t.Fatal("valid login did not clear expired lockout")}
}

func TestAuthenticationPeerHistoryHasMemoryBound(t *testing.T) {
 srv,_,stop:=testServer(t)
 defer stop()
 for i:=0;i<1024;i++ {
  peer:=&net.TCPAddr{IP:net.IPv4(10,byte(i/256),byte(i%256),1),Port:12345}
  if !srv.recordAuthAttempt(peer,false){t.Fatalf("failed to register peer %d",i)}
 }
 blockedPeer:=&net.TCPAddr{IP:net.ParseIP("192.0.2.55"),Port:1}
 if srv.recordAuthAttempt(blockedPeer,false) {t.Fatal("IP history overflow bypassed rate limit")}
 srv.authMu.Lock()
 entries:=len(srv.authFailures)
 srv.authMu.Unlock()
 if entries>1024 {t.Fatalf("peer history unbounded: %d",entries)}
}
