package desktop

import "sync"

// StartupTarget contains only non-sensitive connection metadata supplied by an
// explicit OS protocol launch (for example from the official browser helper).
// Passwords, passphrases and private-key material are deliberately absent.
type StartupTarget struct {
	Protocol string
	Host     string
	Port     int
	Username string
	Path     string
}

var startupTargetState struct {
	sync.Mutex
	target *StartupTarget
}

// SetStartupTarget records a one-shot target for the next desktop UI startup.
// The target lives only in process memory and is cleared as soon as a UI reads
// it.
func SetStartupTarget(target StartupTarget) {
	startupTargetState.Lock()
	defer startupTargetState.Unlock()
	copyTarget := target
	startupTargetState.target = &copyTarget
}

func takeStartupTarget() (StartupTarget, bool) {
	startupTargetState.Lock()
	defer startupTargetState.Unlock()
	if startupTargetState.target == nil {
		return StartupTarget{}, false
	}
	target := *startupTargetState.target
	startupTargetState.target = nil
	return target, true
}

func startupTargetPort(target StartupTarget) int {
	if target.Port > 0 {
		return target.Port
	}
	switch target.Protocol {
	case "sftp":
		return 22
	case "ftpsi":
		return 990
	default:
		return 21
	}
}
