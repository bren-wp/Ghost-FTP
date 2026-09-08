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
	if Light.Panel != (RGB{0xF6, 0xF8, 0xFB}) || Light.Text != (RGB{0x20, 0x25, 0x2B}) {
		t.Fatal("canonical Light panel/text contract changed unexpectedly")
	}
}
