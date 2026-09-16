//go:build windows

package desktop

// siteManagerBlocksStartupTarget keeps a browser handoff pending for the whole
// Site Manager modal lifetime. This covers synchronous credential-consent and
// SaveProfile paths that can restore selectedProfileID after nested modal
// message pumping. The target is applied only after the manager is closed.
func siteManagerBlocksStartupTarget(parent *app) bool {
	if parent == nil {
		return false
	}
	blocked := false
	siteManagerStates.Range(func(_, value any) bool {
		state, ok := value.(*siteManagerState)
		if !ok || state == nil || state.parent != parent || state.closed {
			return true
		}
		blocked = true
		return false
	})
	return blocked
}
