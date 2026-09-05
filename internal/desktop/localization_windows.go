//go:build windows

package desktop

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
)

const (
	idLanguage     = 96
	cbResetContent = 0x014B
	lvmSetColumnW  = lvmFirst + 96
)

func (a *app) languageCode() string {
	if a == nil {
		return i18n.DefaultLanguage
	}
	return i18n.Normalize(a.settings.Language)
}

func (a *app) tr(key string, args ...any) string {
	return i18n.T(a.languageCode(), key, args...)
}

func (a *app) userMessage(err error, fallbackKey string) string {
	fallback := ""
	if fallbackKey != "" {
		fallback = a.tr(fallbackKey)
	}
	return usererror.MessageFor(a.languageCode(), err, fallback)
}

// workspaceSubtitle deliberately excludes company branding from the working
// FTP surface. Brand/company metadata is presented in the dedicated About card
// instead, keeping the primary workspace focused on files and connections.
func (a *app) workspaceSubtitle() string {
	text := a.tr("app.subtitle")
	if separator := strings.LastIndex(text, "·"); separator >= 0 {
		if strings.EqualFold(strings.TrimSpace(text[separator+len("·"):]), brand.Company) {
			return strings.TrimSpace(text[:separator])
		}
	}
	return text
}

func (a *app) setButtonLabel(hwnd uintptr, label string) {
	if hwnd == 0 {
		return
	}
	setText(hwnd, label)
	if visual, ok := a.buttons[hwnd]; ok {
		visual.Label = label
		a.buttons[hwnd] = visual
	}
	invalidateRect.Call(hwnd, 0, 1)
}

func (a *app) populateLanguageCombo() {
	if a.languageCombo == 0 {
		return
	}
	sendMessageW.Call(a.languageCombo, cbResetContent, 0, 0)
	selected := 0
	current := a.languageCode()
	for index, language := range i18n.Languages() {
		label := language.NativeName
		if language.EnglishName != language.NativeName {
			label = language.NativeName + " · " + language.Code
		}
		sendMessageW.Call(a.languageCombo, cbAddString, 0, uintptr(unsafe.Pointer(wstr(label))))
		if language.Code == current {
			selected = index
		}
	}
	sendMessageW.Call(a.languageCombo, cbSetCurSel, uintptr(selected), 0)
}

func protocolLabel(language, protocol string) string {
	words := map[string][2]string{
		"en": {"explicit", "implicit"}, "hr": {"eksplicitni", "implicitni"}, "de": {"explizit", "implizit"},
		"fr": {"explicite", "implicite"}, "es": {"explícito", "implícito"}, "tr": {"açık", "örtük"},
		"el": {"ρητό", "έμμεσο"}, "pt": {"explícito", "implícito"}, "zh": {"显式", "隐式"},
		"ru": {"явный", "неявный"}, "hi": {"स्पष्ट", "निहित"}, "ja": {"明示", "暗黙"},
	}
	pair := words[i18n.Normalize(language)]
	switch protocol {
	case "ftps":
		return fmt.Sprintf("FTPS (%s)", pair[0])
	case "ftpsi":
		return fmt.Sprintf("FTPS (%s)", pair[1])
	case "sftp":
		return "SFTP"
	default:
		return "FTP"
	}
}

func (a *app) reloadProtocolLabels() {
	if a.protocol == 0 {
		return
	}
	selected := selectedComboIndex(a.protocol)
	if selected < 0 || selected >= len(protocolSpecs) {
		selected = 0
	}
	sendMessageW.Call(a.protocol, cbResetContent, 0, 0)
	for _, spec := range protocolSpecs {
		label := protocolLabel(a.languageCode(), spec.Value)
		sendMessageW.Call(a.protocol, cbAddString, 0, uintptr(unsafe.Pointer(wstr(label))))
	}
	sendMessageW.Call(a.protocol, cbSetCurSel, uintptr(selected), 0)
}

func selectedComboIndex(combo uintptr) int {
	if combo == 0 {
		return -1
	}
	index, _, _ := sendMessageW.Call(combo, cbGetCurSel, 0, 0)
	if int32(index) < 0 {
		return -1
	}
	return int(index)
}

func (a *app) reloadProfileLabels() {
	if a.profilesCombo == 0 {
		return
	}
	sendMessageW.Call(a.profilesCombo, cbResetContent, 0, 0)
	sendMessageW.Call(a.profilesCombo, cbAddString, 0, uintptr(unsafe.Pointer(wstr(a.tr("profile.quick")))))
	selected := 0
	for index, profile := range a.profiles {
		sendMessageW.Call(a.profilesCombo, cbAddString, 0, uintptr(unsafe.Pointer(wstr(profile.Name))))
		if profile.ID == a.selectedProfileID {
			selected = index + 1
		}
	}
	sendMessageW.Call(a.profilesCombo, cbSetCurSel, uintptr(selected), 0)
}

func (a *app) setColumnTitle(list uintptr, index int, title string) {
	if list == 0 || index < 0 {
		return
	}
	text := syscall.StringToUTF16(title)
	column := lvColumn{Mask: lvcfText, Text: &text[0]}
	sendMessageW.Call(list, lvmSetColumnW, uintptr(index), uintptr(unsafe.Pointer(&column)))
}

func (a *app) applyColumnLanguage() {
	for _, list := range []uintptr{a.localList, a.remoteList} {
		a.setColumnTitle(list, 0, a.tr("column.name"))
		a.setColumnTitle(list, 1, a.tr("column.type"))
		a.setColumnTitle(list, 2, a.tr("column.size"))
		a.setColumnTitle(list, 3, a.tr("column.modified"))
	}
	for index, key := range []string{"column.direction", "column.local", "column.remote", "column.status", "column.progress"} {
		a.setColumnTitle(a.transferList, index, a.tr(key))
	}
}

func (a *app) applyLanguage(code string) {
	if a == nil {
		return
	}
	a.settings.Language = i18n.Normalize(code)
	a.populateLanguageCombo()
	setText(a.brandSubtitle, a.workspaceSubtitle())
	setText(a.statusVersion, brand.ProductName+" "+a.version)
	if a.connected {
		setText(a.connectionBadge, a.tr("badge.connected"))
	} else {
		setText(a.connectionBadge, a.tr("badge.disconnected"))
	}
	a.setButtonLabel(a.saveProfile, a.tr("profile.save"))
	a.setButtonLabel(a.removeProfile, a.tr("profile.delete"))
	a.setButtonLabel(a.settingsBtn, a.tr("common.settings"))
	a.setButtonLabel(a.aboutBtn, a.tr("common.about"))
	a.setButtonLabel(a.connect, a.tr("common.connect"))
	a.setButtonLabel(a.disconnect, a.tr("common.disconnect"))
	a.setButtonLabel(a.chooseKey, a.tr("auth.private_key"))
	setText(a.sectionLocal, a.tr("section.local"))
	setText(a.sectionRemote, a.tr("section.remote"))
	setText(a.sectionTransfers, a.tr("section.transfers"))
	for _, pair := range []struct {
		hwnd uintptr
		key  string
	}{
		{a.localUp, "common.up"}, {a.localChoose, "common.folder"}, {a.localRefresh, "common.refresh"},
		{a.localMkdir, "common.new_folder"}, {a.localRename, "common.rename"}, {a.localDelete, "common.delete"},
		{a.remoteUp, "common.up"}, {a.remoteRefresh, "common.refresh"}, {a.remoteMkdir, "common.new_folder"},
		{a.remoteRename, "common.rename"}, {a.remoteDelete, "common.delete"}, {a.remoteChmod, "common.permissions"},
		{a.upload, "transfer.upload"}, {a.download, "transfer.download"}, {a.pauseQueue, "transfer.pause"},
		{a.resumeQueue, "transfer.resume"}, {a.cancelJob, "common.cancel"}, {a.retryJob, "transfer.retry"}, {a.clearQueue, "transfer.clear"},
	} {
		a.setButtonLabel(pair.hwnd, a.tr(pair.key))
	}
	cue(a.host, a.tr("cue.host"))
	cue(a.user, a.tr("cue.user"))
	cue(a.pass, a.tr("cue.password"))
	cue(a.keyPath, a.tr("cue.private_key"))
	cue(a.passphrase, a.tr("cue.passphrase"))
	a.reloadProtocolLabels()
	a.reloadProfileLabels()
	a.installMainMenu()
	a.applyColumnLanguage()
	a.fillItemList(a.localList, a.localItems)
	a.fillItemList(a.remoteList, a.remoteItems)
	a.fillTransferList(a.transferList, a.transferJobs)
	a.updateTransferSummary()
	var client rect
	if r, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); r != 0 {
		a.layout(int(client.Right-client.Left), int(client.Bottom-client.Top))
	}
	invalidateRect.Call(a.hwnd, 0, 1)
}

func (a *app) changeLanguageFromUI() {
	index := selectedComboIndex(a.languageCombo)
	languages := i18n.Languages()
	if index < 0 || index >= len(languages) {
		return
	}
	old := a.settings
	next := old
	next.Language = languages[index].Code
	if next.Language == a.languageCode() {
		return
	}
	a.applyLanguage(next.Language)
	saved, err := a.engine.SetSettings(next)
	if err != nil {
		a.settings = old
		a.applyLanguage(old.Language)
		platform.ErrorDialog(a.tr("settings.title"), a.tr("settings.save_failed"), a.userMessage(err, "settings.save_failed_body"))
		return
	}
	a.settings = saved
	a.applyLanguage(saved.Language)
	a.setStatus(a.tr("settings.saved", saved.Parallelism, saved.ConnectionTimeoutSeconds, retrySummary(a, saved), overwriteSummary(a, saved)))
}

func retrySummary(a *app, settings model.Settings) string {
	if settings.AutoRetryCount == 0 {
		return a.tr("settings.retry_none")
	}
	return a.tr("settings.retry_count", settings.AutoRetryCount)
}

func overwriteSummary(a *app, settings model.Settings) string {
	if settings.SkipExisting {
		return a.tr("settings.skip_existing")
	}
	return a.tr("settings.overwrite")
}

func (a *app) localizedItemType(item model.Item) string {
	if item.IsSymlink {
		return a.tr("type.link")
	}
	if item.IsDirectory {
		return a.tr("type.folder")
	}
	ext := fileExtensionType(item.Name)
	if ext != "" {
		return ext
	}
	return a.tr("type.file")
}

func fileExtensionType(name string) string {
	ext := ""
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			if i+1 < len(name) {
				ext = name[i+1:]
			}
			break
		}
		if name[i] == '/' || name[i] == '\\' {
			break
		}
	}
	if ext == "" {
		return ""
	}
	ext = stringToUpperASCII(ext)
	if len(ext) > 8 {
		ext = ext[:8]
	}
	return ext
}

func stringToUpperASCII(value string) string {
	out := []byte(value)
	for i, b := range out {
		if b >= 'a' && b <= 'z' {
			out[i] = b - ('a' - 'A')
		}
	}
	return string(out)
}

func (a *app) fillItemList(list uintptr, items []model.Item) {
	setListRedraw(list, false)
	defer setListRedraw(list, true)
	clearList(list)
	for index, item := range items {
		size := formatSize(item.Size, false)
		if item.IsDirectory {
			size = a.tr("type.folder")
		}
		insertListRowWithImage(list, index, []string{item.Name, a.localizedItemType(item), size, formatTime(item.Modified)}, systemIconIndex(item.Name, item.IsDirectory))
	}
}

func (a *app) transferStatusText(status string) string {
	key := map[string]string{
		"queued": "status.queued", "running": "status.running", "done": "status.done",
		"failed": "status.failed", "cancelled": "status.cancelled", "skipped": "status.skipped",
	}[status]
	if key == "" {
		return status
	}
	return a.tr(key)
}

func (a *app) fillTransferList(list uintptr, jobs []model.TransferJob) {
	setListRedraw(list, false)
	defer setListRedraw(list, true)
	clearList(list)
	for index, job := range jobs {
		direction := a.tr("direction.download")
		if job.Direction == "upload" {
			direction = a.tr("direction.upload")
		}
		status := a.transferStatusText(job.Status)
		if job.Status == "running" && job.Attempts > 1 {
			status += fmt.Sprintf(" · #%d", job.Attempts)
		}
		if job.Error != "" {
			status += ": " + job.Error
		}
		progress := job.Progress
		if progress > 0 && progress <= 1 {
			progress *= 100
		}
		if progress < 0 {
			progress = 0
		}
		if progress > 100 {
			progress = 100
		}
		insertListRow(list, index, []string{direction, job.LocalPath, job.RemotePath, status, fmt.Sprintf("%.0f%%", progress)})
	}
}
