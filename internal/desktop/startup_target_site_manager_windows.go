//go:build windows

package desktop

// siteManagerBlocksStartupTarget keeps a browser handoff pending for the whole
// Site Manager lifetime, including post-WM_DESTROY profile restoration. The
// state remains mapped until openSiteManager has finished restoring/selecting
// profiles and any requested saved-profile connection has been started.
func siteManagerBlocksStartupTarget(parent *app) bool {
	if parent == nil {
		return false
	}
	blocked := false
	siteManagerStates.Range(func(_, value any) bool {
		state, ok := value.(*siteManagerState)
		if !ok || state == nil || state.parent != parent {
			return true
		}
		blocked = true
		return false
	})
	return blocked
}
