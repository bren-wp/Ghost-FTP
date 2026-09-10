package config

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestBandwidthDefaultsAreUnlimited(t *testing.T) {
	settings := DefaultSettings()
	if settings.UploadLimitKiBPerSecond != 0 || settings.DownloadLimitKiBPerSecond != 0 {
		t.Fatalf("fresh settings must default to unlimited bandwidth: %+v", settings)
	}
	if err := validateSettings(settings); err != nil {
		t.Fatalf("default settings must validate: %v", err)
	}
}

func TestBandwidthValidationAcceptsIndependentDirectionalLimits(t *testing.T) {
	settings := DefaultSettings()
	settings.UploadLimitKiBPerSecond = 2048
	settings.DownloadLimitKiBPerSecond = 8192
	if err := validateSettings(settings); err != nil {
		t.Fatalf("valid directional limits rejected: %v", err)
	}
}

func TestBandwidthValidationRejectsOutOfRangeValues(t *testing.T) {
	for _, tc := range []model.Settings{
		func() model.Settings { v := DefaultSettings(); v.UploadLimitKiBPerSecond = -1; return v }(),
		func() model.Settings { v := DefaultSettings(); v.DownloadLimitKiBPerSecond = MaxBandwidthLimitKiBPerSecond + 1; return v }(),
	} {
		if err := validateSettings(tc); err == nil {
			t.Fatalf("out-of-range bandwidth limit unexpectedly accepted: %+v", tc)
		}
	}
}

func TestPersistedInvalidBandwidthFallsBackToUnlimited(t *testing.T) {
	settings := DefaultSettings()
	settings.UploadLimitKiBPerSecond = -50
	settings.DownloadLimitKiBPerSecond = MaxBandwidthLimitKiBPerSecond + 1
	normalized := normalizeSettings(settings)
	if normalized.UploadLimitKiBPerSecond != 0 || normalized.DownloadLimitKiBPerSecond != 0 {
		t.Fatalf("invalid persisted bandwidth state must migrate to unlimited: %+v", normalized)
	}
}
