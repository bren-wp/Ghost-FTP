package desktop

import (
	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

type appearanceWords struct {
	Title string
	Dark  string
	Light string
}

var localizedAppearanceWords = map[string]appearanceWords{
	"en": {"Appearance", "Dark", "Classic Light"},
	"hr": {"Izgled", "Tamna", "Klasična svijetla"},
	"de": {"Erscheinungsbild", "Dunkel", "Klassisch hell"},
	"fr": {"Apparence", "Sombre", "Clair classique"},
	"es": {"Apariencia", "Oscuro", "Claro clásico"},
	"tr": {"Görünüm", "Koyu", "Klasik açık"},
	"el": {"Εμφάνιση", "Σκούρο", "Κλασικό φωτεινό"},
	"pt": {"Aparência", "Escuro", "Claro clássico"},
	"zh": {"外观", "深色", "经典浅色"},
	"ru": {"Оформление", "Тёмная", "Классическая светлая"},
	"hi": {"रूप-रंग", "गहरा", "क्लासिक हल्का"},
	"ja": {"外観", "ダーク", "クラシックライト"},
	"it": {"Aspetto", "Scuro", "Chiaro classico"},
	"pl": {"Wygląd", "Ciemny", "Klasyczny jasny"},
	"nl": {"Weergave", "Donker", "Klassiek licht"},
	"cs": {"Vzhled", "Tmavý", "Klasický světlý"},
	"uk": {"Вигляд", "Темна", "Класична світла"},
	"sv": {"Utseende", "Mörk", "Klassiskt ljus"},
	"ro": {"Aspect", "Întunecat", "Luminos clasic"},
	"hu": {"Megjelenés", "Sötét", "Klasszikus világos"},
	"da": {"Udseende", "Mørk", "Klassisk lys"},
	"fi": {"Ulkoasu", "Tumma", "Klassinen vaalea"},
	"no": {"Utseende", "Mørk", "Klassisk lys"},
	"ko": {"모양", "어둡게", "클래식 밝게"},
}

func appearanceText(language string) appearanceWords {
	if words, ok := localizedAppearanceWords[i18n.Normalize(language)]; ok {
		return words
	}
	return localizedAppearanceWords[i18n.DefaultLanguage]
}

func appearanceIndex(appearance string) int {
	if appearance == model.AppearanceLight {
		return 1
	}
	return 0
}

func applyAppearanceSelection(settings *model.Settings, index int) {
	if settings == nil {
		return
	}
	if index == 1 {
		settings.Appearance = model.AppearanceLight
		return
	}
	settings.Appearance = model.AppearanceDark
}
