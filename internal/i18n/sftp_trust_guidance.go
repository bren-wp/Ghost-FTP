package i18n

// Security-sensitive first-contact SFTP wording is kept in one complete table
// instead of being allowed to drift between the desktop confirmation dialog and
// terminal prompt. Every message tells the user to verify the host-key
// fingerprint through an independent trusted channel before accepting it.
type sftpTrustGuidanceText struct {
	body     string
	terminal string
}

var sftpTrustGuidance = map[string]sftpTrustGuidanceText{
	"en": {
		body:     "Verify this server key fingerprint through an independent trusted channel (for example, your hosting control panel or administrator) before accepting it:\n\n%s\n\nAccept only if it matches exactly. Do you trust this server?",
		terminal: "Verify the fingerprint through an independent trusted channel. Trust this server only if it matches exactly. (yes/no)",
	},
	"hr": {
		body:     "Prije prihvaćanja provjerite otisak ključa poslužitelja preko neovisnog pouzdanog kanala (npr. upravljačke ploče hostinga ili administratora):\n\n%s\n\nPrihvatite samo ako se potpuno podudara. Vjerujete li ovom poslužitelju?",
		terminal: "Provjerite otisak preko neovisnog pouzdanog kanala. Vjerujte poslužitelju samo ako se potpuno podudara. (da/ne)",
	},
	"de": {
		body:     "Prüfen Sie diesen Server-Schlüssel-Fingerabdruck vor dem Akzeptieren über einen unabhängigen vertrauenswürdigen Kanal (z. B. Hosting-Kontrollpanel oder Administrator):\n\n%s\n\nAkzeptieren Sie ihn nur bei exakter Übereinstimmung. Vertrauen Sie diesem Server?",
		terminal: "Prüfen Sie den Fingerabdruck über einen unabhängigen vertrauenswürdigen Kanal. Vertrauen Sie dem Server nur bei exakter Übereinstimmung. (ja/nein)",
	},
	"fr": {
		body:     "Avant de l'accepter, vérifiez cette empreinte de clé serveur via un canal de confiance indépendant (par exemple le panneau d'hébergement ou l'administrateur) :\n\n%s\n\nAcceptez-la uniquement si elle correspond exactement. Faites-vous confiance à ce serveur ?",
		terminal: "Vérifiez l'empreinte via un canal de confiance indépendant. N'accordez votre confiance que si elle correspond exactement. (oui/non)",
	},
	"es": {
		body:     "Antes de aceptarla, verifique esta huella de clave del servidor mediante un canal de confianza independiente (por ejemplo, el panel de hosting o el administrador):\n\n%s\n\nAcéptela solo si coincide exactamente. ¿Confía en este servidor?",
		terminal: "Verifique la huella mediante un canal de confianza independiente. Confíe en el servidor solo si coincide exactamente. (sí/no)",
	},
	"tr": {
		body:     "Kabul etmeden önce bu sunucu anahtarı parmak izini bağımsız ve güvenilir bir kanaldan (ör. hosting kontrol paneli veya yönetici) doğrulayın:\n\n%s\n\nYalnızca tam olarak eşleşiyorsa kabul edin. Bu sunucuya güveniyor musunuz?",
		terminal: "Parmak izini bağımsız ve güvenilir bir kanaldan doğrulayın. Yalnızca tam eşleşiyorsa sunucuya güvenin. (evet/hayır)",
	},
	"el": {
		body:     "Πριν την αποδεχθείτε, επαληθεύστε αυτό το αποτύπωμα κλειδιού διακομιστή μέσω ανεξάρτητου αξιόπιστου καναλιού (π.χ. πίνακα φιλοξενίας ή διαχειριστή):\n\n%s\n\nΑποδεχθείτε το μόνο αν ταιριάζει ακριβώς. Εμπιστεύεστε αυτόν τον διακομιστή;",
		terminal: "Επαληθεύστε το αποτύπωμα μέσω ανεξάρτητου αξιόπιστου καναλιού. Εμπιστευτείτε τον διακομιστή μόνο αν ταιριάζει ακριβώς. (ναι/όχι)",
	},
	"pt": {
		body:     "Antes de aceitar, verifique esta impressão digital da chave do servidor por um canal independente e confiável (por exemplo, painel de alojamento ou administrador):\n\n%s\n\nAceite apenas se corresponder exatamente. Confia neste servidor?",
		terminal: "Verifique a impressão digital por um canal independente e confiável. Confie no servidor apenas se corresponder exatamente. (sim/não)",
	},
	"zh": {
		body:     "接受前，请通过独立且可信的渠道（例如主机控制面板或管理员）核对此服务器密钥指纹：\n\n%s\n\n仅在完全一致时接受。是否信任此服务器？",
		terminal: "请通过独立且可信的渠道核对指纹。仅在完全一致时信任此服务器。（是/否）",
	},
	"ru": {
		body:     "Перед принятием проверьте этот отпечаток ключа сервера через независимый доверенный канал (например, панель хостинга или администратора):\n\n%s\n\nПринимайте только при полном совпадении. Доверяете этому серверу?",
		terminal: "Проверьте отпечаток через независимый доверенный канал. Доверяйте серверу только при полном совпадении. (да/нет)",
	},
	"hi": {
		body:     "स्वीकार करने से पहले इस सर्वर कुंजी फ़िंगरप्रिंट को किसी स्वतंत्र विश्वसनीय माध्यम (जैसे होस्टिंग कंट्रोल पैनल या व्यवस्थापक) से सत्यापित करें:\n\n%s\n\nकेवल पूर्ण मिलान होने पर स्वीकार करें। क्या आप इस सर्वर पर भरोसा करते हैं?",
		terminal: "फ़िंगरप्रिंट को किसी स्वतंत्र विश्वसनीय माध्यम से सत्यापित करें। केवल पूर्ण मिलान होने पर सर्वर पर भरोसा करें। (हाँ/नहीं)",
	},
	"ja": {
		body:     "受け入れる前に、このサーバー鍵フィンガープリントを独立した信頼できる経路（例: ホスティング管理画面または管理者）で確認してください:\n\n%s\n\n完全に一致する場合のみ受け入れてください。このサーバーを信頼しますか？",
		terminal: "フィンガープリントを独立した信頼できる経路で確認してください。完全に一致する場合のみサーバーを信頼してください。(はい/いいえ)",
	},
	"it": {
		body:     "Prima di accettarla, verifica questa impronta della chiave del server tramite un canale affidabile indipendente (ad esempio il pannello di hosting o l'amministratore):\n\n%s\n\nAccetta solo se corrisponde esattamente. Vuoi considerare attendibile questo server?",
		terminal: "Verifica l'impronta tramite un canale affidabile indipendente. Considera attendibile il server solo se corrisponde esattamente. (sì/no)",
	},
	"pl": {
		body:     "Przed zaakceptowaniem sprawdź ten odcisk klucza serwera niezależnym zaufanym kanałem (np. w panelu hostingu lub u administratora):\n\n%s\n\nZaakceptuj tylko przy dokładnej zgodności. Czy ufasz temu serwerowi?",
		terminal: "Sprawdź odcisk niezależnym zaufanym kanałem. Zaufaj serwerowi tylko przy dokładnej zgodności. (tak/nie)",
	},
	"nl": {
		body:     "Controleer deze vingerafdruk van de serversleutel vóór acceptatie via een onafhankelijk vertrouwd kanaal (bijvoorbeeld het hostingpaneel of de beheerder):\n\n%s\n\nAccepteer alleen bij een exacte overeenkomst. Vertrouwt u deze server?",
		terminal: "Controleer de vingerafdruk via een onafhankelijk vertrouwd kanaal. Vertrouw de server alleen bij een exacte overeenkomst. (ja/nee)",
	},
	"cs": {
		body:     "Před přijetím ověřte tento otisk klíče serveru nezávislým důvěryhodným kanálem (např. v panelu hostingu nebo u správce):\n\n%s\n\nPřijměte jej pouze při přesné shodě. Důvěřujete tomuto serveru?",
		terminal: "Ověřte otisk nezávislým důvěryhodným kanálem. Serveru důvěřujte pouze při přesné shodě. (ano/ne)",
	},
	"uk": {
		body:     "Перед прийняттям перевірте цей відбиток ключа сервера через незалежний довірений канал (наприклад, панель хостингу або адміністратора):\n\n%s\n\nПриймайте лише за точного збігу. Довіряєте цьому серверу?",
		terminal: "Перевірте відбиток через незалежний довірений канал. Довіряйте серверу лише за точного збігу. (так/ні)",
	},
	"sv": {
		body:     "Verifiera servernyckelns fingeravtryck via en oberoende betrodd kanal (t.ex. hostingpanelen eller administratören) innan du godkänner det:\n\n%s\n\nGodkänn endast om det stämmer exakt. Litar du på den här servern?",
		terminal: "Verifiera fingeravtrycket via en oberoende betrodd kanal. Lita på servern endast om det stämmer exakt. (ja/nej)",
	},
	"ro": {
		body:     "Înainte de acceptare, verificați această amprentă a cheii serverului printr-un canal de încredere independent (de exemplu, panoul de găzduire sau administratorul):\n\n%s\n\nAcceptați numai dacă se potrivește exact. Aveți încredere în acest server?",
		terminal: "Verificați amprenta printr-un canal de încredere independent. Aveți încredere în server numai dacă se potrivește exact. (da/nu)",
	},
	"hu": {
		body:     "Elfogadás előtt ellenőrizze ezt a szerverkulcs-ujjlenyomatot független, megbízható csatornán (például a tárhely vezérlőpultján vagy a rendszergazdánál):\n\n%s\n\nCsak pontos egyezés esetén fogadja el. Megbízik ebben a szerverben?",
		terminal: "Ellenőrizze az ujjlenyomatot független, megbízható csatornán. Csak pontos egyezés esetén bízzon meg a szerverben. (igen/nem)",
	},
	"da": {
		body:     "Kontrollér dette fingeraftryk for servernøglen via en uafhængig, betroet kanal (f.eks. hostingkontrolpanelet eller administratoren), før du accepterer det:\n\n%s\n\nAcceptér kun ved nøjagtigt match. Har du tillid til denne server?",
		terminal: "Kontrollér fingeraftrykket via en uafhængig, betroet kanal. Hav kun tillid til serveren ved nøjagtigt match. (ja/nej)",
	},
	"fi": {
		body:     "Varmista tämä palvelinavaimen sormenjälki riippumattoman luotetun kanavan kautta (esim. hosting-hallintapaneelista tai ylläpitäjältä) ennen hyväksymistä:\n\n%s\n\nHyväksy vain, jos se täsmää tarkasti. Luotatko tähän palvelimeen?",
		terminal: "Varmista sormenjälki riippumattoman luotetun kanavan kautta. Luota palvelimeen vain, jos se täsmää tarkasti. (kyllä/ei)",
	},
	"no": {
		body:     "Kontroller dette fingeravtrykket for servernøkkelen via en uavhengig, pålitelig kanal (for eksempel hostingpanelet eller administratoren) før du godtar det:\n\n%s\n\nGodta bare hvis det samsvarer nøyaktig. Stoler du på denne serveren?",
		terminal: "Kontroller fingeravtrykket via en uavhengig, pålitelig kanal. Stol bare på serveren hvis det samsvarer nøyaktig. (ja/nei)",
	},
	"ko": {
		body:     "수락하기 전에 독립적인 신뢰 채널(예: 호스팅 제어판 또는 관리자)을 통해 이 서버 키 지문을 확인하세요:\n\n%s\n\n정확히 일치할 때만 수락하세요. 이 서버를 신뢰하시겠습니까?",
		terminal: "독립적인 신뢰 채널을 통해 지문을 확인하세요. 정확히 일치할 때만 서버를 신뢰하세요. (예/아니요)",
	},
}

func sftpTrustText(language, key string) (string, bool) {
	guidance, ok := sftpTrustGuidance[Normalize(language)]
	if !ok {
		guidance, ok = sftpTrustGuidance[DefaultLanguage]
	}
	if !ok {
		return "", false
	}
	switch key {
	case "sftp.trust_body":
		return guidance.body, true
	case "terminal.trust":
		return guidance.terminal, true
	default:
		return "", false
	}
}
