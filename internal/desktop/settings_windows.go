//go:build windows

package desktop

import (
	"strconv"
	"strings"

	"brendigo.com/byftp/internal/brand"
	"brendigo.com/byftp/internal/platform"
)

func (a *app) loadSettings() {
	a.goSafe(func() {
		settings, err := a.engine.Settings()
		a.dispatch(func() {
			if err != nil {
				a.setStatus(a.userMessage(err, "settings.load_failed"))
				return
			}
			a.settings = settings
			a.applyLanguage(settings.Language)
			a.updateActionControls()
		})
	})
}

func (a *app) promptNumber(instructionKey string, current, min, max int) (int, bool) {
	title := a.tr("settings.title")
	value, ok := platform.PromptDialogWithLabels(
		title,
		a.tr(instructionKey),
		strconv.Itoa(current),
		okLabel(a.languageCode()),
		a.tr("common.cancel"),
	)
	if !ok {
		return current, false
	}
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number < min || number > max {
		platform.ErrorDialog(title, a.tr("settings.invalid_value"), a.tr("settings.enter_range", min, max))
		return current, false
	}
	return number, true
}

func (a *app) openSettings() {
	if a.connectionBusy {
		return
	}
	settings := a.settings
	if settings.Parallelism < 1 {
		settings.Parallelism = 2
	}
	if settings.RetryDelaySeconds < 1 {
		settings.RetryDelaySeconds = 3
	}
	if settings.ConnectionTimeoutSeconds < 5 {
		settings.ConnectionTimeoutSeconds = 15
	}

	parallel, ok := a.promptNumber("settings.parallel", settings.Parallelism, 1, 8)
	if !ok {
		return
	}
	settings.Parallelism = parallel

	connectTimeout, ok := a.promptNumber("settings.timeout", settings.ConnectionTimeoutSeconds, 5, 60)
	if !ok {
		return
	}
	settings.ConnectionTimeoutSeconds = connectTimeout

	retries, ok := a.promptNumber("settings.retries", settings.AutoRetryCount, 0, 3)
	if !ok {
		return
	}
	settings.AutoRetryCount = retries
	if retries > 0 {
		delay, ok := a.promptNumber("settings.retry_delay", settings.RetryDelaySeconds, 1, 30)
		if !ok {
			return
		}
		settings.RetryDelaySeconds = delay
	}

	title := a.tr("settings.title")
	settings.BackupBeforeOverwrite = platform.ConfirmDialog(title, a.tr("settings.backup_title"), a.tr("settings.backup_body"))
	settings.SkipExisting = platform.ConfirmDialog(title, a.tr("settings.skip_title"), a.tr("settings.skip_body"))
	settings.ConfirmDelete = platform.ConfirmDialog(title, a.tr("settings.confirm_delete_title"), a.tr("settings.confirm_delete_body"))

	a.goSafe(func() {
		saved, err := a.engine.SetSettings(settings)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog(title, a.tr("settings.save_failed"), a.userMessage(err, "settings.save_failed_body"))
				return
			}
			a.settings = saved
			a.applyLanguage(saved.Language)
			a.setStatus(a.tr("settings.saved", saved.Parallelism, saved.ConnectionTimeoutSeconds, retrySummary(a, saved), overwriteSummary(a, saved)))
			a.updateActionControls()
		})
	})
}

func (a *app) openAbout() {
	// Keep company/brand metadata out of the working FTP surface. The About
	// task dialog is the single deliberate brand card: product identity first,
	// then privacy/security promise, then developer and support metadata.
	content := a.tr("about.heading") + "\n\n" +
		"────────────────────────\n" +
		a.tr("about.body", brand.Website, brand.Support) + "\n\n" +
		"FTP • FTPS • SFTP  ·  " + brand.ProductName + " " + a.version
	platform.InfoDialog(
		brand.ProductName+" — "+a.tr("about.title"),
		brand.ProductFull+" "+a.version,
		content,
	)
}
