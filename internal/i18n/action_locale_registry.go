package i18n

var actionLocaleKeys = []string{
	"action.failed_retry", "action.failed_title", "file.choose_folder_failed_title", "file.choose_folder_failed_body",
	"local.open_failed", "remote.open_failed", "item.open_unsafe", "item.symlink_download_blocked", "item.save_unsafe",
	"file.new_name", "file.folder_name", "file.default_folder", "selection.items",
	"local.mkdir.title", "local.mkdir.failed_title", "local.mkdir.failed_body", "local.mkdir.success",
	"local.rename.select_one", "local.rename.title", "local.rename.failed_title", "local.rename.failed_body", "local.rename.success",
	"local.delete.select", "local.delete.title", "local.delete.question", "local.delete.warning", "local.delete.partial_title", "local.delete.partial_body", "local.delete.status",
	"remote.mkdir.title", "remote.mkdir.progress", "remote.mkdir.success",
	"remote.rename.select_one", "remote.rename.title", "remote.rename.progress", "remote.rename.success",
	"remote.delete.select", "remote.delete.title", "remote.delete.question", "remote.delete.warning", "remote.delete.progress", "remote.delete.success_prefix",
	"permissions.select", "permissions.limit", "permissions.links_only", "permissions.title", "permissions.prompt", "permissions.progress", "permissions.success_prefix",
	"remote.action.title", "remote.action.failed", "remote.action.body", "remote.batch.partial", "remote.batch.body", "batch.failed", "batch.skipped_links",
	"transfer.links_skipped", "transfer.none", "transfer.adding_selection", "transfer.partial_title", "transfer.partial_body", "transfer.queue_files", "transfer.queue_dirs", "transfer.queue_failed", "transfer.queue_skipped_links",
	"transfer.select_upload", "transfer.select_download", "transfer.tree_preparing", "transfer.tree_title", "transfer.tree_failed_title", "transfer.tree_failed_body", "transfer.tree_added",
	"connection.lost_stopping", "connection.lost_done",
	"transfer.adding", "transfer.start_failed_title", "transfer.start_failed_body", "transfer.start_failed_status", "transfer.queued",
	"transfer.pause_status", "transfer.resume_status", "transfer.clear_status",
	"transfer.select_active_cancel", "transfer.cancel_failed_body", "transfer.cancelled_count", "transfer.select_retry", "transfer.retry_requires_connection", "transfer.retry_failed_body", "transfer.requeued_count",
	"profile.edit_blocked_title", "profile.edit_blocked_body", "profile.dialog_title", "profile.name_prompt", "profile.name_missing_title", "profile.name_missing_body",
	"privacy.title", "profile.store_credentials_title", "profile.store_credentials_body", "profile.retain_credentials_title", "profile.retain_credentials_body", "profile.credentials_auto_removed_intro", "profile.old_credentials_title", "profile.old_credentials_body",
	"profile.save_failed_title", "profile.save_failed_body", "profile.saved", "profile.none_selected_title", "profile.none_selected_body", "profile.delete_connected_title", "profile.delete_connected_body", "profile.delete_title", "profile.delete_question", "profile.delete_body", "profile.delete_failed_title", "profile.delete_failed_body", "profile.deleted",
}

func actionLocale(values []string) map[string]string {
	if len(values) != len(actionLocaleKeys) {
		panic("ByFTP action localization key/value count mismatch")
	}
	out := make(map[string]string, len(values))
	for i, key := range actionLocaleKeys {
		out[key] = values[i]
	}
	return out
}

var secondaryActionCatalogs = map[string]map[string]string{
	"de": actionLocale(actionDEValues),
	"fr": actionLocale(actionFRValues),
	"es": actionLocale(actionESValues),
	"tr": actionLocale(actionTRValues),
	"el": actionLocale(actionELValues),
	"pt": actionLocale(actionPTValues),
	"zh": actionLocale(actionZHValues),
	"ru": actionLocale(actionRUValues),
	"hi": actionLocale(actionHIValues),
	"ja": actionLocale(actionJAValues),
}

var _ = registerSecondaryActionCatalogs()

func registerSecondaryActionCatalogs() bool {
	for code, entries := range secondaryActionCatalogs {
		actionCatalogs[code] = entries
	}
	return true
}
