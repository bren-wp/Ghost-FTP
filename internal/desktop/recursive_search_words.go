package desktop

import "github.com/bren-wp/Ghost-FTP/internal/i18n"

type recursiveSearchWords struct {
	Search           string
	Navigate         string
	Cancel           string
	Close            string
	Searching        string
	Done             string
	Cancelled        string
	NoResults        string
	LocalDisclosure  string
	RemoteDisclosure string
}

var recursiveSearchTranslations = map[string]recursiveSearchWords{
	"en": {"Search subfolders…", "Go to result", "Cancel", "Close search", "Searching nested folders…", "Search complete", "Search cancelled", "No matches found", "Reads nested local folders; bounded to 20 seconds and 1,000 results.", "Reads nested server folders; bounded to 20 seconds and 1,000 results."},
	"hr": {"Pretraži podmape…", "Idi na rezultat", "Otkaži", "Zatvori pretragu", "Pretraživanje podmapa…", "Pretraga dovršena", "Pretraga otkazana", "Nema pronađenih rezultata", "Čita lokalne podmape; ograničeno na 20 sekundi i 1.000 rezultata.", "Čita podmape na poslužitelju; ograničeno na 20 sekundi i 1.000 rezultata."},
	"de": {"Unterordner durchsuchen…", "Zum Ergebnis", "Abbrechen", "Suche schließen", "Unterordner werden durchsucht…", "Suche abgeschlossen", "Suche abgebrochen", "Keine Treffer gefunden", "Liest lokale Unterordner; begrenzt auf 20 Sekunden und 1.000 Ergebnisse.", "Liest Server-Unterordner; begrenzt auf 20 Sekunden und 1.000 Ergebnisse."},
	"fr": {"Rechercher dans les sous-dossiers…", "Aller au résultat", "Annuler", "Fermer la recherche", "Recherche dans les sous-dossiers…", "Recherche terminée", "Recherche annulée", "Aucun résultat trouvé", "Lit les sous-dossiers locaux; limité à 20 secondes et 1 000 résultats.", "Lit les sous-dossiers du serveur; limité à 20 secondes et 1 000 résultats."},
	"es": {"Buscar en subcarpetas…", "Ir al resultado", "Cancelar", "Cerrar búsqueda", "Buscando en subcarpetas…", "Búsqueda completada", "Búsqueda cancelada", "No se encontraron resultados", "Lee subcarpetas locales; limitado a 20 segundos y 1.000 resultados.", "Lee subcarpetas del servidor; limitado a 20 segundos y 1.000 resultados."},
	"tr": {"Alt klasörlerde ara…", "Sonuca git", "İptal", "Aramayı kapat", "Alt klasörler aranıyor…", "Arama tamamlandı", "Arama iptal edildi", "Eşleşme bulunamadı", "Yerel alt klasörleri okur; 20 saniye ve 1.000 sonuçla sınırlıdır.", "Sunucu alt klasörlerini okur; 20 saniye ve 1.000 sonuçla sınırlıdır."},
	"el": {"Αναζήτηση σε υποφακέλους…", "Μετάβαση στο αποτέλεσμα", "Ακύρωση", "Κλείσιμο αναζήτησης", "Αναζήτηση σε υποφακέλους…", "Η αναζήτηση ολοκληρώθηκε", "Η αναζήτηση ακυρώθηκε", "Δεν βρέθηκαν αποτελέσματα", "Διαβάζει τοπικούς υποφακέλους· όριο 20 δευτερόλεπτα και 1.000 αποτελέσματα.", "Διαβάζει υποφακέλους διακομιστή· όριο 20 δευτερόλεπτα και 1.000 αποτελέσματα."},
	"pt": {"Pesquisar subpastas…", "Ir para o resultado", "Cancelar", "Fechar pesquisa", "Pesquisando subpastas…", "Pesquisa concluída", "Pesquisa cancelada", "Nenhum resultado encontrado", "Lê subpastas locais; limitado a 20 segundos e 1.000 resultados.", "Lê subpastas do servidor; limitado a 20 segundos e 1.000 resultados."},
	"zh": {"搜索子文件夹…", "转到结果", "取消", "关闭搜索", "正在搜索子文件夹…", "搜索完成", "搜索已取消", "未找到匹配项", "读取本地子文件夹；限制为 20 秒和 1,000 个结果。", "读取服务器子文件夹；限制为 20 秒和 1,000 个结果。"},
	"ru": {"Искать во вложенных папках…", "Перейти к результату", "Отмена", "Закрыть поиск", "Поиск во вложенных папках…", "Поиск завершён", "Поиск отменён", "Совпадений не найдено", "Читает локальные вложенные папки; ограничение 20 секунд и 1 000 результатов.", "Читает вложенные папки сервера; ограничение 20 секунд и 1 000 результатов."},
	"hi": {"सबफ़ोल्डर में खोजें…", "परिणाम पर जाएँ", "रद्द करें", "खोज बंद करें", "सबफ़ोल्डर खोजे जा रहे हैं…", "खोज पूरी हुई", "खोज रद्द की गई", "कोई मिलान नहीं मिला", "स्थानीय सबफ़ोल्डर पढ़ता है; 20 सेकंड और 1,000 परिणामों तक सीमित।", "सर्वर सबफ़ोल्डर पढ़ता है; 20 सेकंड और 1,000 परिणामों तक सीमित।"},
	"ja": {"サブフォルダーを検索…", "結果へ移動", "キャンセル", "検索を閉じる", "サブフォルダーを検索中…", "検索完了", "検索をキャンセルしました", "一致する項目はありません", "ローカルのサブフォルダーを読み取ります。上限は20秒・1,000件です。", "サーバーのサブフォルダーを読み取ります。上限は20秒・1,000件です。"},
	"it": {"Cerca nelle sottocartelle…", "Vai al risultato", "Annulla", "Chiudi ricerca", "Ricerca nelle sottocartelle…", "Ricerca completata", "Ricerca annullata", "Nessun risultato trovato", "Legge le sottocartelle locali; limite di 20 secondi e 1.000 risultati.", "Legge le sottocartelle del server; limite di 20 secondi e 1.000 risultati."},
	"pl": {"Szukaj w podfolderach…", "Przejdź do wyniku", "Anuluj", "Zamknij wyszukiwanie", "Wyszukiwanie w podfolderach…", "Wyszukiwanie zakończone", "Wyszukiwanie anulowane", "Nie znaleziono wyników", "Odczytuje lokalne podfoldery; limit 20 sekund i 1 000 wyników.", "Odczytuje podfoldery serwera; limit 20 sekund i 1 000 wyników."},
	"nl": {"Submappen doorzoeken…", "Ga naar resultaat", "Annuleren", "Zoeken sluiten", "Submappen worden doorzocht…", "Zoeken voltooid", "Zoeken geannuleerd", "Geen overeenkomsten gevonden", "Leest lokale submappen; beperkt tot 20 seconden en 1.000 resultaten.", "Leest server-submappen; beperkt tot 20 seconden en 1.000 resultaten."},
	"cs": {"Prohledat podsložky…", "Přejít na výsledek", "Zrušit", "Zavřít hledání", "Prohledávání podsložek…", "Hledání dokončeno", "Hledání zrušeno", "Nebyly nalezeny žádné výsledky", "Čte místní podsložky; omezeno na 20 sekund a 1 000 výsledků.", "Čte podsložky serveru; omezeno na 20 sekund a 1 000 výsledků."},
	"uk": {"Шукати у вкладених папках…", "Перейти до результату", "Скасувати", "Закрити пошук", "Пошук у вкладених папках…", "Пошук завершено", "Пошук скасовано", "Збігів не знайдено", "Читає локальні вкладені папки; обмеження 20 секунд і 1 000 результатів.", "Читає вкладені папки сервера; обмеження 20 секунд і 1 000 результатів."},
	"sv": {"Sök i undermappar…", "Gå till resultat", "Avbryt", "Stäng sökning", "Söker i undermappar…", "Sökningen är klar", "Sökningen avbröts", "Inga träffar hittades", "Läser lokala undermappar; begränsat till 20 sekunder och 1 000 resultat.", "Läser serverns undermappar; begränsat till 20 sekunder och 1 000 resultat."},
	"ro": {"Caută în subfoldere…", "Mergi la rezultat", "Anulează", "Închide căutarea", "Se caută în subfoldere…", "Căutare finalizată", "Căutare anulată", "Nu s-au găsit rezultate", "Citește subfoldere locale; limitat la 20 de secunde și 1.000 de rezultate.", "Citește subfolderele serverului; limitat la 20 de secunde și 1.000 de rezultate."},
	"hu": {"Keresés almappákban…", "Ugrás a találathoz", "Mégse", "Keresés bezárása", "Keresés az almappákban…", "Keresés kész", "Keresés megszakítva", "Nincs találat", "Helyi almappákat olvas; legfeljebb 20 másodperc és 1 000 találat.", "Kiszolgálói almappákat olvas; legfeljebb 20 másodperc és 1 000 találat."},
	"da": {"Søg i undermapper…", "Gå til resultat", "Annuller", "Luk søgning", "Søger i undermapper…", "Søgning fuldført", "Søgning annulleret", "Ingen resultater fundet", "Læser lokale undermapper; begrænset til 20 sekunder og 1.000 resultater.", "Læser serverens undermapper; begrænset til 20 sekunder og 1.000 resultater."},
	"fi": {"Hae alikansioista…", "Siirry tulokseen", "Peruuta", "Sulje haku", "Haetaan alikansioista…", "Haku valmis", "Haku peruutettu", "Osumia ei löytynyt", "Lukee paikallisia alikansioita; enintään 20 sekuntia ja 1 000 tulosta.", "Lukee palvelimen alikansioita; enintään 20 sekuntia ja 1 000 tulosta."},
	"no": {"Søk i undermapper…", "Gå til resultat", "Avbryt", "Lukk søk", "Søker i undermapper…", "Søk fullført", "Søk avbrutt", "Ingen treff funnet", "Leser lokale undermapper; begrenset til 20 sekunder og 1 000 resultater.", "Leser serverens undermapper; begrenset til 20 sekunder og 1 000 resultater."},
	"ko": {"하위 폴더 검색…", "결과로 이동", "취소", "검색 닫기", "하위 폴더 검색 중…", "검색 완료", "검색 취소됨", "일치하는 항목이 없습니다", "로컬 하위 폴더를 읽습니다. 최대 20초 및 1,000개 결과로 제한됩니다.", "서버 하위 폴더를 읽습니다. 최대 20초 및 1,000개 결과로 제한됩니다."},
}

func recursiveSearchWordsForLanguage(language string) recursiveSearchWords {
	if words, ok := recursiveSearchTranslations[i18n.Normalize(language)]; ok {
		return words
	}
	return recursiveSearchTranslations[i18n.DefaultLanguage]
}
