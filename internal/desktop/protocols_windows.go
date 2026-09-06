//go:build windows

package desktop

import "strings"

type protocolSpec struct {
	Value string
	Port  string
}

// Secure explicit FTPS is the fresh/quick-connect default on Windows, matching
// Linux. Plain FTP remains available for legacy servers but is never selected
// implicitly when protocol state is missing or invalid.
var protocolSpecs = []protocolSpec{
	{Value: defaultConnectionProtocol, Port: "21"},
	{Value: "sftp", Port: "22"},
	{Value: "ftp", Port: "21"},
	{Value: "ftpsi", Port: "990"},
}

func protocolAt(index uintptr) protocolSpec {
	if int(index) >= 0 && int(index) < len(protocolSpecs) {
		return protocolSpecs[int(index)]
	}
	return protocolSpecs[0]
}

func protocolIndex(value string) uintptr {
	value = strings.ToLower(strings.TrimSpace(value))
	for i, spec := range protocolSpecs {
		if spec.Value == value {
			return uintptr(i)
		}
	}
	return 0
}
