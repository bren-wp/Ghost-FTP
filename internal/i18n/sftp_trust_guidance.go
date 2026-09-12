package i18n

// sftpTrustGuidance centralizes security-critical first-contact copy so every
// supported locale tells users to verify an unknown SFTP host key through an
// independent trusted source before accepting it. Keeping this policy in one
// place prevents locale drift from silently weakening TOFU guidance.
type sftpTrustCopy struct {
	dialog   string
	terminal string
}

var sftpTrustGuidance = map[string]sftpTrustCopy{
	"en": {
		dialog:   "Before accepting, verify this server key fingerprint through an independent trusted source (for example, your hosting control panel or server administrator):\n\n%s\n\nAccept only if it matches exactly. Do you trust this server?",
		terminal: "Verify the fingerprint through an independent trusted source and accept only if it matches exactly. Trust this server? (yes/no)",
	},
	"hr": {
		dialog:   "Prije prihvaćanja provjerite ovaj otisak ključa poslužitelja putem neovisnog pouzdanog izvora (na primjer, upravljačke ploče hostinga ili administratora poslužitelja):\n\n%s\n\nPrihvatite samo ako se potpuno podudara. Vjerujete li ovom poslužitelju?",
		terminal: "Provjerite otisak putem neovisnog pouzdanog izvora i prihvatite samo ako se potpuno podudara. Vjerujete li ovom poslužitelju? (da/ne)",
	},
	"de": {
		dialog:   "Prüfen Sie vor der Annahme diesen Fingerabdruck des Serverschlüssels über eine unabhängige vertrauenswürdige Quelle (z. B. das Hosting-Control-Panel oder den Serveradministrator):\n\n%s\n\nAkzeptieren Sie ihn nur bei exakter Übereinstimmung. Vertrauen Sie diesem Server?",
		terminal: "Prüfen Sie den Fingerabdruck über eine unabhängige vertrauenswürdige Quelle und akzeptieren Sie ihn nur bei exakter Übereinstimmung. Diesem Server vertrauen? (ja/nein)",
	},
	"fr": {
		dialog:   "Avant d’accepter, vérifiez cette empreinte de clé du serveur auprès d’une source fiable et indépendante (par exemple, le panneau de contrôle de votre hébergement ou l’administrateur du serveur) :\n\n%s\n\nN’acceptez que si elle correspond exactement. Faites-vous confiance à ce serveur ?",
		terminal: "Vérifiez l’empreinte auprès d’une source fiable et indépendante et n’acceptez que si elle correspond exactement. Faire confiance à ce serveur ? (oui/non)",
	},
	"es": {
		dialog:   "Antes de aceptar, verifique esta huella digital de la clave del servidor mediante una fuente independiente y de confianza (por ejemplo, el panel de control de su alojamiento o el administrador del servidor):\n\n%s\n\nAcepte solo si coincide exactamente. ¿Confía en este servidor?",
		terminal: "Verifique la huella mediante una fuente independiente y de confianza y acepte solo si coincide exactamente. ¿Confiar en este servidor? (sí/no)",
	},
	"tr": {
		dialog:   "Kabul etmeden önce bu sunucu anahtarı parmak izini bağımsız ve güvenilir bir kaynaktan (örneğin hosting kontrol panelinizden veya sunucu yöneticinizden) doğrulayın:\n\n%s\n\nYalnızca tam olarak eşleşiyorsa kabul edin. Bu sunucuya güveniyor musunuz?",
		terminal: "Parmak izini bağımsız ve güvenilir bir kaynaktan doğrulayın ve yalnızca tam olarak eşleşiyorsa kabul edin. Bu sunucuya güvenilsin mi? (evet/hayır)",
	},
	"el": {
		dialog:   "Πριν την αποδοχή, επαληθεύστε αυτό το αποτύπωμα κλειδιού διακομιστή μέσω ανεξάρτητης αξιόπιστης πηγής (π.χ. του πίνακα ελέγχου φιλοξενίας ή του διαχειριστή του διακομιστή):\n\n%s\n\nΑποδεχτείτε το μόνο αν ταιριάζει ακριβώς. Εμπιστεύεστε αυτόν τον διακομιστή;",
		terminal: "Επαληθεύστε το αποτύπωμα μέσω ανεξάρτητης αξιόπιστης πηγής και αποδεχτείτε το μόνο αν ταιριάζει ακριβώς. Εμπιστεύεστε αυτόν τον διακομιστή; (ναι/όχι)",
	},
	"pt": {
		dialog:   "Antes de aceitar, verifique esta impressão digital da chave do servidor por meio de uma fonte independente e confiável (por exemplo, o painel de controle da hospedagem ou o administrador do servidor):\n\n%s\n\nAceite somente se corresponder exatamente. Você confia neste servidor?",
		terminal: "Verifique a impressão digital por meio de uma fonte independente e confiável e aceite somente se corresponder exatamente. Confiar neste servidor? (sim/não)",
	},
	"zh": {
		dialog:   "接受前，请通过独立且可信的来源（例如主机控制面板或服务器管理员）核对此服务器密钥指纹：\n\n%s\n\n仅在完全匹配时接受。您信任此服务器吗？",
		terminal: "请通过独立且可信的来源核对指纹，并仅在完全匹配时接受。信任此服务器吗？（是/否）",
	},
	"ru": {
		dialog:   "Перед принятием проверьте этот отпечаток ключа сервера по независимому доверенному источнику (например, в панели управления хостингом или у администратора сервера):\n\n%s\n\nПринимайте только при точном совпадении. Вы доверяете этому серверу?",
		terminal: "Проверьте отпечаток по независимому доверенному источнику и принимайте только при точном совпадении. Доверять этому серверу? (да/нет)",
	},
	"hi": {
		dialog:   "स्वीकार करने से पहले इस सर्वर कुंजी फ़िंगरप्रिंट को किसी स्वतंत्र और विश्वसनीय स्रोत (जैसे आपके होस्टिंग कंट्रोल पैनल या सर्वर प्रशासक) से सत्यापित करें:\n\n%s\n\nकेवल तभी स्वीकार करें जब यह बिल्कुल मेल खाता हो। क्या आप इस सर्वर पर भरोसा करते हैं?",
		terminal: "फ़िंगरप्रिंट को किसी स्वतंत्र और विश्वसनीय स्रोत से सत्यापित करें और केवल बिल्कुल मेल खाने पर स्वीकार करें। क्या इस सर्वर पर भरोसा करें? (हाँ/नहीं)",
	},
	"ja": {
		dialog:   "受け入れる前に、このサーバー鍵のフィンガープリントを独立した信頼できる情報源（例: ホスティングのコントロールパネルやサーバー管理者）で確認してください:\n\n%s\n\n完全に一致する場合のみ受け入れてください。このサーバーを信頼しますか？",
		terminal: "フィンガープリントを独立した信頼できる情報源で確認し、完全に一致する場合のみ受け入れてください。このサーバーを信頼しますか？ (はい/いいえ)",
	},
	"it": {
		dialog:   "Prima di accettare, verifica questa impronta della chiave del server tramite una fonte indipendente e affidabile (ad esempio, il pannello di controllo dell’hosting o l’amministratore del server):\n\n%s\n\nAccetta solo se corrisponde esattamente. Consideri attendibile questo server?",
		terminal: "Verifica l’impronta tramite una fonte indipendente e affidabile e accetta solo se corrisponde esattamente. Considerare attendibile questo server? (sì/no)",
	},
	"pl": {
		dialog:   "Przed zaakceptowaniem zweryfikuj ten odcisk klucza serwera w niezależnym zaufanym źródle (np. w panelu hostingu lub u administratora serwera):\n\n%s\n\nZaakceptuj tylko wtedy, gdy jest dokładnie zgodny. Czy ufasz temu serwerowi?",
		terminal: "Zweryfikuj odcisk w niezależnym zaufanym źródle i zaakceptuj tylko wtedy, gdy jest dokładnie zgodny. Ufać temu serwerowi? (tak/nie)",
	},
	"nl": {
		dialog:   "Controleer voordat u accepteert deze vingerafdruk van de serversleutel via een onafhankelijke, vertrouwde bron (bijvoorbeeld het hostingcontrolepaneel of de serverbeheerder):\n\n%s\n\nAccepteer alleen als deze exact overeenkomt. Vertrouwt u deze server?",
		terminal: "Controleer de vingerafdruk via een onafhankelijke, vertrouwde bron en accepteer alleen als deze exact overeenkomt. Deze server vertrouwen? (ja/nee)",
	},
	"cs": {
		dialog:   "Před přijetím ověřte tento otisk klíče serveru pomocí nezávislého důvěryhodného zdroje (například ovládacího panelu hostingu nebo správce serveru):\n\n%s\n\nPřijměte jej pouze při přesné shodě. Důvěřujete tomuto serveru?",
		terminal: "Ověřte otisk pomocí nezávislého důvěryhodného zdroje a přijměte jej pouze při přesné shodě. Důvěřovat tomuto serveru? (ano/ne)",
	},
	"uk": {
		dialog:   "Перед прийняттям перевірте цей відбиток ключа сервера через незалежне довірене джерело (наприклад, панель керування хостингом або адміністратора сервера):\n\n%s\n\nПриймайте лише за точного збігу. Ви довіряєте цьому серверу?",
		terminal: "Перевірте відбиток через незалежне довірене джерело та приймайте лише за точного збігу. Довіряти цьому серверу? (так/ні)",
	},
	"sv": {
		dialog:   "Verifiera servernyckelns fingeravtryck via en oberoende betrodd källa (till exempel hostingens kontrollpanel eller serveradministratören) innan du accepterar:\n\n%s\n\nAcceptera bara om det stämmer exakt. Litar du på den här servern?",
		terminal: "Verifiera fingeravtrycket via en oberoende betrodd källa och acceptera bara om det stämmer exakt. Lita på den här servern? (ja/nej)",
	},
	"ro": {
		dialog:   "Înainte de acceptare, verificați această amprentă a cheii serverului printr-o sursă independentă și de încredere (de exemplu, panoul de control al găzduirii sau administratorul serverului):\n\n%s\n\nAcceptați numai dacă se potrivește exact. Aveți încredere în acest server?",
		terminal: "Verificați amprenta printr-o sursă independentă și de încredere și acceptați numai dacă se potrivește exact. Aveți încredere în acest server? (da/nu)",
	},
	"hu": {
		dialog:   "Elfogadás előtt ellenőrizze ezt a szerverkulcs-ujjlenyomatot egy független, megbízható forrásból (például a tárhely vezérlőpultján vagy a szerver rendszergazdájánál):\n\n%s\n\nCsak pontos egyezés esetén fogadja el. Megbízik ebben a szerverben?",
		terminal: "Ellenőrizze az ujjlenyomatot egy független, megbízható forrásból, és csak pontos egyezés esetén fogadja el. Megbízik ebben a szerverben? (igen/nem)",
	},
	"da": {
		dialog:   "Før du accepterer, skal du kontrollere dette fingeraftryk af servernøglen via en uafhængig, betroet kilde (f.eks. hostingkontrolpanelet eller serveradministratoren):\n\n%s\n\nAccepter kun, hvis det matcher nøjagtigt. Har du tillid til denne server?",
		terminal: "Kontrollér fingeraftrykket via en uafhængig, betroet kilde, og accepter kun, hvis det matcher nøjagtigt. Har du tillid til denne server? (ja/nej)",
	},
	"fi": {
		dialog:   "Varmista ennen hyväksymistä tämä palvelinavaimen sormenjälki riippumattomasta luotettavasta lähteestä (esimerkiksi hosting-hallintapaneelista tai palvelimen ylläpitäjältä):\n\n%s\n\nHyväksy vain, jos se täsmää tarkalleen. Luotatko tähän palvelimeen?",
		terminal: "Varmista sormenjälki riippumattomasta luotettavasta lähteestä ja hyväksy vain, jos se täsmää tarkalleen. Luotatko tähän palvelimeen? (kyllä/ei)",
	},
	"no": {
		dialog:   "Før du godtar, må du bekrefte dette fingeravtrykket for servernøkkelen via en uavhengig, betrodd kilde (for eksempel kontrollpanelet for hosting eller serveradministratoren):\n\n%s\n\nGodta bare hvis det samsvarer nøyaktig. Stoler du på denne serveren?",
		terminal: "Bekreft fingeravtrykket via en uavhengig, betrodd kilde og godta bare hvis det samsvarer nøyaktig. Stole på denne serveren? (ja/nei)",
	},
	"ko": {
		dialog:   "수락하기 전에 독립적이고 신뢰할 수 있는 출처(예: 호스팅 제어판 또는 서버 관리자)를 통해 이 서버 키 지문을 확인하세요:\n\n%s\n\n완전히 일치하는 경우에만 수락하세요. 이 서버를 신뢰하시겠습니까?",
		terminal: "독립적이고 신뢰할 수 있는 출처를 통해 지문을 확인하고 완전히 일치하는 경우에만 수락하세요. 이 서버를 신뢰하시겠습니까? (예/아니요)",
	},
}

func init() {
	for code, guidance := range sftpTrustGuidance {
		catalog := catalogs[code]
		if catalog == nil {
			continue
		}
		catalog["sftp.trust_body"] = guidance.dialog
		catalog["terminal.trust"] = guidance.terminal
	}
}
