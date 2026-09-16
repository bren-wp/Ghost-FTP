package main

import (
	"reflect"
	"testing"
)

func TestProtocolRollbackRemovesOnlyKeysCreatedByInstaller(t *testing.T) {
	snapshot := registrySnapshot{keys: []registryKeySnapshot{
		{key: browserProtocolKey, existed: true},
		{key: browserProtocolIconKey, existed: false},
		{key: browserProtocolCommandKey, existed: false},
	}}

	got := snapshot.protocolKeysToRemove()
	want := []string{browserProtocolCommandKey, browserProtocolIconKey}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("protocolKeysToRemove() = %#v, want %#v", got, want)
	}
}

func TestProtocolRollbackPreservesAllPreExistingKeys(t *testing.T) {
	snapshot := registrySnapshot{keys: []registryKeySnapshot{
		{key: browserProtocolKey, existed: true},
		{key: browserProtocolIconKey, existed: true},
		{key: browserProtocolCommandKey, existed: true},
	}}
	if got := snapshot.protocolKeysToRemove(); len(got) != 0 {
		t.Fatalf("pre-existing protocol keys must be preserved, got removals %#v", got)
	}
}

func TestProtocolRollbackDeletesNewTreeLeafToRoot(t *testing.T) {
	snapshot := registrySnapshot{keys: []registryKeySnapshot{
		{key: browserProtocolKey, existed: false},
		{key: browserProtocolIconKey, existed: false},
		{key: browserProtocolCommandKey, existed: false},
	}}
	got := snapshot.protocolKeysToRemove()
	want := []string{browserProtocolCommandKey, browserProtocolIconKey, browserProtocolKey}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("new protocol tree removal order = %#v, want %#v", got, want)
	}
}

func TestProtocolRollbackFailsClosedWithoutKeySnapshot(t *testing.T) {
	var snapshot registrySnapshot
	if got := snapshot.protocolKeysToRemove(); len(got) != 0 {
		t.Fatalf("untracked keys must not be deleted, got %#v", got)
	}
}
