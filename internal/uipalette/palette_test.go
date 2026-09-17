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
	if Dark.Panel != (RGB{0x12, 0x18, 0x24}) || Dark.Text != (RGB{0xF2, 0xF5, 0xFA}) {
		t.Fatal("canonical Dark panel/text contract changed unexpectedly")
	}
	if Light.Panel != (RGB{0xF6, 0xF8, 0xFB}) || Light.Text != (RGB{0x17, 0x20, 0x33}) {
		t.Fatal("canonical Light panel/text contract changed unexpectedly")
	}
}

func TestGhostGoldAccentContract(t *testing.T) {
	if Dark.Accent != (RGB{0xF6, 0xC4, 0x45}) || Dark.AccentStrong != (RGB{0xFF, 0xD7, 0x68}) {
		t.Fatal("dark Ghost Gold accent contract changed unexpectedly")
	}
	if Light.Accent != (RGB{0xA6, 0x65, 0x00}) || Light.AccentStrong != (RGB{0x87, 0x51, 0x00}) {
		t.Fatal("light Ghost Gold accent contract changed unexpectedly")
	}
	if Dark.OnAccent == Dark.Accent || Light.OnAccent == Light.Accent {
		t.Fatal("filled accent controls lost their dedicated foreground role")
	}
	for name, accent := range map[string]RGB{"dark": Dark.Accent, "light": Light.Accent} {
		if !(accent.R > accent.G && accent.G > accent.B) {
			t.Fatalf("%s accent no longer belongs to the warm Ghost Gold family: %#v", name, accent)
		}
	}
}
