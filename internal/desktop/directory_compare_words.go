package desktop

import "github.com/bren-wp/Ghost-FTP/internal/i18n"

type directoryCompareWords struct {
	Compare     string
	Close       string
	OpenBoth    string
	Same        string
	LocalOnly   string
	RemoteOnly  string
	NewerLocal  string
	NewerRemote string
	Conflict    string
	Unknown     string
	Ready       string
	Unavailable string
	Disclosure  string
}

var directoryCompareTranslations = map[string]directoryCompareWords{
	"en": {"Compare", "Close comparison", "Open both", "Same", "Local only", "Server only", "Local newer", "Server newer", "Conflict", "Unknown", "Directory comparison ready", "Synchronized navigation is unavailable for this entry", "Read-only metadata comparison. It never transfers, overwrites, renames or deletes files."},
	"hr": {"Usporedi", "Zatvori usporedbu", "Otvori obje", "Isto", "Samo lokalno", "Samo poslužitelj", "Lokalno novije", "Poslužitelj noviji", "Sukob", "Nepoznato", "Usporedba direktorija spremna", "Sinkronizirana navigacija nije dostupna za ovu stavku", "Usporedba metapodataka samo za čitanje. Nikada ne prenosi, prepisuje, preimenuje niti briše datoteke."},
	"de": {"Vergleichen", "Vergleich schließen", "Beide öffnen", "Gleich", "Nur lokal", "Nur Server", "Lokal neuer", "Server neuer", "Konflikt", "Unbekannt", "Verzeichnisvergleich bereit", "Synchronisierte Navigation ist für diesen Eintrag nicht verfügbar", "Schreibgeschützter Metadatenvergleich. Dateien werden niemals übertragen, überschrieben, umbenannt oder gelöscht."},
	"fr": {"Comparer", "Fermer la comparaison", "Ouvrir les deux", "Identique", "Local uniquement", "Serveur uniquement", "Local plus récent", "Serveur plus récent", "Conflit", "Inconnu", "Comparaison des dossiers prête", "La navigation synchronisée n’est pas disponible pour cet élément", "Comparaison des métadonnées en lecture seule. Aucun transfert, écrasement, renommage ou suppression de fichier."},
	"es": {"Comparar", "Cerrar comparación", "Abrir ambos", "Igual", "Solo local", "Solo servidor", "Local más reciente", "Servidor más reciente", "Conflicto", "Desconocido", "Comparación de carpetas lista", "La navegación sincronizada no está disponible para este elemento", "Comparación de metadatos de solo lectura. Nunca transfiere, sobrescribe, renombra ni elimina archivos."},
	"tr": {"Karşılaştır", "Karşılaştırmayı kapat", "İkisini de aç", "Aynı", "Yalnızca yerel", "Yalnızca sunucu", "Yerel daha yeni", "Sunucu daha yeni", "Çakışma", "Bilinmiyor", "Dizin karşılaştırması hazır", "Bu öğe için eşzamanlı gezinme kullanılamıyor", "Salt okunur meta veri karşılaştırması. Dosyaları asla aktarmaz, üzerine yazmaz, yeniden adlandırmaz veya silmez."},
	"el": {"Σύγκριση", "Κλείσιμο σύγκρισης", "Άνοιγμα και των δύο", "Ίδιο", "Μόνο τοπικά", "Μόνο διακομιστής", "Νεότερο τοπικά", "Νεότερο στον διακομιστή", "Σύγκρουση", "Άγνωστο", "Η σύγκριση φακέλων είναι έτοιμη", "Η συγχρονισμένη πλοήγηση δεν είναι διαθέσιμη για αυτό το στοιχείο", "Σύγκριση μεταδεδομένων μόνο για ανάγνωση. Δεν μεταφέρει, αντικαθιστά, μετονομάζει ή διαγράφει αρχεία."},
	"pt": {"Comparar", "Fechar comparação", "Abrir ambos", "Igual", "Apenas local", "Apenas servidor", "Local mais recente", "Servidor mais recente", "Conflito", "Desconhecido", "Comparação de diretórios pronta", "A navegação sincronizada não está disponível para este item", "Comparação de metadados somente leitura. Nunca transfere, substitui, renomeia ou exclui arquivos."},
	"zh": {"比较", "关闭比较", "同时打开", "相同", "仅本地", "仅服务器", "本地较新", "服务器较新", "冲突", "未知", "目录比较已就绪", "此项目无法使用同步导航", "只读元数据比较。不会传输、覆盖、重命名或删除文件。"},
	"ru": {"Сравнить", "Закрыть сравнение", "Открыть оба", "Одинаково", "Только локально", "Только сервер", "Локальная версия новее", "Серверная версия новее", "Конфликт", "Неизвестно", "Сравнение каталогов готово", "Синхронная навигация недоступна для этого элемента", "Сравнение метаданных только для чтения. Оно не переносит, не перезаписывает, не переименовывает и не удаляет файлы."},
	"hi": {"तुलना करें", "तुलना बंद करें", "दोनों खोलें", "समान", "केवल स्थानीय", "केवल सर्वर", "स्थानीय नया", "सर्वर नया", "टकराव", "अज्ञात", "डायरेक्टरी तुलना तैयार है", "इस आइटम के लिए समकालिक नेविगेशन उपलब्ध नहीं है", "केवल-पठन मेटाडेटा तुलना। यह फ़ाइलों को स्थानांतरित, अधिलेखित, नाम परिवर्तित या हटाती नहीं है।"},
	"ja": {"比較", "比較を閉じる", "両方を開く", "同じ", "ローカルのみ", "サーバーのみ", "ローカルが新しい", "サーバーが新しい", "競合", "不明", "ディレクトリ比較の準備完了", "この項目では同期ナビゲーションを使用できません", "読み取り専用のメタデータ比較です。ファイルの転送、上書き、名前変更、削除は行いません。"},
	"it": {"Confronta", "Chiudi confronto", "Apri entrambi", "Uguale", "Solo locale", "Solo server", "Locale più recente", "Server più recente", "Conflitto", "Sconosciuto", "Confronto directory pronto", "La navigazione sincronizzata non è disponibile per questo elemento", "Confronto dei metadati in sola lettura. Non trasferisce, sovrascrive, rinomina o elimina file."},
	"pl": {"Porównaj", "Zamknij porównanie", "Otwórz oba", "Takie samo", "Tylko lokalnie", "Tylko serwer", "Lokalne nowsze", "Serwerowe nowsze", "Konflikt", "Nieznane", "Porównanie katalogów gotowe", "Nawigacja zsynchronizowana nie jest dostępna dla tego elementu", "Porównanie metadanych tylko do odczytu. Nie przesyła, nie nadpisuje, nie zmienia nazw ani nie usuwa plików."},
	"nl": {"Vergelijken", "Vergelijking sluiten", "Beide openen", "Gelijk", "Alleen lokaal", "Alleen server", "Lokaal nieuwer", "Server nieuwer", "Conflict", "Onbekend", "Mapvergelijking gereed", "Gesynchroniseerde navigatie is niet beschikbaar voor dit item", "Alleen-lezen metadatavergelijking. Bestanden worden nooit overgedragen, overschreven, hernoemd of verwijderd."},
	"cs": {"Porovnat", "Zavřít porovnání", "Otevřít obě", "Stejné", "Pouze místní", "Pouze server", "Místní novější", "Server novější", "Konflikt", "Neznámé", "Porovnání adresářů je připraveno", "Synchronizovaná navigace není pro tuto položku dostupná", "Porovnání metadat pouze pro čtení. Nikdy nepřenáší, nepřepisuje, nepřejmenovává ani nemaže soubory."},
	"uk": {"Порівняти", "Закрити порівняння", "Відкрити обидва", "Однаково", "Лише локально", "Лише сервер", "Локальна версія новіша", "Серверна версія новіша", "Конфлікт", "Невідомо", "Порівняння каталогів готове", "Синхронізована навігація недоступна для цього елемента", "Порівняння метаданих лише для читання. Воно не передає, не перезаписує, не перейменовує та не видаляє файли."},
	"sv": {"Jämför", "Stäng jämförelse", "Öppna båda", "Samma", "Endast lokalt", "Endast server", "Lokalt nyare", "Server nyare", "Konflikt", "Okänt", "Katalogjämförelse klar", "Synkroniserad navigering är inte tillgänglig för denna post", "Skrivskyddad metadatajämförelse. Den överför, skriver över, byter namn på eller tar aldrig bort filer."},
	"ro": {"Compară", "Închide comparația", "Deschide ambele", "Identic", "Doar local", "Doar server", "Local mai nou", "Server mai nou", "Conflict", "Necunoscut", "Compararea directoarelor este gata", "Navigarea sincronizată nu este disponibilă pentru acest element", "Comparare a metadatelor doar în citire. Nu transferă, suprascrie, redenumește sau șterge fișiere."},
	"hu": {"Összehasonlítás", "Összehasonlítás bezárása", "Mindkettő megnyitása", "Azonos", "Csak helyi", "Csak kiszolgáló", "Helyi újabb", "Kiszolgáló újabb", "Ütközés", "Ismeretlen", "A könyvtár-összehasonlítás kész", "A szinkronizált navigáció ehhez az elemhez nem érhető el", "Csak olvasható metaadat-összehasonlítás. Nem visz át, ír felül, nevez át vagy töröl fájlokat."},
	"da": {"Sammenlign", "Luk sammenligning", "Åbn begge", "Samme", "Kun lokal", "Kun server", "Lokal nyere", "Server nyere", "Konflikt", "Ukendt", "Mappesammenligning klar", "Synkroniseret navigation er ikke tilgængelig for dette element", "Skrivebeskyttet metadatasammenligning. Den overfører, overskriver, omdøber eller sletter aldrig filer."},
	"fi": {"Vertaa", "Sulje vertailu", "Avaa molemmat", "Sama", "Vain paikallinen", "Vain palvelin", "Paikallinen uudempi", "Palvelin uudempi", "Ristiriita", "Tuntematon", "Hakemistojen vertailu valmis", "Synkronoitu siirtyminen ei ole käytettävissä tälle kohteelle", "Vain luku -metatietojen vertailu. Se ei siirrä, korvaa, nimeä uudelleen tai poista tiedostoja."},
	"no": {"Sammenlign", "Lukk sammenligning", "Åpne begge", "Samme", "Bare lokal", "Bare server", "Lokal nyere", "Server nyere", "Konflikt", "Ukjent", "Mappe sammenligning klar", "Synkronisert navigasjon er ikke tilgjengelig for dette elementet", "Skrivebeskyttet metadata-sammenligning. Den overfører, overskriver, gir nytt navn til eller sletter aldri filer."},
	"ko": {"비교", "비교 닫기", "둘 다 열기", "같음", "로컬만", "서버만", "로컬이 최신", "서버가 최신", "충돌", "알 수 없음", "디렉터리 비교 준비 완료", "이 항목에는 동기화 탐색을 사용할 수 없습니다", "읽기 전용 메타데이터 비교입니다. 파일을 전송, 덮어쓰기, 이름 변경 또는 삭제하지 않습니다."},
}

func directoryCompareWordsForLanguage(language string) directoryCompareWords {
	if words, ok := directoryCompareTranslations[i18n.Normalize(language)]; ok {
		return words
	}
	return directoryCompareTranslations[i18n.DefaultLanguage]
}
