package desktop

import "testing"

func TestParseStartupTarget(t *testing.T) {
	target, err := ParseStartupTarget("ghostftp://connect?protocol=sftp&host=example.com&port=2222&username=alice&path=%2Fhome%2Falice")
	if err != nil {
		t.Fatalf("ParseStartupTarget returned error: %v", err)
	}
	if target.Protocol != "sftp" || target.Host != "example.com" || target.Port != 2222 || target.Username != "alice" || target.Path != "/home/alice" {
		t.Fatalf("unexpected startup target: %#v", target)
	}
}

func TestParseStartupTargetRejectsSensitiveAndAmbiguousInput(t *testing.T) {
	cases := []string{
		"ghostftp://connect?protocol=sftp&host=example.com&password=secret",
		"ghostftp://connect?protocol=sftp&host=example.com&passphrase=secret",
		"ghostftp://connect?protocol=sftp&host=example.com&token=secret",
		"ghostftp://connect?protocol=sftp&protocol=ftp&host=example.com",
		"ghostftp://connect?protocol=sftp&host=one.example&host=two.example",
		"ghostftp://connect?protocol=https&host=example.com",
		"ghostftp://connect?protocol=sftp&host=example.com&port=022",
		"ghostftp://connect?protocol=sftp&host=example.com&port=70000",
		"ghostftp://connect?protocol=sftp&host=example.com#secret",
		"ghostftp://user:pass@connect?protocol=sftp&host=example.com",
		"ghostftp://connect:99?protocol=sftp&host=example.com",
		"ghostftp://connect/extra?protocol=sftp&host=example.com",
		"ghostftp://connect?protocol=sftp&host=bad%00host",
		"ghostftp://connect?protocol=sftp&host=example.com&username=bad%09name",
		"ghostftp://connect?protocol=sftp&host=example.com\n",
	}

	for _, raw := range cases {
		if _, err := ParseStartupTarget(raw); err == nil {
			t.Fatalf("expected rejection for %q", raw)
		}
	}
}

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
