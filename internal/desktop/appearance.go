package desktop

import (
	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

type appearanceWords struct {
	Title string
	Dark  string
	Light string
	Hint  string
}

var localizedAppearanceWords = map[string]appearanceWords{
	"en": {"Appearance", "Dark", "Classic Light", "The selected appearance is applied the next time Ghost FTP starts."},
	"hr": {"Izgled", "Tamna", "Klasična svijetla", "Odabrani izgled primjenjuje se pri sljedećem pokretanju Ghost FTP-a."},
	"de": {"Erscheinungsbild", "Dunkel", "Klassisch hell", "Das gewählte Erscheinungsbild wird beim nächsten Start von Ghost FTP angewendet."},
	"fr": {"Apparence", "Sombre", "Clair classique", "L’apparence choisie sera appliquée au prochain démarrage de Ghost FTP."},
	"es": {"Apariencia", "Oscuro", "Claro clásico", "La apariencia seleccionada se aplicará la próxima vez que se inicie Ghost FTP."},
	"tr": {"Görünüm", "Koyu", "Klasik açık", "Seçilen görünüm Ghost FTP bir sonraki başlatıldığında uygulanır."},
	"el": {"Εμφάνιση", "Σκούρο", "Κλασικό φωτεινό", "Η επιλεγμένη εμφάνιση εφαρμόζεται στην επόμενη εκκίνηση του Ghost FTP."},
	"pt": {"Aparência", "Escuro", "Claro clássico", "A aparência selecionada será aplicada na próxima inicialização do Ghost FTP."},
	"zh": {"外观", "深色", "经典浅色", "所选外观将在下次启动 Ghost FTP 时应用。"},
	"ru": {"Оформление", "Тёмная", "Классическая светлая", "Выбранное оформление применяется при следующем запуске Ghost FTP."},
	"hi": {"रूप-रंग", "गहरा", "क्लासिक हल्का", "चुना गया रूप-रंग Ghost FTP के अगले प्रारंभ पर लागू होगा।"},
	"ja": {"外観", "ダーク", "クラシックライト", "選択した外観は Ghost FTP の次回起動時に適用されます。"},
	"it": {"Aspetto", "Scuro", "Chiaro classico", "L’aspetto selezionato verrà applicato al prossimo avvio di Ghost FTP."},
	"pl": {"Wygląd", "Ciemny", "Klasyczny jasny", "Wybrany wygląd zostanie zastosowany przy następnym uruchomieniu Ghost FTP."},
	"nl": {"Weergave", "Donker", "Klassiek licht", "De gekozen weergave wordt toegepast wanneer Ghost FTP opnieuw wordt gestart."},
	"cs": {"Vzhled", "Tmavý", "Klasický světlý", "Vybraný vzhled se použije při příštím spuštění Ghost FTP."},
	"uk": {"Вигляд", "Темна", "Класична світла", "Вибраний вигляд буде застосовано під час наступного запуску Ghost FTP."},
	"sv": {"Utseende", "Mörk", "Klassiskt ljus", "Det valda utseendet används nästa gång Ghost FTP startas."},
	"ro": {"Aspect", "Întunecat", "Luminos clasic", "Aspectul selectat va fi aplicat la următoarea pornire Ghost FTP."},
	"hu": {"Megjelenés", "Sötét", "Klasszikus világos", "A kiválasztott megjelenés a Ghost FTP következő indításakor lép életbe."},
	"da": {"Udseende", "Mørk", "Klassisk lys", "Det valgte udseende anvendes næste gang Ghost FTP starter."},
	"fi": {"Ulkoasu", "Tumma", "Klassinen vaalea", "Valittu ulkoasu otetaan käyttöön Ghost FTP:n seuraavan käynnistyksen yhteydessä."},
	"no": {"Utseende", "Mørk", "Klassisk lys", "Det valgte utseendet brukes neste gang Ghost FTP starter."},
	"ko": {"모양", "어둡게", "클래식 밝게", "선택한 모양은 Ghost FTP를 다음에 시작할 때 적용됩니다."},
}

func appearanceText(language string) appearanceWords {
	if words, ok := localizedAppearanceWords[i18n.Normalize(language)]; ok {
		return words
	}
	return localizedAppearanceWords[i18n.DefaultLanguage]
}

func appearanceIndex(appearance string) int {
	if appearance == model.AppearanceDark {
		return 0
	}
	return 1
}

func applyAppearanceSelection(settings *model.Settings, index int) {
	if settings == nil {
		return
	}
	if index == 0 {
		settings.Appearance = model.AppearanceDark
		return
	}
	settings.Appearance = model.AppearanceLight
}
