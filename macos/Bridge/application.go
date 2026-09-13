package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

var (
	bookmarkSnapshot      []model.Bookmark
	lastBookmarkID        string
	lastBookmarkKind      string
	lastBookmarkPath      string
	settingsSnapshot      model.Settings
	settingsSnapshotValid bool
	productVersion        = "development"
)

func refreshBookmarksLocked() error {
	if bridgeState.engine == nil {
		return errors.New("engine is not initialized")
	}
	items, err := bridgeState.engine.Bookmarks()
	if err != nil {
		return err
	}
	bookmarkSnapshot = append(bookmarkSnapshot[:0], items...)
	return nil
}

func bookmarkAt(index C.int) (model.Bookmark, bool) {
	i := int(index)
	if i < 0 || i >= len(bookmarkSnapshot) {
		return model.Bookmark{}, false
	}
	return bookmarkSnapshot[i], true
}

//export GhostFTPRefreshBookmarks
func GhostFTPRefreshBookmarks() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if err := refreshBookmarksLocked(); err != nil {
		setBridgeError(err, "Ghost FTP could not load bookmarks.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPBookmarkCount
func GhostFTPBookmarkCount() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	return C.int(len(bookmarkSnapshot))
}

//export GhostFTPBookmarkID
func GhostFTPBookmarkID(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := bookmarkAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(item.ID)
}

//export GhostFTPBookmarkName
func GhostFTPBookmarkName(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := bookmarkAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(item.Name)
}

//export GhostFTPBookmarkKind
func GhostFTPBookmarkKind(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := bookmarkAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(item.Kind)
}

//export GhostFTPBookmarkPath
func GhostFTPBookmarkPath(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := bookmarkAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(item.Path)
}

//export GhostFTPBookmarkHost
func GhostFTPBookmarkHost(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := bookmarkAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(item.Host)
}

//export GhostFTPBookmarkUsername
func GhostFTPBookmarkUsername(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	item, ok := bookmarkAt(index)
	if !ok {
		return C.CString("")
	}
	return C.CString(item.Username)
}

func validateBookmarkName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("bookmark name is required")
	}
	if len([]rune(value)) > 160 {
		return "", errors.New("bookmark name is too long")
	}
	return value, nil
}

// GhostFTPSaveLocalBookmark deliberately accepts only a name. The path comes
// from the authoritative bridge snapshot that backs the visible local pane.
//
//export GhostFTPSaveLocalBookmark
func GhostFTPSaveLocalBookmark(nameValue *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	name, err := validateBookmarkName(goString(nameValue))
	if err != nil {
		setBridgeError(err, "Enter a bookmark name.")
		return 0
	}
	path := strings.TrimSpace(bridgeState.localPath)
	if path == "" {
		setBridgeError(errors.New("local folder is unavailable"), "Open a local folder before saving a bookmark.")
		return 0
	}
	saved, err := bridgeState.engine.SaveLocalBookmark("", name, path)
	if err != nil {
		setBridgeError(err, "Ghost FTP could not save the local bookmark.")
		return 0
	}
	lastBookmarkID = saved.ID
	if err := refreshBookmarksLocked(); err != nil {
		setBridgeError(err, "The bookmark was saved but the list could not be refreshed.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}

// GhostFTPSaveRemoteBookmark deliberately accepts only a name. The shared
// engine binds the authoritative visible path to the active server account.
//
//export GhostFTPSaveRemoteBookmark
func GhostFTPSaveRemoteBookmark(nameValue *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	name, err := validateBookmarkName(goString(nameValue))
	if err != nil {
		setBridgeError(err, "Enter a bookmark name.")
		return 0
	}
	path := strings.TrimSpace(bridgeState.remotePath)
	if path == "" {
		setBridgeError(errors.New("remote folder is unavailable"), "Open a remote folder before saving a bookmark.")
		return 0
	}
	saved, err := bridgeState.engine.SaveRemoteBookmark("", name, path)
	if err != nil {
		setBridgeError(err, "Ghost FTP could not save the remote bookmark.")
		return 0
	}
	lastBookmarkID = saved.ID
	if err := refreshBookmarksLocked(); err != nil {
		setBridgeError(err, "The bookmark was saved but the list could not be refreshed.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPLastSavedBookmarkID
func GhostFTPLastSavedBookmarkID() *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	return C.CString(lastBookmarkID)
}

//export GhostFTPRemoveBookmark
func GhostFTPRemoveBookmark(idValue *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	id := strings.TrimSpace(goString(idValue))
	if id == "" {
		setBridgeError(errors.New("bookmark id is empty"), "Select a bookmark first.")
		return 0
	}
	if err := bridgeState.engine.RemoveBookmark(id); err != nil {
		setBridgeError(err, "Ghost FTP could not remove the bookmark.")
		return 0
	}
	if err := refreshBookmarksLocked(); err != nil {
		setBridgeError(err, "The bookmark was removed but the list could not be refreshed.")
		return 0
	}
	if lastBookmarkID == id {
		lastBookmarkID = ""
	}
	bridgeState.lastError = ""
	return 1
}

// GhostFTPNavigateBookmark commits a visible bridge snapshot only after the
// shared engine has proved the target. Remote navigation therefore retains the
// account/connection-identity checks in Engine.NavigateBookmark.
//
//export GhostFTPNavigateBookmark
func GhostFTPNavigateBookmark(idValue *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	id := strings.TrimSpace(goString(idValue))
	if id == "" {
		setBridgeError(errors.New("bookmark id is empty"), "Select a bookmark first.")
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	navigation, err := bridgeState.engine.NavigateBookmark(ctx, id)
	if err != nil {
		setBridgeError(err, "Ghost FTP could not open the bookmark safely.")
		return 0
	}
	lastBookmarkKind = navigation.Bookmark.Kind
	switch navigation.Bookmark.Kind {
	case model.BookmarkKindLocal:
		bridgeState.localPath = navigation.LocalBase
		bridgeState.localItems = append(bridgeState.localItems[:0], navigation.Items...)
		lastBookmarkPath = navigation.LocalBase
	case model.BookmarkKindRemote:
		bridgeState.remotePath = cleanRemotePath(navigation.Bookmark.Path)
		bridgeState.remoteItems = append(bridgeState.remoteItems[:0], navigation.Items...)
		lastBookmarkPath = bridgeState.remotePath
	default:
		setBridgeError(errors.New("unsupported bookmark kind"), "Ghost FTP could not open this bookmark.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPLastBookmarkKind
func GhostFTPLastBookmarkKind() *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	return C.CString(lastBookmarkKind)
}

//export GhostFTPLastBookmarkPath
func GhostFTPLastBookmarkPath() *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	return C.CString(lastBookmarkPath)
}

func refreshSettingsLocked() error {
	if bridgeState.engine == nil {
		return errors.New("engine is not initialized")
	}
	value, err := bridgeState.engine.Settings()
	if err != nil {
		return err
	}
	settingsSnapshot = value
	settingsSnapshotValid = true
	return nil
}

func ensureSettingsLocked() error {
	if settingsSnapshotValid {
		return nil
	}
	return refreshSettingsLocked()
}

//export GhostFTPRefreshSettings
func GhostFTPRefreshSettings() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if err := refreshSettingsLocked(); err != nil {
		setBridgeError(err, "Ghost FTP could not load settings.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPSettingsLanguage
func GhostFTPSettingsLanguage() *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if ensureSettingsLocked() != nil {
		return C.CString("")
	}
	return C.CString(settingsSnapshot.Language)
}

//export GhostFTPSettingsAppearance
func GhostFTPSettingsAppearance() *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if ensureSettingsLocked() != nil {
		return C.CString("")
	}
	return C.CString(settingsSnapshot.Appearance)
}

//export GhostFTPSettingsParallelism
func GhostFTPSettingsParallelism() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if ensureSettingsLocked() != nil {
		return 0
	}
	return C.int(settingsSnapshot.Parallelism)
}

//export GhostFTPSettingsUploadLimit
func GhostFTPSettingsUploadLimit() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if ensureSettingsLocked() != nil {
		return 0
	}
	return C.int(settingsSnapshot.UploadLimitKiBPerSecond)
}

//export GhostFTPSettingsDownloadLimit
func GhostFTPSettingsDownloadLimit() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if ensureSettingsLocked() != nil {
		return 0
	}
	return C.int(settingsSnapshot.DownloadLimitKiBPerSecond)
}

//export GhostFTPSettingsConnectionTimeout
func GhostFTPSettingsConnectionTimeout() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if ensureSettingsLocked() != nil {
		return 0
	}
	return C.int(settingsSnapshot.ConnectionTimeoutSeconds)
}

//export GhostFTPSettingsAutoRetryCount
func GhostFTPSettingsAutoRetryCount() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if ensureSettingsLocked() != nil {
		return 0
	}
	return C.int(settingsSnapshot.AutoRetryCount)
}

//export GhostFTPSettingsRetryDelay
func GhostFTPSettingsRetryDelay() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if ensureSettingsLocked() != nil {
		return 0
	}
	return C.int(settingsSnapshot.RetryDelaySeconds)
}

//export GhostFTPSettingsConflictPolicy
func GhostFTPSettingsConflictPolicy() *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if ensureSettingsLocked() != nil {
		return C.CString("")
	}
	return C.CString(settingsSnapshot.ConflictPolicy)
}

//export GhostFTPSettingsConfirmDelete
func GhostFTPSettingsConfirmDelete() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if ensureSettingsLocked() != nil && !settingsSnapshot.ConfirmDelete {
		return 0
	}
	if settingsSnapshot.ConfirmDelete {
		return 1
	}
	return 0
}

// GhostFTPSaveSettings starts from the authoritative current shared settings so
// future fields are preserved rather than zeroed by an older native frontend.
//
//export GhostFTPSaveSettings
func GhostFTPSaveSettings(languageValue, appearanceValue *C.char, parallelism, uploadLimit, downloadLimit, connectionTimeout, autoRetryCount, retryDelay C.int, conflictPolicyValue *C.char, confirmDelete C.int) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	if err := ensureSettingsLocked(); err != nil {
		setBridgeError(err, "Ghost FTP could not load the current settings.")
		return 0
	}
	next := settingsSnapshot
	next.Language = strings.TrimSpace(goString(languageValue))
	next.Appearance = strings.TrimSpace(goString(appearanceValue))
	next.Parallelism = int(parallelism)
	next.UploadLimitKiBPerSecond = int(uploadLimit)
	next.DownloadLimitKiBPerSecond = int(downloadLimit)
	next.ConnectionTimeoutSeconds = int(connectionTimeout)
	next.AutoRetryCount = int(autoRetryCount)
	next.RetryDelaySeconds = int(retryDelay)
	next.ConflictPolicy = strings.TrimSpace(goString(conflictPolicyValue))
	next.ConfirmDelete = confirmDelete != 0
	saved, err := bridgeState.engine.SetSettings(next)
	if err != nil {
		setBridgeError(err, "Ghost FTP could not save these settings.")
		return 0
	}
	settingsSnapshot = saved
	settingsSnapshotValid = true
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPLanguageCount
func GhostFTPLanguageCount() C.int { return C.int(len(i18n.Languages())) }

//export GhostFTPLanguageCode
func GhostFTPLanguageCode(index C.int) *C.char {
	languages := i18n.Languages()
	i := int(index)
	if i < 0 || i >= len(languages) {
		return C.CString("")
	}
	return C.CString(languages[i].Code)
}

//export GhostFTPLanguageNativeName
func GhostFTPLanguageNativeName(index C.int) *C.char {
	languages := i18n.Languages()
	i := int(index)
	if i < 0 || i >= len(languages) {
		return C.CString("")
	}
	return C.CString(languages[i].NativeName)
}

//export GhostFTPLanguageEnglishName
func GhostFTPLanguageEnglishName(index C.int) *C.char {
	languages := i18n.Languages()
	i := int(index)
	if i < 0 || i >= len(languages) {
		return C.CString("")
	}
	return C.CString(languages[i].EnglishName)
}

//export GhostFTPDiagnosticsConnected
func GhostFTPDiagnosticsConnected() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		return 0
	}
	_, connected := bridgeState.engine.ActiveConnection()
	if connected {
		return 1
	}
	return 0
}

//export GhostFTPDiagnosticsProtocol
func GhostFTPDiagnosticsProtocol() *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		return C.CString("")
	}
	cfg, connected := bridgeState.engine.ActiveConnection()
	if !connected {
		return C.CString("")
	}
	return C.CString(strings.ToUpper(cfg.Protocol))
}

//export GhostFTPDiagnosticsRemotePath
func GhostFTPDiagnosticsRemotePath() *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	return C.CString(bridgeState.remotePath)
}

//export GhostFTPAboutVersion
func GhostFTPAboutVersion() *C.char { return C.CString(productVersion) }

//export GhostFTPAboutPublisher
func GhostFTPAboutPublisher() *C.char { return C.CString(macAboutPublisher) }

//export GhostFTPAboutWebsite
func GhostFTPAboutWebsite() *C.char { return C.CString(macAboutWebsite) }

//export GhostFTPAboutAuthorWebsite
func GhostFTPAboutAuthorWebsite() *C.char { return C.CString(macAboutAuthorWebsite) }

//export GhostFTPAboutSupport
func GhostFTPAboutSupport() *C.char { return C.CString(macAboutSupport) }
