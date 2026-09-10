package desktop

import (
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

type queuePriorityAction int

const (
	queuePriorityTop queuePriorityAction = iota
	queuePriorityUp
	queuePriorityDown
	queuePriorityBottom
)

type queuePriorityState struct {
	MoveTop    bool
	MoveUp     bool
	MoveDown   bool
	MoveBottom bool
}

// deriveQueuePriorityState is shared by Windows and Linux. Reordering is
// intentionally single-selection and queued-only; running and terminal jobs
// are immutable from all priority controls.
func deriveQueuePriorityState(jobs []model.TransferJob, selected []int) queuePriorityState {
	if len(selected) != 1 {
		return queuePriorityState{}
	}
	index := selected[0]
	if index < 0 || index >= len(jobs) || jobs[index].Status != "queued" {
		return queuePriorityState{}
	}

	state := queuePriorityState{}
	for i := index - 1; i >= 0; i-- {
		if jobs[i].Status == "queued" {
			state.MoveTop = true
			state.MoveUp = true
			break
		}
	}
	for i := index + 1; i < len(jobs); i++ {
		if jobs[i].Status == "queued" {
			state.MoveDown = true
			state.MoveBottom = true
			break
		}
	}
	return state
}

type queuePriorityText struct {
	MoveTop     string
	MoveUp      string
	MoveDown    string
	MoveBottom  string
	MovedTop    string
	MovedUp     string
	MovedDown   string
	MovedBottom string
}

var queuePriorityTranslations = map[string]queuePriorityText{
	"en": {"Top", "Move up", "Move down", "Bottom", "Transfer moved to top.", "Transfer moved up.", "Transfer moved down.", "Transfer moved to bottom."},
	"hr": {"Na vrh", "Pomakni gore", "Pomakni dolje", "Na dno", "Prijenos je pomaknut na vrh.", "Prijenos je pomaknut gore.", "Prijenos je pomaknut dolje.", "Prijenos je pomaknut na dno."},
	"de": {"Ganz nach oben", "Nach oben", "Nach unten", "Ganz nach unten", "Übertragung ganz nach oben verschoben.", "Übertragung nach oben verschoben.", "Übertragung nach unten verschoben.", "Übertragung ganz nach unten verschoben."},
	"fr": {"Tout en haut", "Monter", "Descendre", "Tout en bas", "Transfert déplacé tout en haut.", "Transfert déplacé vers le haut.", "Transfert déplacé vers le bas.", "Transfert déplacé tout en bas."},
	"es": {"Al inicio", "Subir", "Bajar", "Al final", "Transferencia movida al inicio.", "Transferencia movida hacia arriba.", "Transferencia movida hacia abajo.", "Transferencia movida al final."},
	"tr": {"En üste", "Yukarı taşı", "Aşağı taşı", "En alta", "Aktarım en üste taşındı.", "Aktarım yukarı taşındı.", "Aktarım aşağı taşındı.", "Aktarım en alta taşındı."},
	"el": {"Στην κορυφή", "Μετακίνηση πάνω", "Μετακίνηση κάτω", "Στο τέλος", "Η μεταφορά μετακινήθηκε στην κορυφή.", "Η μεταφορά μετακινήθηκε πάνω.", "Η μεταφορά μετακινήθηκε κάτω.", "Η μεταφορά μετακινήθηκε στο τέλος."},
	"pt": {"Para o topo", "Mover para cima", "Mover para baixo", "Para o fim", "Transferência movida para o topo.", "Transferência movida para cima.", "Transferência movida para baixo.", "Transferência movida para o fim."},
	"zh": {"移到顶部", "上移", "下移", "移到底部", "传输已移到顶部。", "传输已上移。", "传输已下移。", "传输已移到底部。"},
	"ru": {"В начало", "Переместить вверх", "Переместить вниз", "В конец", "Передача перемещена в начало.", "Передача перемещена вверх.", "Передача перемещена вниз.", "Передача перемещена в конец."},
	"hi": {"सबसे ऊपर", "ऊपर ले जाएँ", "नीचे ले जाएँ", "सबसे नीचे", "स्थानांतरण सबसे ऊपर ले जाया गया।", "स्थानांतरण ऊपर ले जाया गया।", "स्थानांतरण नीचे ले जाया गया।", "स्थानांतरण सबसे नीचे ले जाया गया।"},
	"ja": {"先頭へ", "上へ移動", "下へ移動", "末尾へ", "転送を先頭へ移動しました。", "転送を上へ移動しました。", "転送を下へ移動しました。", "転送を末尾へ移動しました。"},
	"it": {"In cima", "Sposta su", "Sposta giù", "In fondo", "Trasferimento spostato in cima.", "Trasferimento spostato in alto.", "Trasferimento spostato in basso.", "Trasferimento spostato in fondo."},
	"pl": {"Na początek", "Przenieś wyżej", "Przenieś niżej", "Na koniec", "Transfer przeniesiono na początek.", "Transfer przeniesiono wyżej.", "Transfer przeniesiono niżej.", "Transfer przeniesiono na koniec."},
	"nl": {"Naar bovenaan", "Omhoog", "Omlaag", "Naar onderaan", "Overdracht naar bovenaan verplaatst.", "Overdracht omhoog verplaatst.", "Overdracht omlaag verplaatst.", "Overdracht naar onderaan verplaatst."},
	"cs": {"Na začátek", "Posunout nahoru", "Posunout dolů", "Na konec", "Přenos byl přesunut na začátek.", "Přenos byl posunut nahoru.", "Přenos byl posunut dolů.", "Přenos byl přesunut na konec."},
	"uk": {"На початок", "Перемістити вгору", "Перемістити вниз", "У кінець", "Передачу переміщено на початок.", "Передачу переміщено вгору.", "Передачу переміщено вниз.", "Передачу переміщено в кінець."},
	"sv": {"Överst", "Flytta upp", "Flytta ner", "Nederst", "Överföringen flyttades överst.", "Överföringen flyttades upp.", "Överföringen flyttades ner.", "Överföringen flyttades nederst."},
	"ro": {"La început", "Mută în sus", "Mută în jos", "La sfârșit", "Transferul a fost mutat la început.", "Transferul a fost mutat în sus.", "Transferul a fost mutat în jos.", "Transferul a fost mutat la sfârșit."},
	"hu": {"Legfelülre", "Mozgatás fel", "Mozgatás le", "Legalulra", "Az átvitel a sor elejére került.", "Az átvitel feljebb került.", "Az átvitel lejjebb került.", "Az átvitel a sor végére került."},
	"da": {"Øverst", "Flyt op", "Flyt ned", "Nederst", "Overførslen blev flyttet øverst.", "Overførslen blev flyttet op.", "Overførslen blev flyttet ned.", "Overførslen blev flyttet nederst."},
	"fi": {"Ylimmäksi", "Siirrä ylös", "Siirrä alas", "Alimmaksi", "Siirto siirrettiin ylimmäksi.", "Siirto siirrettiin ylöspäin.", "Siirto siirrettiin alaspäin.", "Siirto siirrettiin alimmaksi."},
	"no": {"Øverst", "Flytt opp", "Flytt ned", "Nederst", "Overføringen ble flyttet øverst.", "Overføringen ble flyttet opp.", "Overføringen ble flyttet ned.", "Overføringen ble flyttet nederst."},
	"ko": {"맨 위로", "위로 이동", "아래로 이동", "맨 아래로", "전송을 맨 위로 이동했습니다.", "전송을 위로 이동했습니다.", "전송을 아래로 이동했습니다.", "전송을 맨 아래로 이동했습니다."},
}

func queuePriorityWords(language string) queuePriorityText {
	code := i18n.Normalize(strings.TrimSpace(language))
	if text, ok := queuePriorityTranslations[code]; ok {
		return text
	}
	return queuePriorityTranslations[i18n.DefaultLanguage]
}
