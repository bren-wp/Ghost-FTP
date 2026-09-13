package platform

import "fmt"

const (
	windowsProcessorArchitectureIntel = 0
	windowsProcessorArchitectureAMD64 = 9
	windowsProcessorArchitectureARM64 = 12
)

func windowsArchitectureFromProcessor(code uint16) (string, error) {
	switch code {
	case windowsProcessorArchitectureIntel:
		return "x86", nil
	case windowsProcessorArchitectureAMD64:
		return "x64", nil
	case windowsProcessorArchitectureARM64:
		return "arm64", nil
	default:
		return "", fmt.Errorf("unsupported Windows processor architecture: %d", code)
	}
}
