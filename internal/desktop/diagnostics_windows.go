//go:build windows

package desktop

import (
	"fmt"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

type diagnosticsWordsSet struct {
	PrivacyBody   string
	RemoteOffline string
}

var diagnosticsWords = map[string]diagnosticsWordsSet{
	"en": {"No telemetry or tracking. Saved profiles stay on this computer.", "Not connected"},
	"hr": {"Bez telemetrije i praćenja. Spremljeni profili ostaju na ovom računalu.", "Nije povezano"},
	"de": {"Keine Telemetrie oder Verfolgung. Gespeicherte Profile bleiben auf diesem Computer.", "Nicht verbunden"},
	"fr": {"Aucune télémétrie ni suivi. Les profils enregistrés restent sur cet ordinateur.", "Non connecté"},
	"es": {"Sin telemetría ni seguimiento. Los perfiles guardados permanecen en este equipo.", "Sin conexión"},
	"tr": {"Telemetri veya izleme yok. Kaydedilen profiller bu bilgisayarda kalır.", "Bağlı değil"},
	"el": {"Χωρίς τηλεμετρία ή παρακολούθηση. Τα αποθηκευμένα προφίλ μένουν σε αυτόν τον υπολογιστή.", "Δεν υπάρχει σύνδεση"},
	"pt": {"Sem telemetria ou rastreio. Os perfis guardados ficam neste computador.", "Não ligado"},
	"zh": {"无遥测或跟踪。已保存的配置保留在此计算机上。", "未连接"},
	"ru": {"Без телеметрии и отслеживания. Сохранённые профили остаются на этом компьютере.", "Не подключено"},
	"hi": {"कोई टेलीमेट्री या ट्रैकिंग नहीं। सहेजे गए प्रोफ़ाइल इसी कंप्यूटर पर रहते हैं।", "कनेक्ट नहीं"},
	"ja": {"テレメトリや追跡はありません。保存したプロファイルはこのコンピューター内に残ります。", "未接続"},
	"it": {"Nessuna telemetria o tracciamento. I profili salvati restano su questo computer.", "Non connesso"},
	"pl": {"Brak telemetrii i śledzenia. Zapisane profile pozostają na tym komputerze.", "Brak połączenia"},
	"nl": {"Geen telemetrie of tracking. Opgeslagen profielen blijven op deze computer.", "Niet verbonden"},
	"cs": {"Bez telemetrie a sledování. Uložené profily zůstávají v tomto počítači.", "Nepřipojeno"},
	"uk": {"Без телеметрії та відстеження. Збережені профілі залишаються на цьому комп’ютері.", "Не підключено"},
	"sv": {"Ingen telemetri eller spårning. Sparade profiler stannar på den här datorn.", "Inte ansluten"},
	"ro": {"Fără telemetrie sau urmărire. Profilurile salvate rămân pe acest computer.", "Neconectat"},
	"hu": {"Nincs telemetria vagy követés. A mentett profilok ezen a számítógépen maradnak.", "Nincs kapcsolat"},
	"da": {"Ingen telemetri eller sporing. Gemte profiler forbliver på denne computer.", "Ikke forbundet"},
	"fi": {"Ei telemetriaa tai seurantaa. Tallennetut profiilit pysyvät tällä tietokoneella.", "Ei yhteyttä"},
	"no": {"Ingen telemetri eller sporing. Lagrede profiler forblir på denne datamaskinen.", "Ikke tilkoblet"},
	"ko": {"원격 측정이나 추적이 없습니다. 저장된 프로필은 이 컴퓨터에만 남습니다.", "연결되지 않음"},
}

func (a *app) showDiagnostics() {
	if a == nil {
		return
	}
	words, ok := diagnosticsWords[a.languageCode()]
	if !ok {
		words = diagnosticsWords["en"]
	}
	state := words.RemoteOffline
	if a.connected {
		state = a.tr("badge.connected")
	}
	title := nativeMenuWords(a.languageCode())[8]
	body := fmt.Sprintf(
		"Ghost FTP %s\n\n%s · %s\n%s\n\n%s",
		a.version, strings.ToUpper(a.protocolValue()), state, a.remoteCurrent, words.PrivacyBody,
	)
	platform.InfoDialog("Ghost FTP — "+title, title, body)
}
