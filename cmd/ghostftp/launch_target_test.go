package main

import "testing"

func TestParseDesktopLaunchTarget(t *testing.T) {
	target, err := parseDesktopLaunchTarget("ghostftp://connect?protocol=sftp&host=example.com&port=22&username=alice&path=%2Fhome%2Falice")
	if err != nil {
		t.Fatalf("parseDesktopLaunchTarget returned error: %v", err)
	}
	if target.Protocol != "sftp" || target.Host != "example.com" || target.Port != 22 || target.Username != "alice" || target.Path != "/home/alice" {
		t.Fatalf("unexpected target: %#v", target)
	}
}

func TestParseDesktopLaunchTargetAcceptsLegacyOpenRoute(t *testing.T) {
	target, err := parseDesktopLaunchTarget("ghostftp://open?protocol=ftps&host=example.com")
	if err != nil {
		t.Fatalf("legacy open route returned error: %v", err)
	}
	if target.Protocol != "ftps" || target.Host != "example.com" || target.Path != "/" {
		t.Fatalf("unexpected target: %#v", target)
	}
}

func TestParseDesktopLaunchTargetRejectsSensitiveAndAmbiguousFields(t *testing.T) {
	cases := []string{
		"ghostftp://connect?protocol=sftp&host=example.com&password=secret",
		"ghostftp://connect?protocol=sftp&host=example.com&token=secret",
		"ghostftp://connect?protocol=sftp&protocol=ftp&host=example.com",
		"ghostftp://connect?protocol=sftp&host=one.example&host=two.example",
		"ghostftp://connect?protocol=https&host=example.com",
		"ghostftp://connect?protocol=sftp&host=example.com&port=70000",
		"ghostftp://connect?protocol=sftp&host=example.com#secret",
		"ghostftp://user:pass@connect?protocol=sftp&host=example.com",
		"ghostftp://other?protocol=sftp&host=example.com",
		"ghostftp://connect?protocol=sftp&host=bad%00host",
	}

	for _, raw := range cases {
		if _, err := parseDesktopLaunchTarget(raw); err == nil {
			t.Fatalf("expected rejection for %q", raw)
		}
	}
}

func TestDesktopLaunchArgument(t *testing.T) {
	target, found, err := desktopLaunchArgument([]string{"--portable", "ghostftp://connect?protocol=ftp&host=ftp.example.com"})
	if err != nil || !found {
		t.Fatalf("expected launch argument, found=%v err=%v", found, err)
	}
	if target.Protocol != "ftp" || target.Host != "ftp.example.com" || target.Path != "/" {
		t.Fatalf("unexpected target: %#v", target)
	}
}

func TestDesktopLaunchArgumentIgnoresUnrelatedArguments(t *testing.T) {
	if _, found, err := desktopLaunchArgument([]string{"--portable", "--uninstall"}); err != nil || found {
		t.Fatalf("unexpected launch target, found=%v err=%v", found, err)
	}
}
