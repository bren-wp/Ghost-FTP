//go:build windows

package desktop

import "fmt"

func profileBusyHeading(language string) string {
	return localizedPrompt(language, [24]string{
		"Profile changes are unavailable during a connection",
		"Promjene profila nisu dostupne tijekom veze",
		"Profiländerungen sind während einer Verbindung nicht verfügbar",
		"Les modifications de profil sont indisponibles pendant une connexion",
		"Los cambios de perfil no están disponibles durante una conexión",
		"Bağlantı sırasında profil değişiklikleri kullanılamaz",
		"Οι αλλαγές προφίλ δεν είναι διαθέσιμες κατά τη σύνδεση",
		"As alterações de perfil não estão disponíveis durante uma ligação",
		"连接期间无法更改配置文件",
		"Изменение профиля недоступно во время подключения",
		"कनेक्शन के दौरान प्रोफ़ाइल बदलाव उपलब्ध नहीं हैं",
		"接続中はプロファイルを変更できません",
		"Le modifiche al profilo non sono disponibili durante una connessione",
		"Zmiany profilu są niedostępne podczas połączenia",
		"Profielwijzigingen zijn niet beschikbaar tijdens een verbinding",
		"Změny profilu nejsou během připojení dostupné",
		"Зміни профілю недоступні під час з’єднання",
		"Profiländringar är inte tillgängliga under en anslutning",
		"Modificările profilului nu sunt disponibile în timpul unei conexiuni",
		"A profil módosítása kapcsolat közben nem érhető el",
		"Profilændringer er ikke tilgængelige under en forbindelse",
		"Profiilin muutokset eivät ole käytettävissä yhteyden aikana",
		"Profilendringer er ikke tilgjengelige under en tilkobling",
		"연결 중에는 프로필을 변경할 수 없습니다",
	})
}

func profileBusyBody(language string) string {
	return localizedPrompt(language, [24]string{
		"Wait for the connection attempt to finish or disconnect before saving profile changes.",
		"Pričekajte završetak pokušaja povezivanja ili prekinite vezu prije spremanja promjena profila.",
		"Warten Sie, bis der Verbindungsversuch beendet ist, oder trennen Sie die Verbindung, bevor Sie Profiländerungen speichern.",
		"Attendez la fin de la tentative de connexion ou déconnectez-vous avant d’enregistrer les modifications du profil.",
		"Espere a que finalice el intento de conexión o desconéctese antes de guardar los cambios del perfil.",
		"Profil değişikliklerini kaydetmeden önce bağlantı denemesinin tamamlanmasını bekleyin veya bağlantıyı kesin.",
		"Περιμένετε να ολοκληρωθεί η προσπάθεια σύνδεσης ή αποσυνδεθείτε πριν αποθηκεύσετε αλλαγές στο προφίλ.",
		"Aguarde a conclusão da tentativa de ligação ou desligue antes de guardar alterações ao perfil.",
		"请等待连接尝试完成，或先断开连接，再保存配置文件更改。",
		"Дождитесь завершения попытки подключения или отключитесь перед сохранением изменений профиля.",
		"प्रोफ़ाइल बदलाव सहेजने से पहले कनेक्शन प्रयास पूरा होने दें या डिस्कनेक्ट करें।",
		"接続試行の完了を待つか切断してから、プロファイルの変更を保存してください。",
		"Attendi il termine del tentativo di connessione oppure disconnettiti prima di salvare le modifiche al profilo.",
		"Poczekaj na zakończenie próby połączenia lub rozłącz się przed zapisaniem zmian profilu.",
		"Wacht tot de verbindingspoging is voltooid of verbreek de verbinding voordat u profielwijzigingen opslaat.",
		"Před uložením změn profilu počkejte na dokončení pokusu o připojení nebo se odpojte.",
		"Дочекайтеся завершення спроби з’єднання або від’єднайтеся перед збереженням змін профілю.",
		"Vänta tills anslutningsförsöket är klart eller koppla från innan profiländringar sparas.",
		"Așteptați finalizarea încercării de conectare sau deconectați-vă înainte de a salva modificările profilului.",
		"A profilmódosítások mentése előtt várja meg a kapcsolódási kísérlet végét, vagy bontsa a kapcsolatot.",
		"Vent til forbindelsesforsøget er afsluttet, eller afbryd forbindelsen, før profilændringer gemmes.",
		"Odota yhteysyrityksen valmistumista tai katkaise yhteys ennen profiilimuutosten tallentamista.",
		"Vent til tilkoblingsforsøket er ferdig, eller koble fra før du lagrer profilendringer.",
		"프로필 변경 사항을 저장하기 전에 연결 시도가 끝날 때까지 기다리거나 연결을 끊으세요.",
	})
}

func profileNameLabel(language string) string {
	return localizedPrompt(language, [24]string{
		"Profile name:", "Naziv profila:", "Profilname:", "Nom du profil :", "Nombre del perfil:", "Profil adı:",
		"Όνομα προφίλ:", "Nome do perfil:", "配置文件名称：", "Имя профиля:", "प्रोफ़ाइल नाम:", "プロファイル名:",
		"Nome profilo:", "Nazwa profilu:", "Profielnaam:", "Název profilu:", "Назва профілю:", "Profilnamn:",
		"Numele profilului:", "Profil neve:", "Profilnavn:", "Profiilin nimi:", "Profilnavn:", "프로필 이름:",
	})
}

func profileNameRequiredHeading(language string) string {
	return localizedPrompt(language, [24]string{
		"Profile name is required", "Naziv profila je obavezan", "Ein Profilname ist erforderlich", "Le nom du profil est obligatoire",
		"El nombre del perfil es obligatorio", "Profil adı gereklidir", "Απαιτείται όνομα προφίλ", "O nome do perfil é obrigatório",
		"必须填写配置文件名称", "Необходимо указать имя профиля", "प्रोफ़ाइल नाम आवश्यक है", "プロファイル名は必須です",
		"Il nome del profilo è obbligatorio", "Nazwa profilu jest wymagana", "Een profielnaam is vereist", "Název profilu je povinný",
		"Потрібно вказати назву профілю", "Profilnamn krävs", "Numele profilului este obligatoriu", "A profil neve kötelező",
		"Profilnavn er påkrævet", "Profiilin nimi vaaditaan", "Profilnavn er påkrevd", "프로필 이름이 필요합니다",
	})
}

func profileNameRequiredBody(language string) string {
	return localizedPrompt(language, [24]string{
		"Enter a name that identifies this connection.", "Unesite naziv po kojem ćete prepoznati ovu vezu.", "Geben Sie einen Namen ein, der diese Verbindung identifiziert.",
		"Saisissez un nom permettant d’identifier cette connexion.", "Introduzca un nombre que identifique esta conexión.", "Bu bağlantıyı tanımlayan bir ad girin.",
		"Εισαγάγετε ένα όνομα που προσδιορίζει αυτή τη σύνδεση.", "Introduza um nome que identifique esta ligação.", "请输入用于识别此连接的名称。",
		"Введите имя, по которому можно определить это подключение.", "इस कनेक्शन की पहचान करने वाला नाम दर्ज करें।", "この接続を識別できる名前を入力してください。",
		"Inserisci un nome che identifichi questa connessione.", "Wprowadź nazwę identyfikującą to połączenie.", "Voer een naam in waarmee deze verbinding kan worden herkend.",
		"Zadejte název, podle kterého toto připojení poznáte.", "Введіть назву, за якою можна розпізнати це з’єднання.", "Ange ett namn som identifierar den här anslutningen.",
		"Introduceți un nume care identifică această conexiune.", "Adjon meg egy nevet, amely azonosítja ezt a kapcsolatot.", "Indtast et navn, der identificerer denne forbindelse.",
		"Anna nimi, josta tämä yhteys tunnistetaan.", "Skriv inn et navn som identifiserer denne tilkoblingen.", "이 연결을 식별할 이름을 입력하세요.",
	})
}

func profilePrivacyTitle(language string) string {
	return localizedPrompt(language, [24]string{
		"privacy", "privatnost", "Datenschutz", "confidentialité", "privacidad", "gizlilik", "απόρρητο", "privacidade", "隐私", "конфиденциальность", "गोपनीयता", "プライバシー",
		"privacy", "prywatność", "privacy", "soukromí", "конфіденційність", "integritet", "confidențialitate", "adatvédelem", "privatliv", "tietosuoja", "personvern", "개인정보 보호",
	})
}

func profileRetainHeading(language string) string {
	return localizedPrompt(language, [24]string{
		"Retain stored credentials?", "Zadržati spremljene vjerodajnice?", "Gespeicherte Anmeldedaten beibehalten?", "Conserver les identifiants enregistrés ?",
		"¿Conservar las credenciales guardadas?", "Kayıtlı kimlik bilgileri korunsun mu?", "Διατήρηση αποθηκευμένων διαπιστευτηρίων;", "Manter as credenciais guardadas?",
		"保留已保存的凭据？", "Сохранить сохранённые учётные данные?", "सहेजे गए क्रेडेंशियल रखें?", "保存済みの認証情報を保持しますか？",
		"Mantenere le credenziali salvate?", "Zachować zapisane dane logowania?", "Opgeslagen aanmeldgegevens behouden?", "Ponechat uložené přihlašovací údaje?",
		"Зберегти збережені облікові дані?", "Behålla sparade autentiseringsuppgifter?", "Păstrați datele de autentificare salvate?", "Megtartja a mentett hitelesítő adatokat?",
		"Behold gemte legitimationsoplysninger?", "Säilytetäänkö tallennetut tunnistetiedot?", "Beholde lagret påloggingsinformasjon?", "저장된 자격 증명을 유지할까요?",
	})
}

func profileRetainBody(language string) string {
	template := localizedPrompt(language, [24]string{
		"This profile already contains stored credentials that still belong to the same identity.\n\n%s = retain them.\n%s = remove them from the Ghost FTP profile store.",
		"Ovaj profil već sadrži spremljene vjerodajnice koje i dalje pripadaju istom identitetu.\n\n%s = zadrži ih.\n%s = ukloni ih iz spremišta profila Ghost FTP-a.",
		"Dieses Profil enthält bereits gespeicherte Anmeldedaten, die weiterhin zur selben Identität gehören.\n\n%s = beibehalten.\n%s = aus dem Ghost-FTP-Profilspeicher entfernen.",
		"Ce profil contient déjà des identifiants enregistrés qui appartiennent toujours à la même identité.\n\n%s = les conserver.\n%s = les supprimer du stockage de profils Ghost FTP.",
		"Este perfil ya contiene credenciales guardadas que siguen perteneciendo a la misma identidad.\n\n%s = conservarlas.\n%s = eliminarlas del almacén de perfiles de Ghost FTP.",
		"Bu profil, hâlâ aynı kimliğe ait kayıtlı kimlik bilgileri içeriyor.\n\n%s = koru.\n%s = Ghost FTP profil deposundan kaldır.",
		"Αυτό το προφίλ περιέχει ήδη αποθηκευμένα διαπιστευτήρια που εξακολουθούν να ανήκουν στην ίδια ταυτότητα.\n\n%s = διατήρηση.\n%s = αφαίρεση από το χώρο αποθήκευσης προφίλ Ghost FTP.",
		"Este perfil já contém credenciais guardadas que continuam a pertencer à mesma identidade.\n\n%s = manter.\n%s = remover do armazenamento de perfis do Ghost FTP.",
		"此配置文件已包含仍属于同一身份的已保存凭据。\n\n%s = 保留。\n%s = 从 Ghost FTP 配置文件存储中删除。",
		"В этом профиле уже есть сохранённые учётные данные, которые по-прежнему относятся к той же учётной записи.\n\n%s = сохранить.\n%s = удалить из хранилища профиля Ghost FTP.",
		"इस प्रोफ़ाइल में पहले से सहेजे गए क्रेडेंशियल हैं जो अभी भी उसी पहचान से संबंधित हैं।\n\n%s = उन्हें रखें।\n%s = उन्हें Ghost FTP प्रोफ़ाइल स्टोर से हटाएँ।",
		"このプロファイルには、同じ認証対象に属する保存済みの認証情報があります。\n\n%s = 保持。\n%s = Ghost FTP のプロファイルストアから削除。",
		"Questo profilo contiene già credenziali salvate che appartengono ancora alla stessa identità.\n\n%s = mantenerle.\n%s = rimuoverle dall'archivio profili di Ghost FTP.",
		"Ten profil zawiera już zapisane dane logowania, które nadal należą do tej samej tożsamości.\n\n%s = zachowaj je.\n%s = usuń je z magazynu profilu Ghost FTP.",
		"Dit profiel bevat al opgeslagen aanmeldgegevens die nog bij dezelfde identiteit horen.\n\n%s = behouden.\n%s = verwijderen uit de profielopslag van Ghost FTP.",
		"Tento profil již obsahuje uložené přihlašovací údaje, které stále patří ke stejné identitě.\n\n%s = ponechat.\n%s = odstranit z úložiště profilu Ghost FTP.",
		"Цей профіль уже містить збережені облікові дані, які й надалі належать до тієї самої ідентичності.\n\n%s = зберегти їх.\n%s = видалити зі сховища профілю Ghost FTP.",
		"Den här profilen innehåller redan sparade autentiseringsuppgifter som fortfarande tillhör samma identitet.\n\n%s = behåll dem.\n%s = ta bort dem från Ghost FTP:s profillagring.",
		"Acest profil conține deja date de autentificare salvate care aparțin în continuare aceleiași identități.\n\n%s = păstrați-le.\n%s = eliminați-le din stocarea profilului Ghost FTP.",
		"Ez a profil már tartalmaz olyan mentett hitelesítő adatokat, amelyek továbbra is ugyanahhoz az identitáshoz tartoznak.\n\n%s = megtartás.\n%s = eltávolítás a Ghost FTP profiltárolójából.",
		"Denne profil indeholder allerede gemte legitimationsoplysninger, som stadig tilhører den samme identitet.\n\n%s = behold dem.\n%s = fjern dem fra Ghost FTP-profildataene.",
		"Tässä profiilissa on jo tallennettuja tunnistetietoja, jotka kuuluvat edelleen samaan identiteettiin.\n\n%s = säilytä ne.\n%s = poista ne Ghost FTP:n profiilivarastosta.",
		"Denne profilen inneholder allerede lagret påloggingsinformasjon som fortsatt tilhører samme identitet.\n\n%s = behold den.\n%s = fjern den fra Ghost FTP-profillageret.",
		"이 프로필에는 동일한 자격에 계속 속하는 저장된 자격 증명이 이미 있습니다.\n\n%s = 유지.\n%s = Ghost FTP 프로필 저장소에서 제거.",
	})
	return fmt.Sprintf(template, yesLabel(language), noLabel(language))
}

func profileAutoRemovedPrefix(language string) string {
	return localizedPrompt(language, [24]string{
		"Credentials that no longer belong to the changed server, account or private key will be removed automatically.",
		"Vjerodajnice koje više ne pripadaju promijenjenom poslužitelju, računu ili privatnom ključu bit će automatski uklonjene.",
		"Anmeldedaten, die nicht mehr zum geänderten Server, Konto oder privaten Schlüssel gehören, werden automatisch entfernt.",
		"Les identifiants qui ne correspondent plus au serveur, au compte ou à la clé privée modifiés seront supprimés automatiquement.",
		"Las credenciales que ya no correspondan al servidor, la cuenta o la clave privada modificados se eliminarán automáticamente.",
		"Değiştirilen sunucuya, hesaba veya özel anahtara artık ait olmayan kimlik bilgileri otomatik olarak kaldırılacaktır.",
		"Τα διαπιστευτήρια που δεν ανήκουν πλέον στον αλλαγμένο διακομιστή, λογαριασμό ή ιδιωτικό κλειδί θα αφαιρεθούν αυτόματα.",
		"As credenciais que já não pertencem ao servidor, conta ou chave privada alterados serão removidas automaticamente.",
		"不再属于已更改服务器、帐户或私钥的凭据将自动删除。",
		"Учётные данные, которые больше не относятся к изменённому серверу, учётной записи или закрытому ключу, будут удалены автоматически.",
		"बदले गए सर्वर, खाते या निजी कुंजी से अब संबंधित न रहने वाले क्रेडेंशियल अपने आप हटा दिए जाएंगे।",
		"変更後のサーバー、アカウント、または秘密鍵に属さない認証情報は自動的に削除されます。",
		"Le credenziali che non appartengono più al server, all'account o alla chiave privata modificati verranno rimosse automaticamente.",
		"Dane logowania, które nie należą już do zmienionego serwera, konta lub klucza prywatnego, zostaną usunięte automatycznie.",
		"Aanmeldgegevens die niet meer bij de gewijzigde server, account of privésleutel horen, worden automatisch verwijderd.",
		"Přihlašovací údaje, které již nepatří ke změněnému serveru, účtu nebo soukromému klíči, budou automaticky odstraněny.",
		"Облікові дані, які більше не належать зміненому серверу, обліковому запису або приватному ключу, буде видалено автоматично.",
		"Autentiseringsuppgifter som inte längre hör till den ändrade servern, kontot eller privata nyckeln tas bort automatiskt.",
		"Datele de autentificare care nu mai aparțin serverului, contului sau cheii private modificate vor fi eliminate automat.",
		"A módosított szerverhez, fiókhoz vagy privát kulcshoz már nem tartozó hitelesítő adatok automatikusan törlődnek.",
		"Legitimationsoplysninger, der ikke længere tilhører den ændrede server, konto eller private nøgle, fjernes automatisk.",
		"Muuttuneeseen palvelimeen, tiliin tai yksityiseen avaimeen enää kuulumattomat tunnistetiedot poistetaan automaattisesti.",
		"Påloggingsinformasjon som ikke lenger tilhører den endrede serveren, kontoen eller private nøkkelen, fjernes automatisk.",
		"변경된 서버, 계정 또는 개인 키에 더 이상 속하지 않는 자격 증명은 자동으로 제거됩니다.",
	})
}

func profileOldCredentialsHeading(language string) string {
	return localizedPrompt(language, [24]string{
		"Old credentials will not be transferred", "Stare vjerodajnice neće se prenijeti", "Alte Anmeldedaten werden nicht übernommen", "Les anciens identifiants ne seront pas transférés",
		"Las credenciales antiguas no se transferirán", "Eski kimlik bilgileri aktarılmayacak", "Τα παλιά διαπιστευτήρια δεν θα μεταφερθούν", "As credenciais antigas não serão transferidas",
		"旧凭据不会转移", "Старые учётные данные не будут перенесены", "पुराने क्रेडेंशियल स्थानांतरित नहीं किए जाएंगे", "以前の認証情報は引き継がれません",
		"Le vecchie credenziali non verranno trasferite", "Stare dane logowania nie zostaną przeniesione", "Oude aanmeldgegevens worden niet overgenomen", "Staré přihlašovací údaje nebudou přeneseny",
		"Старі облікові дані не буде перенесено", "Gamla autentiseringsuppgifter överförs inte", "Datele de autentificare vechi nu vor fi transferate", "A régi hitelesítő adatok nem kerülnek át",
		"Gamle legitimationsoplysninger overføres ikke", "Vanhoja tunnistetietoja ei siirretä", "Gammel påloggingsinformasjon overføres ikke", "이전 자격 증명은 이전되지 않습니다",
	})
}

func profileOldCredentialsBody(language string) string {
	return localizedPrompt(language, [24]string{
		"The server, port, username or private key changed. For safety, old stored credentials are removed from this profile. Enter them again if you want to store credentials for the new identity.",
		"Promijenjen je poslužitelj, port, korisničko ime ili privatni ključ. Radi sigurnosti stare spremljene vjerodajnice uklanjaju se iz ovog profila. Unesite ih ponovno ako želite spremiti vjerodajnice za novi identitet.",
		"Server, Port, Benutzername oder privater Schlüssel wurden geändert. Aus Sicherheitsgründen werden alte gespeicherte Anmeldedaten aus diesem Profil entfernt. Geben Sie sie erneut ein, wenn Sie Anmeldedaten für die neue Identität speichern möchten.",
		"Le serveur, le port, le nom d’utilisateur ou la clé privée a changé. Par sécurité, les anciens identifiants enregistrés sont supprimés de ce profil. Saisissez-les à nouveau si vous souhaitez enregistrer des identifiants pour la nouvelle identité.",
		"El servidor, el puerto, el nombre de usuario o la clave privada han cambiado. Por seguridad, las credenciales guardadas anteriormente se eliminan de este perfil. Vuelva a introducirlas si desea guardar credenciales para la nueva identidad.",
		"Sunucu, bağlantı noktası, kullanıcı adı veya özel anahtar değişti. Güvenlik için eski kayıtlı kimlik bilgileri bu profilden kaldırılır. Yeni kimlik için kimlik bilgilerini kaydetmek istiyorsanız yeniden girin.",
		"Ο διακομιστής, η θύρα, το όνομα χρήστη ή το ιδιωτικό κλειδί άλλαξε. Για λόγους ασφαλείας, τα παλιά αποθηκευμένα διαπιστευτήρια αφαιρούνται από αυτό το προφίλ. Εισαγάγετέ τα ξανά αν θέλετε να αποθηκεύσετε διαπιστευτήρια για τη νέα ταυτότητα.",
		"O servidor, a porta, o nome de utilizador ou a chave privada mudou. Por segurança, as credenciais anteriormente guardadas são removidas deste perfil. Introduza-as novamente se quiser guardar credenciais para a nova identidade.",
		"服务器、端口、用户名或私钥已更改。为安全起见，旧的已保存凭据将从此配置文件中删除。如果要为新身份保存凭据，请重新输入。",
		"Сервер, порт, имя пользователя или закрытый ключ изменились. В целях безопасности старые сохранённые учётные данные удаляются из этого профиля. Введите их снова, если хотите сохранить данные для новой учётной записи.",
		"सर्वर, पोर्ट, उपयोगकर्ता नाम या निजी कुंजी बदल गई है। सुरक्षा के लिए पुराने सहेजे गए क्रेडेंशियल इस प्रोफ़ाइल से हटा दिए जाते हैं। नई पहचान के लिए क्रेडेंशियल सहेजना हो तो उन्हें फिर से दर्ज करें।",
		"サーバー、ポート、ユーザー名、または秘密鍵が変更されました。安全のため、以前に保存された認証情報はこのプロファイルから削除されます。新しい認証対象の情報を保存する場合は、もう一度入力してください。",
		"Il server, la porta, il nome utente o la chiave privata sono cambiati. Per sicurezza, le vecchie credenziali salvate vengono rimosse da questo profilo. Inseriscile di nuovo se vuoi salvare credenziali per la nuova identità.",
		"Serwer, port, nazwa użytkownika lub klucz prywatny uległy zmianie. Ze względów bezpieczeństwa stare zapisane dane logowania są usuwane z tego profilu. Wprowadź je ponownie, jeśli chcesz zapisać dane dla nowej tożsamości.",
		"De server, poort, gebruikersnaam of privésleutel is gewijzigd. Voor de veiligheid worden oude opgeslagen aanmeldgegevens uit dit profiel verwijderd. Voer ze opnieuw in als u aanmeldgegevens voor de nieuwe identiteit wilt opslaan.",
		"Server, port, uživatelské jméno nebo soukromý klíč se změnily. Z bezpečnostních důvodů budou staré uložené přihlašovací údaje z tohoto profilu odstraněny. Pokud chcete uložit údaje pro novou identitu, zadejte je znovu.",
		"Сервер, порт, ім’я користувача або приватний ключ змінилися. З міркувань безпеки старі збережені облікові дані буде видалено з цього профілю. Введіть їх знову, якщо хочете зберегти дані для нової ідентичності.",
		"Servern, porten, användarnamnet eller den privata nyckeln har ändrats. Av säkerhetsskäl tas gamla sparade autentiseringsuppgifter bort från profilen. Ange dem igen om du vill spara uppgifter för den nya identiteten.",
		"Serverul, portul, numele de utilizator sau cheia privată s-a schimbat. Pentru siguranță, datele de autentificare vechi salvate sunt eliminate din acest profil. Introduceți-le din nou dacă doriți să salvați date pentru noua identitate.",
		"A szerver, a port, a felhasználónév vagy a privát kulcs megváltozott. Biztonsági okból a régi mentett hitelesítő adatok törlődnek ebből a profilból. Adja meg őket újra, ha az új identitáshoz szeretne adatokat menteni.",
		"Serveren, porten, brugernavnet eller den private nøgle er ændret. Af sikkerhedshensyn fjernes gamle gemte legitimationsoplysninger fra denne profil. Indtast dem igen, hvis du vil gemme oplysninger for den nye identitet.",
		"Palvelin, portti, käyttäjänimi tai yksityinen avain muuttui. Turvallisuuden vuoksi vanhat tallennetut tunnistetiedot poistetaan tästä profiilista. Syötä ne uudelleen, jos haluat tallentaa tunnistetiedot uudelle identiteetille.",
		"Serveren, porten, brukernavnet eller den private nøkkelen er endret. Av sikkerhetsgrunner fjernes gammel lagret påloggingsinformasjon fra denne profilen. Skriv den inn på nytt hvis du vil lagre informasjon for den nye identiteten.",
		"서버, 포트, 사용자 이름 또는 개인 키가 변경되었습니다. 보안을 위해 이전에 저장된 자격 증명은 이 프로필에서 제거됩니다. 새 자격에 대한 정보를 저장하려면 다시 입력하세요.",
	})
}
