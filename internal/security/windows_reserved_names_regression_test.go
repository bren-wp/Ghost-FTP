package security

import "testing"

func TestValidateNameRejectsSuperscriptWindowsDeviceNames(t *testing.T) {
	invalid := []string{
		"COM¹", "COM².txt", "com³.log",
		"LPT¹", "LPT².doc", "lpt³.bin",
	}
	for _, name := range invalid {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) unexpectedly accepted a Windows device name", name)
		}
	}

	valid := []string{"COM0", "COM10", "LPT0.txt", "LPT10.txt", "COM⁴.txt"}
	for _, name := range valid {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) unexpectedly rejected a normal filename: %v", name, err)
		}
	}
}
