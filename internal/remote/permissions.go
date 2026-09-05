package remote

import "strings"

// normalizePermissionDisplay accepts only bounded permission forms that can be
// obtained from UNIX-style LIST/SFTP listings or the MLSD unix.mode extension.
// Unknown server-specific capability strings are intentionally not shown as if
// they were POSIX file modes.
func normalizePermissionDisplay(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 10 && len(raw) <= 11 {
		first := raw[0]
		if !strings.ContainsRune("-bcdlps", rune(first)) {
			return ""
		}
		for _, ch := range raw[1:10] {
			if !strings.ContainsRune("rwxstST-", ch) {
				return ""
			}
		}
		return raw[:10]
	}
	if len(raw) == 3 || len(raw) == 4 {
		for _, ch := range raw {
			if ch < '0' || ch > '7' {
				return ""
			}
		}
		return raw
	}
	return ""
}
