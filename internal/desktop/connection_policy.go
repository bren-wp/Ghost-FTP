package desktop

import (
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func connectionTimeoutDuration(settings model.Settings) time.Duration {
	seconds := settings.ConnectionTimeoutSeconds
	if seconds < 5 {
		seconds = 15
	}
	if seconds > 60 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}
