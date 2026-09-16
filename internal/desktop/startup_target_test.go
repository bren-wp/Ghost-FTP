package desktop

import "testing"

func TestStartupTargetIsConsumedOnce(t *testing.T) {
	SetStartupTarget(StartupTarget{
		Protocol: "sftp",
		Host:     "example.invalid",
		Port:     22,
		Username: "alice",
		Path:     "/home/alice",
	})

	got, ok := takeStartupTarget()
	if !ok {
		t.Fatal("expected startup target")
	}
	if got.Protocol != "sftp" || got.Host != "example.invalid" || got.Port != 22 || got.Username != "alice" || got.Path != "/home/alice" {
		t.Fatalf("unexpected startup target: %#v", got)
	}
	if _, ok := takeStartupTarget(); ok {
		t.Fatal("startup target must be one-shot")
	}
}

func TestStartupTargetDefaultPorts(t *testing.T) {
	cases := []struct {
		protocol string
		want     int
	}{
		{protocol: "ftp", want: 21},
		{protocol: "ftps", want: 21},
		{protocol: "sftp", want: 22},
	}
	for _, tc := range cases {
		if got := startupTargetPort(StartupTarget{Protocol: tc.protocol}); got != tc.want {
			t.Fatalf("startupTargetPort(%q) = %d, want %d", tc.protocol, got, tc.want)
		}
	}
	if got := startupTargetPort(StartupTarget{Protocol: "sftp", Port: 2222}); got != 2222 {
		t.Fatalf("explicit port = %d, want 2222", got)
	}
}
