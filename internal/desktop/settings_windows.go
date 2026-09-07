//go:build windows

package desktop

import (
	"strconv"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

func (a *app) loadSettings() {
	a.goSafe(func() {
		settings, err := a.engine.Settings()
		a.dispatch(func() {
			if err != nil {
				a.setStatus(a.userMessage(err, "settings.load_failed"))
				return
			}
			previousLanguage := a.languageCode()
			a.settings = settings
			// The controls were already created in the canonical startup locale.
			// Reapplying the same locale used to refill every list, relayout every
			// control and erase the whole client immediately after ShowWindow,
			// producing a visible startup flash. Only rebuild localized UI when the
			// persisted locale actually differs.
			if a.languageCode() != previousLanguage {
				a.applyLanguage(settings.Language)
			}
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

func conflictPolicyIndex(settings model.Settings) int {
	switch settings.ConflictPolicy {
	case model.ConflictPolicySkip:
		return 0
	case model.ConflictPolicyReplace:
		return 1
	case model.ConflictPolicyReplaceBackup:
		return 2
	}
	if settings.SkipExisting {
		return 0
	}
	if settings.BackupBeforeOverwrite {
		return 2
	}
	return 1
}

func applyConflictPolicySelection(settings *model.Settings, index int) {
	if settings == nil {
		return
	}
	switch index {
	case 0:
		settings.ConflictPolicy = model.ConflictPolicySkip
		settings.SkipExisting = true
		// Legacy consumers historically saw backup enabled together with skip.
		// The backup flag has no effect because skip performs no overwrite.
		settings.BackupBeforeOverwrite = true
	case 1:
		settings.ConflictPolicy = model.ConflictPolicyReplace
		settings.SkipExisting = false
		settings.BackupBeforeOverwrite = false
	default:
		settings.ConflictPolicy = model.ConflictPolicyReplaceBackup
		settings.SkipExisting = false
		settings.BackupBeforeOverwrite = true
	}
}

func (a *app) promptAppearance(settings model.Settings) (model.Settings, bool) {
	words := appearanceText(a.languageCode())
	index, ok := platform.SelectOptionDialog(
		a.tr("settings.title"),
		words.Hint,
		brand.ProductName+" · "+words.Title,
		okLabel(a.languageCode()),
		a.tr("common.cancel"),
		[]string{words.Dark, words.Light},
		appearanceIndex(settings.Appearance),
	)
	if !ok {
		return settings, false
	}
	applyAppearanceSelection(&settings, index)
	return settings, true
}

func (a *app) promptConflictPolicy(settings model.Settings) (model.Settings, bool) {
	title := a.tr("settings.title")
	options := []string{
		a.tr("settings.skip_existing"),
		a.tr("settings.overwrite"),
		a.tr("settings.overwrite") + " + " + a.tr("settings.backup_title"),
	}
	instruction := a.tr("settings.skip_body") + "\n" + a.tr("settings.backup_body")
	index, ok := platform.SelectOptionDialog(
		title,
		instruction,
		brand.ProductName+" · "+a.tr("settings.title"),
		okLabel(a.languageCode()),
		a.tr("common.cancel"),
		options,
		conflictPolicyIndex(settings),
	)
	if !ok {
		return settings, false
	}
	applyConflictPolicySelection(&settings, index)
	return settings, true
}

func (a *app) openSettings() {
	if a.connectionBusy {
		return
	}
	settings := a.settings
	if settings.Appearance == "" {
		settings.Appearance = model.AppearanceLight
	}
	if settings.Parallelism < 1 {
		settings.Parallelism = 2
	}
	if settings.RetryDelaySeconds < 1 {
		settings.RetryDelaySeconds = 3
	}
	if settings.ConnectionTimeoutSeconds < 5 {
		settings.ConnectionTimeoutSeconds = 15
	}

	var ok bool
	settings, ok = a.promptAppearance(settings)
	if !ok {
		return
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

	settings, ok = a.promptConflictPolicy(settings)
	if !ok {
		return
	}

	title := a.tr("settings.title")
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
			status := a.tr("settings.saved", saved.Parallelism, saved.ConnectionTimeoutSeconds, retrySummary(a, saved), overwriteSummary(a, saved))
			if isDarkAppearance(saved.Appearance) != activeThemeIsDark() {
				status += " · " + appearanceText(saved.Language).Hint
			}
			a.setStatus(status)
			a.updateActionControls()
		})
	})
}

func (a *app) openAbout() {
	// About is an application-owned card so its Light/Dark appearance does not
	// drift with the Windows TaskDialog theme. Runtime destinations are limited
	// to official BRENDIGO LTD web properties; repository links remain in docs.
	// Localization catalogs may legitimately use the internal GhostFTP identity
	// in non-public surfaces, but About always renders the public Ghost FTP name.
	aboutBody := strings.ReplaceAll(a.tr("about.body", brand.Website, brand.Support), "GhostFTP", brand.ProductName)
	body := aboutBody + "\n\n" +
		brand.Publisher + "\n" +
		"FTP • FTPS • SFTP  ·  " + brand.ProductName + " " + a.version
	platform.InfoCardDialog(
		brand.ProductName+" — "+a.tr("about.title"),
		a.tr("about.heading"),
		body,
		okLabel(a.languageCode()),
	)
}
