package desktop

import (
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

type queuePriorityState struct {
	MoveUp   bool
	MoveDown bool
}

// deriveQueuePriorityState is shared by Windows and Linux. Reordering is
// intentionally single-selection and queued-only; running and terminal jobs
// are immutable from the priority controls.
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
			state.MoveUp = true
			break
		}
	}
	for i := index + 1; i < len(jobs); i++ {
		if jobs[i].Status == "queued" {
			state.MoveDown = true
			break
		}
	}
	return state
}

type queuePriorityText struct {
	MoveUp   string
	MoveDown string
	MovedUp  string
	MovedDown string
}

var queuePriorityTranslations = map[string]queuePriorityText{
	"en": {"Move up", "Move down", "Transfer moved up.", "Transfer moved down."},
	"hr": {"Pomakni gore", "Pomakni dolje", "Prijenos je pomaknut gore.", "Prijenos je pomaknut dolje."},
	"de": {"Nach oben", "Nach unten", "Übertragung nach oben verschoben.", "Übertragung nach unten verschoben."},
	"fr": {"Monter", "Descendre", "Transfert déplacé vers le haut.", "Transfert déplacé vers le bas."},
	"es": {"Subir", "Bajar", "Transferencia movida hacia arriba.", "Transferencia movida hacia abajo."},
	"tr": {"Yukarı taşı", "Aşağı taşı", "Aktarım yukarı taşındı.", "Aktarım aşağı taşındı."},
	"el": {"Μετακίνηση πάνω", "Μετακίνηση κάτω", "Η μεταφορά μετακινήθηκε πάνω.", "Η μεταφορά μετακινήθηκε κάτω."},
	"pt": {"Mover para cima", "Mover para baixo", "Transferência movida para cima.", "Transferência movida para baixo."},
	"zh": {"上移", "下移", "传输已上移。", "传输已下移。"},
	"ru": {"Переместить вверх", "Переместить вниз", "Передача перемещена вверх.", "Передача перемещена вниз."},
	"hi": {"ऊपर ले जाएँ", "नीचे ले जाएँ", "स्थानांतरण ऊपर ले जाया गया।", "स्थानांतरण नीचे ले जाया गया।"},
	"ja": {"上へ移動", "下へ移動", "転送を上へ移動しました。", "転送を下へ移動しました。"},
	"it": {"Sposta su", "Sposta giù", "Trasferimento spostato in alto.", "Trasferimento spostato in basso."},
	"pl": {"Przenieś wyżej", "Przenieś niżej", "Transfer przeniesiono wyżej.", "Transfer przeniesiono niżej."},
	"nl": {"Omhoog", "Omlaag", "Overdracht omhoog verplaatst.", "Overdracht omlaag verplaatst."},
	"cs": {"Posunout nahoru", "Posunout dolů", "Přenos byl posunut nahoru.", "Přenos byl posunut dolů."},
	"uk": {"Перемістити вгору", "Перемістити вниз", "Передачу переміщено вгору.", "Передачу переміщено вниз."},
	"sv": {"Flytta upp", "Flytta ner", "Överföringen flyttades upp.", "Överföringen flyttades ner."},
	"ro": {"Mută în sus", "Mută în jos", "Transferul a fost mutat în sus.", "Transferul a fost mutat în jos."},
	"hu": {"Mozgatás fel", "Mozgatás le", "Az átvitel feljebb került.", "Az átvitel lejjebb került."},
	"da": {"Flyt op", "Flyt ned", "Overførslen blev flyttet op.", "Overførslen blev flyttet ned."},
	"fi": {"Siirrä ylös", "Siirrä alas", "Siirto siirrettiin ylöspäin.", "Siirto siirrettiin alaspäin."},
	"no": {"Flytt opp", "Flytt ned", "Overføringen ble flyttet opp.", "Overføringen ble flyttet ned."},
	"ko": {"위로 이동", "아래로 이동", "전송을 위로 이동했습니다.", "전송을 아래로 이동했습니다."},
}

func queuePriorityWords(language string) queuePriorityText {
	code := i18n.Normalize(strings.TrimSpace(language))
	if text, ok := queuePriorityTranslations[code]; ok {
		return text
	}
	return queuePriorityTranslations[i18n.DefaultLanguage]
}
