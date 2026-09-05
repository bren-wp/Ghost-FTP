package main

import (
	"fmt"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
)

type installerCopy struct {
	ConfirmTitle    string
	ConfirmBody     string
	CompletedTitle  string
	ReadyBody       string
	LaunchQuestion  string
	InstalledTitle  string
	LaunchFailed    string
	LanguageWarning string
	ShortcutWarning string
}

var installerCopies = map[string]installerCopy{
	"en": {
		ConfirmTitle: "Install Ghost FTP %s?", ConfirmBody: "Ghost FTP will be installed for your Windows user account and will be available from the Start menu.",
		CompletedTitle: "Setup completed successfully", ReadyBody: "Ghost FTP is ready to use.", LaunchQuestion: "Launch Ghost FTP now?", InstalledTitle: "Ghost FTP is installed",
		LaunchFailed: "Ghost FTP could not be launched automatically. Start it from the Windows Start menu.", LanguageWarning: "The selected language could not be saved. Ghost FTP will start in English; you can change the language in Settings.",
		ShortcutWarning: "A shortcut could not be created. You can start Ghost FTP from its installation folder.",
	},
	"hr": {
		ConfirmTitle: "Instalirati Ghost FTP %s?", ConfirmBody: "Ghost FTP instalirat će se za vaš Windows korisnički račun i bit će dostupan iz izbornika Start.",
		CompletedTitle: "Instalacija je uspješno dovršena", ReadyBody: "Ghost FTP je spreman za korištenje.", LaunchQuestion: "Pokrenuti Ghost FTP sada?", InstalledTitle: "Ghost FTP je instaliran",
		LaunchFailed: "Ghost FTP nije moguće automatski pokrenuti. Pokrenite ga iz Windows izbornika Start.", LanguageWarning: "Odabrani jezik nije spremljen. Ghost FTP će se pokrenuti na engleskom; jezik možete promijeniti u Postavkama.",
		ShortcutWarning: "Prečac nije moguće izraditi. Ghost FTP možete pokrenuti iz instalacijske mape.",
	},
	"de": {
		ConfirmTitle: "Ghost FTP %s installieren?", ConfirmBody: "Ghost FTP wird für Ihr Windows-Benutzerkonto installiert und ist über das Startmenü verfügbar.",
		CompletedTitle: "Setup erfolgreich abgeschlossen", ReadyBody: "Ghost FTP ist einsatzbereit.", LaunchQuestion: "Ghost FTP jetzt starten?", InstalledTitle: "Ghost FTP ist installiert",
		LaunchFailed: "Ghost FTP konnte nicht automatisch gestartet werden. Starten Sie es über das Windows-Startmenü.", LanguageWarning: "Die ausgewählte Sprache konnte nicht gespeichert werden. Ghost FTP startet auf Englisch; Sie können die Sprache in den Einstellungen ändern.",
		ShortcutWarning: "Eine Verknüpfung konnte nicht erstellt werden. Sie können Ghost FTP aus dem Installationsordner starten.",
	},
	"fr": {
		ConfirmTitle: "Installer Ghost FTP %s ?", ConfirmBody: "Ghost FTP sera installé pour votre compte Windows et sera disponible depuis le menu Démarrer.",
		CompletedTitle: "Installation terminée avec succès", ReadyBody: "Ghost FTP est prêt à être utilisé.", LaunchQuestion: "Lancer Ghost FTP maintenant ?", InstalledTitle: "Ghost FTP est installé",
		LaunchFailed: "Ghost FTP n'a pas pu être lancé automatiquement. Démarrez-le depuis le menu Démarrer de Windows.", LanguageWarning: "La langue sélectionnée n'a pas pu être enregistrée. Ghost FTP démarrera en anglais ; vous pourrez modifier la langue dans les paramètres.",
		ShortcutWarning: "Un raccourci n'a pas pu être créé. Vous pouvez démarrer Ghost FTP depuis son dossier d'installation.",
	},
	"es": {
		ConfirmTitle: "¿Instalar Ghost FTP %s?", ConfirmBody: "Ghost FTP se instalará para su cuenta de Windows y estará disponible desde el menú Inicio.",
		CompletedTitle: "Instalación completada correctamente", ReadyBody: "Ghost FTP está listo para usarse.", LaunchQuestion: "¿Iniciar Ghost FTP ahora?", InstalledTitle: "Ghost FTP está instalado",
		LaunchFailed: "Ghost FTP no pudo iniciarse automáticamente. Inícielo desde el menú Inicio de Windows.", LanguageWarning: "No se pudo guardar el idioma seleccionado. Ghost FTP se iniciará en inglés; puede cambiar el idioma en Configuración.",
		ShortcutWarning: "No se pudo crear un acceso directo. Puede iniciar Ghost FTP desde su carpeta de instalación.",
	},
	"tr": {
		ConfirmTitle: "Ghost FTP %s yüklensin mi?", ConfirmBody: "Ghost FTP Windows kullanıcı hesabınız için yüklenecek ve Başlat menüsünden erişilebilir olacaktır.",
		CompletedTitle: "Kurulum başarıyla tamamlandı", ReadyBody: "Ghost FTP kullanıma hazır.", LaunchQuestion: "Ghost FTP şimdi başlatılsın mı?", InstalledTitle: "Ghost FTP yüklendi",
		LaunchFailed: "Ghost FTP otomatik olarak başlatılamadı. Windows Başlat menüsünden başlatın.", LanguageWarning: "Seçilen dil kaydedilemedi. Ghost FTP İngilizce başlayacak; dili Ayarlar'dan değiştirebilirsiniz.",
		ShortcutWarning: "Kısayol oluşturulamadı. Ghost FTP'yi kurulum klasöründen başlatabilirsiniz.",
	},
	"el": {
		ConfirmTitle: "Εγκατάσταση Ghost FTP %s;", ConfirmBody: "Το Ghost FTP θα εγκατασταθεί για τον λογαριασμό Windows και θα είναι διαθέσιμο από το μενού Έναρξη.",
		CompletedTitle: "Η εγκατάσταση ολοκληρώθηκε επιτυχώς", ReadyBody: "Το Ghost FTP είναι έτοιμο για χρήση.", LaunchQuestion: "Εκκίνηση του Ghost FTP τώρα;", InstalledTitle: "Το Ghost FTP εγκαταστάθηκε",
		LaunchFailed: "Δεν ήταν δυνατή η αυτόματη εκκίνηση του Ghost FTP. Εκκινήστε το από το μενού Έναρξη των Windows.", LanguageWarning: "Η επιλεγμένη γλώσσα δεν αποθηκεύτηκε. Το Ghost FTP θα ξεκινήσει στα Αγγλικά· μπορείτε να αλλάξετε γλώσσα στις Ρυθμίσεις.",
		ShortcutWarning: "Δεν ήταν δυνατή η δημιουργία συντόμευσης. Μπορείτε να ξεκινήσετε το Ghost FTP από τον φάκελο εγκατάστασης.",
	},
	"pt": {
		ConfirmTitle: "Instalar Ghost FTP %s?", ConfirmBody: "O Ghost FTP será instalado para a sua conta do Windows e ficará disponível no menu Iniciar.",
		CompletedTitle: "Instalação concluída com sucesso", ReadyBody: "O Ghost FTP está pronto para uso.", LaunchQuestion: "Iniciar o Ghost FTP agora?", InstalledTitle: "O Ghost FTP está instalado",
		LaunchFailed: "O Ghost FTP não pôde ser iniciado automaticamente. Inicie-o pelo menu Iniciar do Windows.", LanguageWarning: "O idioma selecionado não pôde ser salvo. O Ghost FTP iniciará em inglês; você pode alterar o idioma nas Configurações.",
		ShortcutWarning: "Não foi possível criar um atalho. Você pode iniciar o Ghost FTP pela pasta de instalação.",
	},
	"zh": {
		ConfirmTitle: "安装 Ghost FTP %s？", ConfirmBody: "Ghost FTP 将为当前 Windows 用户账户安装，并可从“开始”菜单启动。",
		CompletedTitle: "安装成功完成", ReadyBody: "Ghost FTP 已可使用。", LaunchQuestion: "现在启动 Ghost FTP？", InstalledTitle: "Ghost FTP 已安装",
		LaunchFailed: "无法自动启动 Ghost FTP。请从 Windows“开始”菜单启动。", LanguageWarning: "无法保存所选语言。Ghost FTP 将以英语启动；您可以在“设置”中更改语言。",
		ShortcutWarning: "无法创建快捷方式。您可以从安装文件夹启动 Ghost FTP。",
	},
	"ru": {
		ConfirmTitle: "Установить Ghost FTP %s?", ConfirmBody: "Ghost FTP будет установлен для вашей учетной записи Windows и будет доступен из меню «Пуск».",
		CompletedTitle: "Установка успешно завершена", ReadyBody: "Ghost FTP готов к работе.", LaunchQuestion: "Запустить Ghost FTP сейчас?", InstalledTitle: "Ghost FTP установлен",
		LaunchFailed: "Не удалось автоматически запустить Ghost FTP. Запустите его из меню «Пуск» Windows.", LanguageWarning: "Не удалось сохранить выбранный язык. Ghost FTP запустится на английском; язык можно изменить в настройках.",
		ShortcutWarning: "Не удалось создать ярлык. Ghost FTP можно запустить из папки установки.",
	},
	"hi": {
		ConfirmTitle: "Ghost FTP %s इंस्टॉल करें?", ConfirmBody: "Ghost FTP आपके Windows उपयोगकर्ता खाते के लिए इंस्टॉल होगा और Start मेनू से उपलब्ध रहेगा।",
		CompletedTitle: "सेटअप सफलतापूर्वक पूरा हुआ", ReadyBody: "Ghost FTP उपयोग के लिए तैयार है।", LaunchQuestion: "Ghost FTP अभी शुरू करें?", InstalledTitle: "Ghost FTP इंस्टॉल हो गया है",
		LaunchFailed: "Ghost FTP अपने आप शुरू नहीं हो सका। इसे Windows Start मेनू से शुरू करें।", LanguageWarning: "चुनी हुई भाषा सहेजी नहीं जा सकी। Ghost FTP अंग्रेज़ी में शुरू होगा; भाषा Settings में बदली जा सकती है।",
		ShortcutWarning: "शॉर्टकट नहीं बनाया जा सका। Ghost FTP को इंस्टॉलेशन फ़ोल्डर से शुरू किया जा सकता है।",
	},
	"ja": {
		ConfirmTitle: "Ghost FTP %s をインストールしますか？", ConfirmBody: "Ghost FTP は現在の Windows ユーザー用にインストールされ、スタートメニューから利用できます。",
		CompletedTitle: "セットアップが正常に完了しました", ReadyBody: "Ghost FTP を使用できます。", LaunchQuestion: "Ghost FTP を今すぐ起動しますか？", InstalledTitle: "Ghost FTP はインストール済みです",
		LaunchFailed: "Ghost FTP を自動起動できませんでした。Windows のスタートメニューから起動してください。", LanguageWarning: "選択した言語を保存できませんでした。Ghost FTP は英語で起動します。言語は設定で変更できます。",
		ShortcutWarning: "ショートカットを作成できませんでした。インストールフォルダーから Ghost FTP を起動できます。",
	},
	"it": {
		ConfirmTitle: "Installare Ghost FTP %s?", ConfirmBody: "Ghost FTP verrà installato per il tuo account Windows e sarà disponibile dal menu Start.",
		CompletedTitle: "Installazione completata correttamente", ReadyBody: "Ghost FTP è pronto all'uso.", LaunchQuestion: "Avviare Ghost FTP ora?", InstalledTitle: "Ghost FTP è installato",
		LaunchFailed: "Ghost FTP non è stato avviato automaticamente. Avvialo dal menu Start di Windows.", LanguageWarning: "La lingua selezionata non è stata salvata. Ghost FTP si avvierà in inglese; puoi cambiare lingua nelle Impostazioni.",
		ShortcutWarning: "Non è stato possibile creare un collegamento. Puoi avviare Ghost FTP dalla cartella di installazione.",
	},
	"pl": {
		ConfirmTitle: "Zainstalować Ghost FTP %s?", ConfirmBody: "Ghost FTP zostanie zainstalowany dla Twojego konta Windows i będzie dostępny z menu Start.",
		CompletedTitle: "Instalacja zakończona pomyślnie", ReadyBody: "Ghost FTP jest gotowy do użycia.", LaunchQuestion: "Uruchomić Ghost FTP teraz?", InstalledTitle: "Ghost FTP jest zainstalowany",
		LaunchFailed: "Nie udało się automatycznie uruchomić Ghost FTP. Uruchom program z menu Start systemu Windows.", LanguageWarning: "Nie udało się zapisać wybranego języka. Ghost FTP uruchomi się po angielsku; język można zmienić w Ustawieniach.",
		ShortcutWarning: "Nie udało się utworzyć skrótu. Ghost FTP można uruchomić z folderu instalacyjnego.",
	},
	"nl": {
		ConfirmTitle: "Ghost FTP %s installeren?", ConfirmBody: "Ghost FTP wordt voor uw Windows-account geïnstalleerd en is beschikbaar via het Startmenu.",
		CompletedTitle: "Installatie succesvol voltooid", ReadyBody: "Ghost FTP is klaar voor gebruik.", LaunchQuestion: "Ghost FTP nu starten?", InstalledTitle: "Ghost FTP is geïnstalleerd",
		LaunchFailed: "Ghost FTP kon niet automatisch worden gestart. Start het via het Windows Startmenu.", LanguageWarning: "De gekozen taal kon niet worden opgeslagen. Ghost FTP start in het Engels; u kunt de taal wijzigen in Instellingen.",
		ShortcutWarning: "Er kon geen snelkoppeling worden gemaakt. U kunt Ghost FTP vanuit de installatiemap starten.",
	},
	"cs": {
		ConfirmTitle: "Nainstalovat Ghost FTP %s?", ConfirmBody: "Ghost FTP bude nainstalován pro váš účet Windows a bude dostupný z nabídky Start.",
		CompletedTitle: "Instalace byla úspěšně dokončena", ReadyBody: "Ghost FTP je připraven k použití.", LaunchQuestion: "Spustit Ghost FTP nyní?", InstalledTitle: "Ghost FTP je nainstalován",
		LaunchFailed: "Ghost FTP se nepodařilo spustit automaticky. Spusťte jej z nabídky Start systému Windows.", LanguageWarning: "Vybraný jazyk se nepodařilo uložit. Ghost FTP se spustí anglicky; jazyk můžete změnit v Nastavení.",
		ShortcutWarning: "Zástupce se nepodařilo vytvořit. Ghost FTP můžete spustit z instalační složky.",
	},
	"uk": {
		ConfirmTitle: "Встановити Ghost FTP %s?", ConfirmBody: "Ghost FTP буде встановлено для вашого облікового запису Windows і буде доступний у меню «Пуск».",
		CompletedTitle: "Встановлення успішно завершено", ReadyBody: "Ghost FTP готовий до використання.", LaunchQuestion: "Запустити Ghost FTP зараз?", InstalledTitle: "Ghost FTP встановлено",
		LaunchFailed: "Не вдалося автоматично запустити Ghost FTP. Запустіть його з меню «Пуск» Windows.", LanguageWarning: "Не вдалося зберегти вибрану мову. Ghost FTP запуститься англійською; мову можна змінити в Налаштуваннях.",
		ShortcutWarning: "Не вдалося створити ярлик. Ghost FTP можна запустити з папки встановлення.",
	},
	"sv": {
		ConfirmTitle: "Installera Ghost FTP %s?", ConfirmBody: "Ghost FTP installeras för ditt Windows-konto och blir tillgängligt från Start-menyn.",
		CompletedTitle: "Installationen slutfördes", ReadyBody: "Ghost FTP är klart att använda.", LaunchQuestion: "Starta Ghost FTP nu?", InstalledTitle: "Ghost FTP är installerat",
		LaunchFailed: "Ghost FTP kunde inte startas automatiskt. Starta det från Windows Start-meny.", LanguageWarning: "Det valda språket kunde inte sparas. Ghost FTP startar på engelska; du kan byta språk i Inställningar.",
		ShortcutWarning: "En genväg kunde inte skapas. Du kan starta Ghost FTP från installationsmappen.",
	},
	"ro": {
		ConfirmTitle: "Instalați Ghost FTP %s?", ConfirmBody: "Ghost FTP va fi instalat pentru contul dvs. Windows și va fi disponibil din meniul Start.",
		CompletedTitle: "Instalare finalizată cu succes", ReadyBody: "Ghost FTP este gata de utilizare.", LaunchQuestion: "Porniți Ghost FTP acum?", InstalledTitle: "Ghost FTP este instalat",
		LaunchFailed: "Ghost FTP nu a putut fi pornit automat. Porniți-l din meniul Start Windows.", LanguageWarning: "Limba selectată nu a putut fi salvată. Ghost FTP va porni în engleză; puteți schimba limba în Setări.",
		ShortcutWarning: "Nu s-a putut crea o scurtătură. Puteți porni Ghost FTP din folderul de instalare.",
	},
	"hu": {
		ConfirmTitle: "Telepíti a Ghost FTP %s verziót?", ConfirmBody: "A Ghost FTP a Windows-felhasználói fiókhoz települ, és a Start menüből lesz elérhető.",
		CompletedTitle: "A telepítés sikeresen befejeződött", ReadyBody: "A Ghost FTP használatra kész.", LaunchQuestion: "Elindítja most a Ghost FTP-t?", InstalledTitle: "A Ghost FTP telepítve van",
		LaunchFailed: "A Ghost FTP nem indult el automatikusan. Indítsa el a Windows Start menüből.", LanguageWarning: "A kiválasztott nyelvet nem sikerült menteni. A Ghost FTP angolul indul; a nyelv a Beállításokban módosítható.",
		ShortcutWarning: "Nem sikerült parancsikont létrehozni. A Ghost FTP a telepítési mappából indítható.",
	},
	"da": {
		ConfirmTitle: "Installer Ghost FTP %s?", ConfirmBody: "Ghost FTP installeres til din Windows-brugerkonto og bliver tilgængelig fra Start-menuen.",
		CompletedTitle: "Installationen blev gennemført", ReadyBody: "Ghost FTP er klar til brug.", LaunchQuestion: "Start Ghost FTP nu?", InstalledTitle: "Ghost FTP er installeret",
		LaunchFailed: "Ghost FTP kunne ikke startes automatisk. Start det fra Windows Start-menuen.", LanguageWarning: "Det valgte sprog kunne ikke gemmes. Ghost FTP starter på engelsk; sproget kan ændres i Indstillinger.",
		ShortcutWarning: "Der kunne ikke oprettes en genvej. Du kan starte Ghost FTP fra installationsmappen.",
	},
	"fi": {
		ConfirmTitle: "Asennetaanko Ghost FTP %s?", ConfirmBody: "Ghost FTP asennetaan Windows-käyttäjätilillesi ja se on käytettävissä Käynnistä-valikosta.",
		CompletedTitle: "Asennus valmistui onnistuneesti", ReadyBody: "Ghost FTP on valmis käyttöön.", LaunchQuestion: "Käynnistetäänkö Ghost FTP nyt?", InstalledTitle: "Ghost FTP on asennettu",
		LaunchFailed: "Ghost FTP:tä ei voitu käynnistää automaattisesti. Käynnistä se Windowsin Käynnistä-valikosta.", LanguageWarning: "Valittua kieltä ei voitu tallentaa. Ghost FTP käynnistyy englanniksi; kielen voi vaihtaa Asetuksissa.",
		ShortcutWarning: "Pikakuvaketta ei voitu luoda. Voit käynnistää Ghost FTP:n asennuskansiosta.",
	},
	"no": {
		ConfirmTitle: "Installere Ghost FTP %s?", ConfirmBody: "Ghost FTP installeres for Windows-brukerkontoen din og blir tilgjengelig fra Start-menyen.",
		CompletedTitle: "Installasjonen ble fullført", ReadyBody: "Ghost FTP er klar til bruk.", LaunchQuestion: "Starte Ghost FTP nå?", InstalledTitle: "Ghost FTP er installert",
		LaunchFailed: "Ghost FTP kunne ikke startes automatisk. Start det fra Windows Start-menyen.", LanguageWarning: "Det valgte språket kunne ikke lagres. Ghost FTP starter på engelsk; språket kan endres i Innstillinger.",
		ShortcutWarning: "En snarvei kunne ikke opprettes. Du kan starte Ghost FTP fra installasjonsmappen.",
	},
	"ko": {
		ConfirmTitle: "Ghost FTP %s을(를) 설치하시겠습니까?", ConfirmBody: "Ghost FTP는 현재 Windows 사용자 계정에 설치되며 시작 메뉴에서 사용할 수 있습니다.",
		CompletedTitle: "설치가 완료되었습니다", ReadyBody: "Ghost FTP를 사용할 준비가 되었습니다.", LaunchQuestion: "지금 Ghost FTP를 실행하시겠습니까?", InstalledTitle: "Ghost FTP가 설치되었습니다",
		LaunchFailed: "Ghost FTP를 자동으로 실행하지 못했습니다. Windows 시작 메뉴에서 실행하세요.", LanguageWarning: "선택한 언어를 저장하지 못했습니다. Ghost FTP는 영어로 시작하며 설정에서 언어를 변경할 수 있습니다.",
		ShortcutWarning: "바로 가기를 만들지 못했습니다. 설치 폴더에서 Ghost FTP를 실행할 수 있습니다.",
	},
}

func installerCopyFor(language string) installerCopy {
	base := installerCopies[i18n.DefaultLanguage]
	localized, ok := installerCopies[i18n.Normalize(language)]
	if !ok {
		return base
	}
	if localized.ConfirmTitle == "" {
		localized.ConfirmTitle = base.ConfirmTitle
	}
	if localized.ConfirmBody == "" {
		localized.ConfirmBody = base.ConfirmBody
	}
	if localized.CompletedTitle == "" {
		localized.CompletedTitle = base.CompletedTitle
	}
	if localized.ReadyBody == "" {
		localized.ReadyBody = base.ReadyBody
	}
	if localized.LaunchQuestion == "" {
		localized.LaunchQuestion = base.LaunchQuestion
	}
	if localized.InstalledTitle == "" {
		localized.InstalledTitle = base.InstalledTitle
	}
	if localized.LaunchFailed == "" {
		localized.LaunchFailed = base.LaunchFailed
	}
	if localized.LanguageWarning == "" {
		localized.LanguageWarning = base.LanguageWarning
	}
	if localized.ShortcutWarning == "" {
		localized.ShortcutWarning = base.ShortcutWarning
	}
	return localized
}

func installerConfirmTitle(language, version string) string {
	return fmt.Sprintf(installerCopyFor(language).ConfirmTitle, version)
}
