package desktop

import "github.com/bren-wp/Ghost-FTP/internal/i18n"

type bookmarkWords struct {
	Title, AddLocal, AddRemote, Open, Delete, Close, Name, Empty, Saved, Removed string
}

var bookmarkWordCatalog = map[string]bookmarkWords{
	"en": {"Bookmarks", "Add local", "Add remote", "Open", "Delete", "Close", "Name", "No bookmarks yet.", "Bookmark saved.", "Bookmark removed."},
	"hr": {"Oznake", "Dodaj lokalnu", "Dodaj udaljenu", "Otvori", "Izbriši", "Zatvori", "Naziv", "Još nema oznaka.", "Oznaka je spremljena.", "Oznaka je uklonjena."},
	"de": {"Lesezeichen", "Lokal hinzufügen", "Remote hinzufügen", "Öffnen", "Löschen", "Schließen", "Name", "Noch keine Lesezeichen.", "Lesezeichen gespeichert.", "Lesezeichen entfernt."},
	"fr": {"Favoris", "Ajouter local", "Ajouter distant", "Ouvrir", "Supprimer", "Fermer", "Nom", "Aucun favori.", "Favori enregistré.", "Favori supprimé."},
	"es": {"Marcadores", "Añadir local", "Añadir remoto", "Abrir", "Eliminar", "Cerrar", "Nombre", "Aún no hay marcadores.", "Marcador guardado.", "Marcador eliminado."},
	"tr": {"Yer imleri", "Yerel ekle", "Uzak ekle", "Aç", "Sil", "Kapat", "Ad", "Henüz yer imi yok.", "Yer imi kaydedildi.", "Yer imi kaldırıldı."},
	"el": {"Σελιδοδείκτες", "Προσθήκη τοπικού", "Προσθήκη απομακρυσμένου", "Άνοιγμα", "Διαγραφή", "Κλείσιμο", "Όνομα", "Δεν υπάρχουν σελιδοδείκτες.", "Ο σελιδοδείκτης αποθηκεύτηκε.", "Ο σελιδοδείκτης αφαιρέθηκε."},
	"pt": {"Favoritos", "Adicionar local", "Adicionar remoto", "Abrir", "Eliminar", "Fechar", "Nome", "Ainda não há favoritos.", "Favorito guardado.", "Favorito removido."},
	"zh": {"书签", "添加本地", "添加远程", "打开", "删除", "关闭", "名称", "暂无书签。", "书签已保存。", "书签已删除。"},
	"ru": {"Закладки", "Добавить локальную", "Добавить удалённую", "Открыть", "Удалить", "Закрыть", "Имя", "Закладок пока нет.", "Закладка сохранена.", "Закладка удалена."},
	"hi": {"बुकमार्क", "लोकल जोड़ें", "रिमोट जोड़ें", "खोलें", "हटाएँ", "बंद करें", "नाम", "अभी कोई बुकमार्क नहीं है।", "बुकमार्क सहेजा गया।", "बुकमार्क हटाया गया।"},
	"ja": {"ブックマーク", "ローカルを追加", "リモートを追加", "開く", "削除", "閉じる", "名前", "ブックマークはまだありません。", "ブックマークを保存しました。", "ブックマークを削除しました。"},
	"it": {"Segnalibri", "Aggiungi locale", "Aggiungi remoto", "Apri", "Elimina", "Chiudi", "Nome", "Nessun segnalibro.", "Segnalibro salvato.", "Segnalibro rimosso."},
	"pl": {"Zakładki", "Dodaj lokalną", "Dodaj zdalną", "Otwórz", "Usuń", "Zamknij", "Nazwa", "Brak zakładek.", "Zakładka zapisana.", "Zakładka usunięta."},
	"nl": {"Bladwijzers", "Lokaal toevoegen", "Extern toevoegen", "Openen", "Verwijderen", "Sluiten", "Naam", "Nog geen bladwijzers.", "Bladwijzer opgeslagen.", "Bladwijzer verwijderd."},
	"cs": {"Záložky", "Přidat místní", "Přidat vzdálenou", "Otevřít", "Smazat", "Zavřít", "Název", "Zatím žádné záložky.", "Záložka uložena.", "Záložka odstraněna."},
	"uk": {"Закладки", "Додати локальну", "Додати віддалену", "Відкрити", "Видалити", "Закрити", "Назва", "Закладок ще немає.", "Закладку збережено.", "Закладку видалено."},
	"sv": {"Bokmärken", "Lägg till lokal", "Lägg till fjärr", "Öppna", "Ta bort", "Stäng", "Namn", "Inga bokmärken ännu.", "Bokmärket sparades.", "Bokmärket togs bort."},
	"ro": {"Marcaje", "Adaugă local", "Adaugă la distanță", "Deschide", "Șterge", "Închide", "Nume", "Nu există încă marcaje.", "Marcaj salvat.", "Marcaj eliminat."},
	"hu": {"Könyvjelzők", "Helyi hozzáadása", "Távoli hozzáadása", "Megnyitás", "Törlés", "Bezárás", "Név", "Még nincsenek könyvjelzők.", "Könyvjelző mentve.", "Könyvjelző eltávolítva."},
	"da": {"Bogmærker", "Tilføj lokal", "Tilføj ekstern", "Åbn", "Slet", "Luk", "Navn", "Ingen bogmærker endnu.", "Bogmærke gemt.", "Bogmærke fjernet."},
	"fi": {"Kirjanmerkit", "Lisää paikallinen", "Lisää etä", "Avaa", "Poista", "Sulje", "Nimi", "Ei vielä kirjanmerkkejä.", "Kirjanmerkki tallennettu.", "Kirjanmerkki poistettu."},
	"no": {"Bokmerker", "Legg til lokal", "Legg til ekstern", "Åpne", "Slett", "Lukk", "Navn", "Ingen bokmerker ennå.", "Bokmerke lagret.", "Bokmerke fjernet."},
	"ko": {"북마크", "로컬 추가", "원격 추가", "열기", "삭제", "닫기", "이름", "아직 북마크가 없습니다.", "북마크가 저장되었습니다.", "북마크가 삭제되었습니다."},
}

func bookmarkWordsForLanguage(language string) bookmarkWords {
	if words, ok := bookmarkWordCatalog[i18n.Normalize(language)]; ok {
		return words
	}
	return bookmarkWordCatalog[i18n.DefaultLanguage]
}
