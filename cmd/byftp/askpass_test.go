package main

import "testing"

func TestValidAskpassInvocation(t *testing.T) {
	exe := `C:\\Program Files\\ByFTP\\ByFTP.exe`
	token := "0123456789abcdef0123456789abcdef"
	if !validAskpassInvocation(exe, exe, "force", token) {
		t.Fatal("expected controlled AskPass invocation to validate")
	}
	for _, tc := range []struct {
		name, askExe, require, token string
	}{
		{"wrong exe", `C:\\Other\\ByFTP.exe`, "force", token},
		{"wrong require", exe, "prefer", token},
		{"short token", exe, "force", "abc"},
		{"nonhex token", exe, "force", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"},
	} {
		if validAskpassInvocation(exe, tc.askExe, tc.require, tc.token) {
			t.Fatalf("%s unexpectedly validated", tc.name)
		}
	}
}
