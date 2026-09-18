package desktop

import "github.com/bren-wp/Ghost-FTP/internal/i18n"

var settingsResetLabels = map[string]string{
	"en": "Restore defaults",
	"hr": "Vrati zadane postavke",
	"de": "Standardwerte wiederherstellen",
	"fr": "Rétablir les valeurs par défaut",
	"es": "Restaurar valores predeterminados",
	"tr": "Varsayılanları geri yükle",
	"el": "Επαναφορά προεπιλογών",
	"pt": "Restaurar predefinições",
	"zh": "恢复默认设置",
	"ru": "Восстановить значения по умолчанию",
	"hi": "डिफ़ॉल्ट पुनर्स्थापित करें",
	"ja": "既定値に戻す",
	"it": "Ripristina predefiniti",
	"pl": "Przywróć domyślne",
	"nl": "Standaardwaarden herstellen",
	"cs": "Obnovit výchozí hodnoty",
	"uk": "Відновити типові значення",
	"sv": "Återställ standardvärden",
	"ro": "Restabilește valorile implicite",
	"hu": "Alapértékek visszaállítása",
	"da": "Gendan standardværdier",
	"fi": "Palauta oletukset",
	"no": "Gjenopprett standardverdier",
	"ko": "기본값 복원",
}

func settingsResetLabel(language string) string {
	if label, ok := settingsResetLabels[i18n.Normalize(language)]; ok {
		return label
	}
	return settingsResetLabels[i18n.DefaultLanguage]
}
