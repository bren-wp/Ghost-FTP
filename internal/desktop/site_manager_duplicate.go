package desktop

import (
	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

var siteManagerDuplicateLabels = map[string]string{
	"en": "Duplicate",
	"hr": "Dupliciraj",
	"de": "Duplizieren",
	"fr": "Dupliquer",
	"es": "Duplicar",
	"tr": "Çoğalt",
	"el": "Δημιουργία αντιγράφου",
	"pt": "Duplicar",
	"zh": "创建副本",
	"ru": "Дублировать",
	"hi": "डुप्लिकेट",
	"ja": "複製",
	"it": "Duplica",
	"pl": "Duplikuj",
	"nl": "Dupliceren",
	"cs": "Duplikovat",
	"uk": "Дублювати",
	"sv": "Duplicera",
	"ro": "Duplică",
	"hu": "Duplikálás",
	"da": "Dupliker",
	"fi": "Monista",
	"no": "Dupliser",
	"ko": "복제",
}

func siteManagerDuplicateLabel(language string) string {
	if label := siteManagerDuplicateLabels[i18n.Normalize(language)]; label != "" {
		return label
	}
	return siteManagerDuplicateLabels[i18n.DefaultLanguage]
}

// duplicateProfileDraft copies only non-secret profile configuration into a
// new profile draft. Stored credentials and the SFTP host-key pin deliberately
// do not cross the new profile boundary: credentials must be entered again and
// SFTP trust must be established for the duplicate before it can be persisted.
func duplicateProfileDraft(profile model.PublicProfile) model.ProfileInput {
	return model.ProfileInput{
		Name:           profile.Name + " 2",
		Protocol:       profile.Protocol,
		Host:           profile.Host,
		Port:           profile.Port,
		Username:       profile.Username,
		PrivateKeyPath: profile.PrivateKeyPath,
		RemotePath:     profile.RemotePath,
		LocalPath:      profile.LocalPath,
	}
}
