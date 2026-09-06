package desktop

import (
	"testing"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestConnectionTimeoutDurationUsesValidatedSetting(t *testing.T) {
	cases := []struct {
		seconds int
		want    time.Duration
	}{
		{0, 15 * time.Second},
		{5, 5 * time.Second},
		{27, 27 * time.Second},
		{60, 60 * time.Second},
		{999, 60 * time.Second},
	}
	for _, tc := range cases {
		got := connectionTimeoutDuration(model.Settings{ConnectionTimeoutSeconds: tc.seconds})
		if got != tc.want {
			t.Fatalf("seconds=%d got=%s want=%s", tc.seconds, got, tc.want)
		}
	}
}
