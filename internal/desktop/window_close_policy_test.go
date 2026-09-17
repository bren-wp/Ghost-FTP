package desktop

import "testing"

func TestDeriveWindowCloseAction(t *testing.T) {
	tests := []struct {
		name                string
		profileMutationBusy bool
		connected           bool
		connectionBusy      bool
		activeTransfers     bool
		want                windowCloseAction
	}{
		{name: "idle closes", want: windowCloseNow},
		{name: "connected confirms", connected: true, want: windowCloseConfirmActiveWork},
		{name: "connecting confirms", connectionBusy: true, want: windowCloseConfirmActiveWork},
		{name: "active transfers confirm", activeTransfers: true, want: windowCloseConfirmActiveWork},
		{name: "profile mutation defers", profileMutationBusy: true, want: windowCloseWaitForProfileMutation},
		{
			name:                "profile mutation wins over active work",
			profileMutationBusy: true,
			connected:           true,
			connectionBusy:      true,
			activeTransfers:     true,
			want:                windowCloseWaitForProfileMutation,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := deriveWindowCloseAction(test.profileMutationBusy, test.connected, test.connectionBusy, test.activeTransfers)
			if got != test.want {
				t.Fatalf("close action = %d, want %d", got, test.want)
			}
		})
	}
}
