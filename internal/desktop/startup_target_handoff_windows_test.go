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

func TestCanApplyStartupTargetDefersWhileSiteManagerIsOpen(t *testing.T) {
	a := &app{}
	state := &siteManagerState{parent: a}
	const testWindow = uintptr(0x7f01)
	siteManagerStates.Store(testWindow, state)
	defer siteManagerStates.Delete(testWindow)

	if !siteManagerOpenForApp(a) {
		t.Fatal("expected Site Manager state to be detected for the parent app")
	}
	if a.canApplyStartupTarget() {
		t.Fatal("startup target must stay pending while Site Manager is open")
	}
}

func TestHasStartupTargetDoesNotConsumeOneShotTarget(t *testing.T) {
	_, _ = takeStartupTarget()
	want := StartupTarget{Protocol: "sftp", Host: "example.com", Port: 22, Username: "alice", Path: "/home/alice"}
	SetStartupTarget(want)
	if !hasStartupTarget() {
		t.Fatal("expected pending startup target")
	}
	got, ok := takeStartupTarget()
	if !ok {
		t.Fatal("presence check must not consume startup target")
	}
	if got != want {
		t.Fatalf("unexpected target after presence check: %#v", got)
	}
}
