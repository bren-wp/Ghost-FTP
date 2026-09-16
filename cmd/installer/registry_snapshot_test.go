package main

import "testing"

func TestProtocolKeyOriginallyExisted(t *testing.T) {
	snapshot := registrySnapshot{keys: []registryKeySnapshot{
		{key: browserProtocolKey, existed: true},
		{key: browserProtocolIconKey, existed: false},
		{key: browserProtocolShellKey, existed: true},
		{key: browserProtocolOpenKey, existed: false},
		{key: browserProtocolCommandKey, existed: false},
	}}

	if !snapshot.protocolKeyOriginallyExisted(browserProtocolKey) {
		t.Fatal("pre-existing protocol root must be preserved")
	}
	if !snapshot.protocolKeyOriginallyExisted(browserProtocolShellKey) {
		t.Fatal("pre-existing shell key must be preserved even without tracked values")
	}
	if snapshot.protocolKeyOriginallyExisted(browserProtocolCommandKey) {
		t.Fatal("installer-created command key must remain eligible for rollback removal")
	}
	if snapshot.protocolKeyOriginallyExisted(`Software\\Classes\\ghostftp\\missing`) {
		t.Fatal("unknown key must not be treated as pre-existing")
	}
}
