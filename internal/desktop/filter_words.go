package desktop

import "github.com/bren-wp/Ghost-FTP/internal/i18n"

type fileFilterWords struct {
	Cue string
}

var fileFilterTranslations = map[string]fileFilterWords{
	"en": {"Filter current folder…"},
	"hr": {"Filtriraj trenutnu mapu…"},
	"de": {"Aktuellen Ordner filtern…"},
	"fr": {"Filtrer le dossier actuel…"},
	"es": {"Filtrar la carpeta actual…"},
	"tr": {"Geçerli klasörü filtrele…"},
	"el": {"Φιλτράρισμα τρέχοντος φακέλου…"},
	"pt": {"Filtrar a pasta atual…"},
	"zh": {"筛选当前文件夹…"},
	"ru": {"Фильтр текущей папки…"},
	"hi": {"वर्तमान फ़ोल्डर फ़िल्टर करें…"},
	"ja": {"現在のフォルダーを絞り込み…"},
	"it": {"Filtra la cartella corrente…"},
	"pl": {"Filtruj bieżący folder…"},
	"nl": {"Huidige map filteren…"},
	"cs": {"Filtrovat aktuální složku…"},
	"uk": {"Фільтрувати поточну папку…"},
	"sv": {"Filtrera aktuell mapp…"},
	"ro": {"Filtrează folderul curent…"},
	"hu": {"Aktuális mappa szűrése…"},
	"da": {"Filtrer aktuel mappe…"},
	"fi": {"Suodata nykyinen kansio…"},
	"no": {"Filtrer gjeldende mappe…"},
	"ko": {"현재 폴더 필터링…"},
}

func fileFilterWordsForLanguage(language string) fileFilterWords {
	if words, ok := fileFilterTranslations[i18n.Normalize(language)]; ok {
		return words
	}
	return fileFilterTranslations[i18n.DefaultLanguage]
}
