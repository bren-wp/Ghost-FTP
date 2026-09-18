package uipalette

import "testing"

func TestCanonicalPalettesRemainDistinct(t *testing.T) {
	if Light == Dark {
		t.Fatal("Light and Dark palettes unexpectedly match")
	}
	if Light.Panel == Dark.Panel || Light.Text == Dark.Text || Light.Selection == Dark.Selection {
		t.Fatal("critical Light/Dark roles are not visually distinct")
	}
}

func TestClassicLightAvoidsPureWhitePrimarySurfaces(t *testing.T) {
	pureWhite := RGB{0xFF, 0xFF, 0xFF}
	for name, value := range map[string]RGB{
		"window": Light.Window,
		"panel":  Light.Panel,
		"list":   Light.List,
	} {
		if value == pureWhite {
			t.Fatalf("Classic Light %s surface regressed to pure white", name)
		}
	}
}

func TestMaintainedModalRolesMatchDesktopContract(t *testing.T) {
	if Dark.Panel != (RGB{0x0D, 0x17, 0x24}) || Dark.Text != (RGB{0xE5, 0xE7, 0xEB}) {
		t.Fatal("canonical Dark panel/text contract changed unexpectedly")
	}
	if Light.Panel != (RGB{0xF6, 0xF8, 0xFB}) || Light.Text != (RGB{0x17, 0x20, 0x33}) {
		t.Fatal("canonical Light panel/text contract changed unexpectedly")
	}
}

func TestGhostCyanBlueAccentContract(t *testing.T) {
	if Dark.Accent != (RGB{0x00, 0xE5, 0xFF}) || Dark.AccentStrong != (RGB{0x3B, 0x82, 0xF6}) {
		t.Fatal("dark Ghost cyan/blue accent contract changed unexpectedly")
	}
	if Light.Accent != (RGB{0x00, 0x82, 0xB8}) || Light.AccentStrong != (RGB{0x25, 0x63, 0xEB}) {
		t.Fatal("light Ghost cyan/blue accent contract changed unexpectedly")
	}
	if Dark.OnAccent == Dark.Accent || Light.OnAccent == Light.Accent {
		t.Fatal("filled accent controls lost their dedicated foreground role")
	}
	if Dark.Accent.B < Dark.Accent.G || Dark.Accent.G <= Dark.Accent.R {
		t.Fatalf("dark accent no longer belongs to the Ghost cyan family: %#v", Dark.Accent)
	}
}
