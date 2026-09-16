//go:build windows

package desktop

import "testing"

func TestStartupTargetHandoffRoundTrip(t *testing.T) {
	// Keep this test isolated from any startup target left by another test.
	_, _ = takeStartupTarget()

	stop := StartStartupTargetReceiver()
	defer stop()

	raw := "ghostftp://connect?protocol=sftp&host=example.com&port=2222&username=alice&path=%2Fhome%2Falice"
	if !ForwardStartupTarget(raw) {
		t.Fatal("expected local Windows startup target handoff to succeed")
	}

	target, ok := takeStartupTarget()
	if !ok {
		t.Fatal("expected forwarded startup target")
	}
	if target.Protocol != "sftp" || target.Host != "example.com" || target.Port != 2222 || target.Username != "alice" || target.Path != "/home/alice" {
		t.Fatalf("unexpected forwarded target: %#v", target)
	}
}

func TestStartupTargetHandoffRejectsSensitivePayload(t *testing.T) {
	if ForwardStartupTarget("ghostftp://connect?protocol=sftp&host=example.com&password=secret") {
		t.Fatal("sensitive startup target must be rejected before Windows handoff")
	}
}

func TestCanApplyStartupTargetDefersUnsafeStateTransitions(t *testing.T) {
	cases := []struct {
		name string
		app  *app
		want bool
	}{
		{name: "ready", app: &app{}, want: true},
		{name: "connected", app: &app{connected: true}, want: false},
		{name: "connection busy", app: &app{connectionBusy: true}, want: false},
		{name: "profile mutation busy", app: &app{profileMutationBusy: true}, want: false},
		{name: "nil app", app: nil, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.app.canApplyStartupTarget(); got != tc.want {
				t.Fatalf("canApplyStartupTarget() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSiteManagerBlocksStartupTargetUntilStateIsUnmapped(t *testing.T) {
	a := &app{}
	state := &siteManagerState{parent: a, closed: true}
	const testWindow = uintptr(0x7f02)
	siteManagerStates.Store(testWindow, state)

	if !siteManagerBlocksStartupTarget(a) {
		t.Fatal("mapped Site Manager must block startup target even after WM_DESTROY marks it closed")
	}
	if a.canApplyStartupTarget() {
		t.Fatal("startup target must remain pending through post-close profile restoration")
	}

	siteManagerStates.Delete(testWindow)
	if siteManagerBlocksStartupTarget(a) {
		t.Fatal("unmapped Site Manager must no longer block startup target")
	}
	if !a.canApplyStartupTarget() {
		t.Fatal("startup target should become applicable after Site Manager teardown completes")
	}
}
