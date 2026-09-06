package desktop

import "github.com/bren-wp/Ghost-FTP/internal/i18n"

type credentialConsentWords struct {
	Title    string
	Question string
	Body     string
}

var localizedCredentialConsent = map[string]credentialConsentWords{
	"en": {"Ghost FTP — Privacy", "Save credentials on this computer?", "Yes = store the entered password/private-key passphrase in the protected local Ghost FTP profile store.\nNo = save the profile without stored credentials and remove any previously stored credentials for it."},
	"hr": {"Ghost FTP — Privatnost", "Spremiti vjerodajnice na ovo računalo?", "Da = spremi unesenu lozinku/zaporku privatnog ključa u zaštićenu lokalnu pohranu profila Ghost FTP-a.\nNe = spremi profil bez spremljenih vjerodajnica i ukloni prethodno spremljene vjerodajnice tog profila."},
	"de": {"Ghost FTP — Datenschutz", "Anmeldedaten auf diesem Computer speichern?", "Ja = eingegebenes Passwort bzw. die Passphrase des privaten Schlüssels im geschützten lokalen Profilspeicher von Ghost FTP speichern.\nNein = Profil ohne gespeicherte Anmeldedaten sichern und zuvor gespeicherte Anmeldedaten dieses Profils entfernen."},
	"fr": {"Ghost FTP — Confidentialité", "Enregistrer les identifiants sur cet ordinateur ?", "Oui = enregistrer le mot de passe ou la phrase secrète de la clé privée dans le stockage local protégé des profils Ghost FTP.\nNon = enregistrer le profil sans identifiants et supprimer ceux déjà enregistrés pour ce profil."},
	"es": {"Ghost FTP — Privacidad", "¿Guardar las credenciales en este equipo?", "Sí = guardar la contraseña o frase de la clave privada en el almacén local protegido de perfiles de Ghost FTP.\nNo = guardar el perfil sin credenciales y eliminar las credenciales guardadas anteriormente para ese perfil."},
	"tr": {"Ghost FTP — Gizlilik", "Kimlik bilgileri bu bilgisayara kaydedilsin mi?", "Evet = girilen parola veya özel anahtar parolasını Ghost FTP'nin korumalı yerel profil deposunda sakla.\nHayır = profili kayıtlı kimlik bilgileri olmadan kaydet ve bu profil için önceden saklanan bilgileri kaldır."},
	"el": {"Ghost FTP — Απόρρητο", "Αποθήκευση διαπιστευτηρίων σε αυτόν τον υπολογιστή;", "Ναι = αποθήκευση του κωδικού ή της φράσης πρόσβασης ιδιωτικού κλειδιού στην προστατευμένη τοπική αποθήκη προφίλ του Ghost FTP.\nΌχι = αποθήκευση του προφίλ χωρίς διαπιστευτήρια και αφαίρεση όσων είχαν αποθηκευτεί προηγουμένως."},
	"pt": {"Ghost FTP — Privacidade", "Guardar credenciais neste computador?", "Sim = guardar a palavra-passe ou frase-passe da chave privada no armazenamento local protegido de perfis do Ghost FTP.\nNão = guardar o perfil sem credenciais e remover as credenciais anteriormente guardadas para esse perfil."},
	"zh": {"Ghost FTP — 隐私", "在此计算机上保存凭据？", "是 = 将输入的密码或私钥口令保存到 Ghost FTP 受保护的本地配置文件存储中。\n否 = 保存不含凭据的配置文件，并删除此前为该配置文件保存的凭据。"},
	"ru": {"Ghost FTP — Конфиденциальность", "Сохранить учётные данные на этом компьютере?", "Да = сохранить введённый пароль или парольную фразу закрытого ключа в защищённом локальном хранилище профилей Ghost FTP.\nНет = сохранить профиль без учётных данных и удалить ранее сохранённые данные этого профиля."},
	"hi": {"Ghost FTP — गोपनीयता", "क्या क्रेडेंशियल इस कंप्यूटर पर सहेजें?", "हाँ = दर्ज पासवर्ड या निजी-कुंजी पासफ्रेज़ को Ghost FTP के सुरक्षित स्थानीय प्रोफ़ाइल स्टोर में सहेजें।\nनहीं = प्रोफ़ाइल को बिना सहेजे क्रेडेंशियल के रखें और पहले सहेजे गए क्रेडेंशियल हटा दें।"},
	"ja": {"Ghost FTP — プライバシー", "このコンピューターに認証情報を保存しますか？", "はい = 入力したパスワードまたは秘密鍵パスフレーズを Ghost FTP の保護されたローカルプロファイルストアに保存します。\nいいえ = 認証情報なしでプロファイルを保存し、以前保存した認証情報を削除します。"},
	"it": {"Ghost FTP — Privacy", "Salvare le credenziali su questo computer?", "Sì = salva la password o la passphrase della chiave privata nell'archivio locale protetto dei profili Ghost FTP.\nNo = salva il profilo senza credenziali e rimuovi quelle precedentemente memorizzate per il profilo."},
	"pl": {"Ghost FTP — Prywatność", "Zapisać dane logowania na tym komputerze?", "Tak = zapisz hasło lub frazę klucza prywatnego w chronionym lokalnym magazynie profili Ghost FTP.\nNie = zapisz profil bez danych logowania i usuń wcześniej zapisane dane tego profilu."},
	"nl": {"Ghost FTP — Privacy", "Aanmeldgegevens op deze computer opslaan?", "Ja = sla het wachtwoord of de wachtzin van de privésleutel op in de beveiligde lokale profielopslag van Ghost FTP.\nNee = sla het profiel zonder aanmeldgegevens op en verwijder eerder opgeslagen gegevens voor dit profiel."},
	"cs": {"Ghost FTP — Soukromí", "Uložit přihlašovací údaje do tohoto počítače?", "Ano = uložit heslo nebo heslovou frázi soukromého klíče do chráněného místního úložiště profilů Ghost FTP.\nNe = uložit profil bez přihlašovacích údajů a odstranit dříve uložené údaje tohoto profilu."},
	"uk": {"Ghost FTP — Конфіденційність", "Зберегти облікові дані на цьому комп’ютері?", "Так = зберегти пароль або парольну фразу приватного ключа в захищеному локальному сховищі профілів Ghost FTP.\nНі = зберегти профіль без облікових даних і видалити раніше збережені дані цього профілю."},
	"sv": {"Ghost FTP — Sekretess", "Spara inloggningsuppgifter på den här datorn?", "Ja = spara lösenordet eller den privata nyckelns lösenfras i Ghost FTP:s skyddade lokala profillagring.\nNej = spara profilen utan inloggningsuppgifter och ta bort tidigare sparade uppgifter för profilen."},
	"ro": {"Ghost FTP — Confidențialitate", "Salvați acreditările pe acest computer?", "Da = salvați parola sau fraza cheii private în spațiul local protejat pentru profiluri Ghost FTP.\nNu = salvați profilul fără acreditări și eliminați acreditările salvate anterior pentru acest profil."},
	"hu": {"Ghost FTP — Adatvédelem", "Mentse a hitelesítő adatokat erre a számítógépre?", "Igen = a megadott jelszó vagy privátkulcs-jelmondat mentése a Ghost FTP védett helyi profiltárolójába.\nNem = a profil mentése hitelesítő adatok nélkül és a korábban mentett adatok eltávolítása."},
	"da": {"Ghost FTP — Privatliv", "Gem legitimationsoplysninger på denne computer?", "Ja = gem adgangskoden eller den private nøgles adgangsfrase i Ghost FTP's beskyttede lokale profillager.\nNej = gem profilen uden legitimationsoplysninger og fjern tidligere gemte oplysninger for profilen."},
	"fi": {"Ghost FTP — Tietosuoja", "Tallennetaanko tunnistetiedot tälle tietokoneelle?", "Kyllä = tallenna salasana tai yksityisen avaimen tunnuslause Ghost FTP:n suojattuun paikalliseen profiilitallennukseen.\nEi = tallenna profiili ilman tunnistetietoja ja poista profiilin aiemmin tallennetut tunnistetiedot."},
	"no": {"Ghost FTP — Personvern", "Lagre påloggingsinformasjon på denne datamaskinen?", "Ja = lagre passordet eller passfrasen for privatnøkkelen i Ghost FTPs beskyttede lokale profillager.\nNei = lagre profilen uten påloggingsinformasjon og fjern tidligere lagrede opplysninger for profilen."},
	"ko": {"Ghost FTP — 개인정보", "이 컴퓨터에 자격 증명을 저장하시겠습니까?", "예 = 입력한 암호 또는 개인 키 암호문을 Ghost FTP의 보호된 로컬 프로필 저장소에 저장합니다.\n아니요 = 자격 증명 없이 프로필을 저장하고 이 프로필에 이전에 저장된 자격 증명을 제거합니다."},
}

func credentialConsentText(language string) credentialConsentWords {
	if words, ok := localizedCredentialConsent[i18n.Normalize(language)]; ok {
		return words
	}
	return localizedCredentialConsent[i18n.DefaultLanguage]
}
