//go:build windows

package desktop

import "strings"

type protocolSpec struct {
	Value string
	Label string
	Port  string
}

var protocolSpecs = []protocolSpec{
	{Value: "ftp", Label: "FTP", Port: "21"},
	{Value: "ftps", Label: "FTPS (eksplicitni)", Port: "21"},
	{Value: "sftp", Label: "SFTP", Port: "22"},
	{Value: "ftpsi", Label: "FTPS (implicitni)", Port: "990"},
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
