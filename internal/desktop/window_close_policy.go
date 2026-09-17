package desktop

type windowCloseAction uint8

const (
	windowCloseNow windowCloseAction = iota
	windowCloseWaitForProfileMutation
	windowCloseConfirmActiveWork
)

// deriveWindowCloseAction keeps the native titlebar lifecycle independent from
// any particular dialog implementation. A profile-store mutation is the only
// state that must defer closing outright; connections and transfer work require
// an explicit user confirmation, while an idle window may close immediately.
func deriveWindowCloseAction(profileMutationBusy, connected, connectionBusy, activeTransfers bool) windowCloseAction {
	if profileMutationBusy {
		return windowCloseWaitForProfileMutation
	}
	if connected || connectionBusy || activeTransfers {
		return windowCloseConfirmActiveWork
	}
	return windowCloseNow
}
