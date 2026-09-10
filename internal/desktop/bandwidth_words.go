package desktop

import "github.com/bren-wp/Ghost-FTP/internal/i18n"

type bandwidthWords struct {
	UploadLabel   string
	DownloadLabel string
}

var bandwidthTranslations = map[string]bandwidthWords{
	"en": {"Upload limit (KiB/s; 0 = unlimited)", "Download limit (KiB/s; 0 = unlimited)"},
	"hr": {"Ograničenje slanja (KiB/s; 0 = neograničeno)", "Ograničenje preuzimanja (KiB/s; 0 = neograničeno)"},
	"de": {"Upload-Limit (KiB/s; 0 = unbegrenzt)", "Download-Limit (KiB/s; 0 = unbegrenzt)"},
	"fr": {"Limite d’envoi (KiB/s ; 0 = illimité)", "Limite de téléchargement (KiB/s ; 0 = illimité)"},
	"es": {"Límite de subida (KiB/s; 0 = sin límite)", "Límite de descarga (KiB/s; 0 = sin límite)"},
	"tr": {"Yükleme sınırı (KiB/s; 0 = sınırsız)", "İndirme sınırı (KiB/s; 0 = sınırsız)"},
	"el": {"Όριο αποστολής (KiB/s· 0 = απεριόριστο)", "Όριο λήψης (KiB/s· 0 = απεριόριστο)"},
	"pt": {"Limite de envio (KiB/s; 0 = ilimitado)", "Limite de download (KiB/s; 0 = ilimitado)"},
	"zh": {"上传限制（KiB/s；0 = 不限速）", "下载限制（KiB/s；0 = 不限速）"},
	"ru": {"Лимит отдачи (KiB/s; 0 = без ограничений)", "Лимит загрузки (KiB/s; 0 = без ограничений)"},
	"hi": {"अपलोड सीमा (KiB/s; 0 = असीमित)", "डाउनलोड सीमा (KiB/s; 0 = असीमित)"},
	"ja": {"アップロード上限 (KiB/s; 0 = 無制限)", "ダウンロード上限 (KiB/s; 0 = 無制限)"},
	"it": {"Limite upload (KiB/s; 0 = illimitato)", "Limite download (KiB/s; 0 = illimitato)"},
	"pl": {"Limit wysyłania (KiB/s; 0 = bez limitu)", "Limit pobierania (KiB/s; 0 = bez limitu)"},
	"nl": {"Uploadlimiet (KiB/s; 0 = onbeperkt)", "Downloadlimiet (KiB/s; 0 = onbeperkt)"},
	"cs": {"Limit odesílání (KiB/s; 0 = bez omezení)", "Limit stahování (KiB/s; 0 = bez omezení)"},
	"uk": {"Ліміт віддачі (KiB/s; 0 = без обмежень)", "Ліміт завантаження (KiB/s; 0 = без обмежень)"},
	"sv": {"Uppladdningsgräns (KiB/s; 0 = obegränsat)", "Nedladdningsgräns (KiB/s; 0 = obegränsat)"},
	"ro": {"Limită încărcare (KiB/s; 0 = nelimitat)", "Limită descărcare (KiB/s; 0 = nelimitat)"},
	"hu": {"Feltöltési korlát (KiB/s; 0 = korlátlan)", "Letöltési korlát (KiB/s; 0 = korlátlan)"},
	"da": {"Uploadgrænse (KiB/s; 0 = ubegrænset)", "Downloadgrænse (KiB/s; 0 = ubegrænset)"},
	"fi": {"Lähetysraja (KiB/s; 0 = rajoittamaton)", "Latausraja (KiB/s; 0 = rajoittamaton)"},
	"no": {"Opplastingsgrense (KiB/s; 0 = ubegrenset)", "Nedlastingsgrense (KiB/s; 0 = ubegrenset)"},
	"ko": {"업로드 제한 (KiB/s; 0 = 무제한)", "다운로드 제한 (KiB/s; 0 = 무제한)"},
}

func bandwidthWordsForLanguage(language string) bandwidthWords {
	if words, ok := bandwidthTranslations[i18n.Normalize(language)]; ok {
		return words
	}
	return bandwidthTranslations[i18n.DefaultLanguage]
}
