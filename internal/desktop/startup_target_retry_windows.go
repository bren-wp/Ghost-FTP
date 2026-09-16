//go:build windows

package desktop

import (
	"sync"
	"time"
)

var startupTargetRetryPending sync.Map

// scheduleStartupTargetRetry keeps at most one delayed retry per desktop app.
// The one-shot target itself remains protected by startupTargetState and is not
// consumed while Site Manager can still restore a credential-bearing profile.
func scheduleStartupTargetRetry(a *app) {
	if a == nil || a.hwnd == 0 {
		return
	}
	if _, loaded := startupTargetRetryPending.LoadOrStore(a, struct{}{}); loaded {
		return
	}
	time.AfterFunc(250*time.Millisecond, func() {
		startupTargetRetryPending.Delete(a)
		dispatchStartupTargetToOpenWindow()
	})
}
