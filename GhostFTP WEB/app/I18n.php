<?php
declare(strict_types=1);

namespace GhostFTP;

final class I18n
{
    public const DEFAULT_LANGUAGE = 'en';

    private const LANGUAGES = [
        'en' => ['English', 'English'],
        'hr' => ['Croatian', 'Hrvatski'],
        'de' => ['German', 'Deutsch'],
        'fr' => ['French', 'Français'],
        'es' => ['Spanish', 'Español'],
        'tr' => ['Turkish', 'Türkçe'],
        'el' => ['Greek', 'Ελληνικά'],
        'pt' => ['Portuguese', 'Português'],
        'zh' => ['Chinese (Simplified)', '简体中文'],
        'ru' => ['Russian', 'Русский'],
        'hi' => ['Hindi', 'हिन्दी'],
        'ja' => ['Japanese', '日本語'],
        'it' => ['Italian', 'Italiano'],
        'pl' => ['Polish', 'Polski'],
        'nl' => ['Dutch', 'Nederlands'],
        'cs' => ['Czech', 'Čeština'],
        'uk' => ['Ukrainian', 'Українська'],
        'sv' => ['Swedish', 'Svenska'],
        'ro' => ['Romanian', 'Română'],
        'hu' => ['Hungarian', 'Magyar'],
        'da' => ['Danish', 'Dansk'],
        'fi' => ['Finnish', 'Suomi'],
        'no' => ['Norwegian', 'Norsk'],
        'ko' => ['Korean', '한국어'],
    ];

    private const ALIASES = [
        'nb' => 'no',
        'nn' => 'no',
        'zh-cn' => 'zh',
        'zh-hans' => 'zh',
        'zh-sg' => 'zh',
    ];

    /**
     * English is the canonical Web/PWA source catalog. Every locale is merged
     * over it so incomplete future additions fail safely to English instead of
     * leaking a raw key or a different language into the UI.
     */
    private const CORE = [
        'en' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Private, direct file transfer',
            'connection.saved' => 'Saved connection',
            'connection.forget' => 'Forget saved connection',
            'connection.title' => 'Connection',
            'connection.protocol' => 'Protocol',
            'connection.host' => 'Host',
            'connection.username' => 'Username',
            'connection.password' => 'Password',
            'connection.ftp_warning' => 'FTP is unencrypted. Prefer FTPS whenever the server supports it.',
            'connection.connecting' => 'Connecting…',
            'connection.connect' => 'Connect',
        ],
        'hr' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Privatan, izravan prijenos datoteka',
            'connection.saved' => 'Spremljena veza',
            'connection.forget' => 'Zaboravi spremljenu vezu',
            'connection.title' => 'Veza',
            'connection.protocol' => 'Protokol',
            'connection.host' => 'Poslužitelj',
            'connection.username' => 'Korisničko ime',
            'connection.password' => 'Lozinka',
            'connection.ftp_warning' => 'FTP nije šifriran. Koristite FTPS kad ga poslužitelj podržava.',
            'connection.connecting' => 'Povezivanje…',
            'connection.connect' => 'Poveži se',
        ],
        'de' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Privater, direkter Dateitransfer',
            'connection.saved' => 'Gespeicherte Verbindung',
            'connection.forget' => 'Gespeicherte Verbindung vergessen',
            'connection.title' => 'Verbindung',
            'connection.protocol' => 'Protokoll',
            'connection.host' => 'Host',
            'connection.username' => 'Benutzername',
            'connection.password' => 'Passwort',
            'connection.ftp_warning' => 'FTP ist unverschlüsselt. Bevorzugen Sie FTPS, wenn der Server es unterstützt.',
            'connection.connecting' => 'Verbindung wird hergestellt…',
            'connection.connect' => 'Verbinden',
        ],
        'fr' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Transfert de fichiers privé et direct',
            'connection.saved' => 'Connexion enregistrée',
            'connection.forget' => 'Oublier la connexion enregistrée',
            'connection.title' => 'Connexion',
            'connection.protocol' => 'Protocole',
            'connection.host' => 'Hôte',
            'connection.username' => 'Nom d’utilisateur',
            'connection.password' => 'Mot de passe',
            'connection.ftp_warning' => 'FTP n’est pas chiffré. Préférez FTPS lorsque le serveur le prend en charge.',
            'connection.connecting' => 'Connexion…',
            'connection.connect' => 'Se connecter',
        ],
        'es' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Transferencia de archivos privada y directa',
            'connection.saved' => 'Conexión guardada',
            'connection.forget' => 'Olvidar conexión guardada',
            'connection.title' => 'Conexión',
            'connection.protocol' => 'Protocolo',
            'connection.host' => 'Servidor',
            'connection.username' => 'Nombre de usuario',
            'connection.password' => 'Contraseña',
            'connection.ftp_warning' => 'FTP no está cifrado. Prefiera FTPS cuando el servidor lo admita.',
            'connection.connecting' => 'Conectando…',
            'connection.connect' => 'Conectar',
        ],
        'tr' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Özel, doğrudan dosya aktarımı',
            'connection.saved' => 'Kayıtlı bağlantı',
            'connection.forget' => 'Kayıtlı bağlantıyı unut',
            'connection.title' => 'Bağlantı',
            'connection.protocol' => 'Protokol',
            'connection.host' => 'Sunucu',
            'connection.username' => 'Kullanıcı adı',
            'connection.password' => 'Parola',
            'connection.ftp_warning' => 'FTP şifrelenmemiştir. Sunucu destekliyorsa FTPS kullanın.',
            'connection.connecting' => 'Bağlanıyor…',
            'connection.connect' => 'Bağlan',
        ],
        'el' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Ιδιωτική, άμεση μεταφορά αρχείων',
            'connection.saved' => 'Αποθηκευμένη σύνδεση',
            'connection.forget' => 'Διαγραφή αποθηκευμένης σύνδεσης',
            'connection.title' => 'Σύνδεση',
            'connection.protocol' => 'Πρωτόκολλο',
            'connection.host' => 'Κεντρικός υπολογιστής',
            'connection.username' => 'Όνομα χρήστη',
            'connection.password' => 'Κωδικός πρόσβασης',
            'connection.ftp_warning' => 'Το FTP δεν είναι κρυπτογραφημένο. Προτιμήστε FTPS όταν το υποστηρίζει ο διακομιστής.',
            'connection.connecting' => 'Σύνδεση…',
            'connection.connect' => 'Σύνδεση',
        ],
        'pt' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Transferência de ficheiros privada e direta',
            'connection.saved' => 'Ligação guardada',
            'connection.forget' => 'Esquecer ligação guardada',
            'connection.title' => 'Ligação',
            'connection.protocol' => 'Protocolo',
            'connection.host' => 'Anfitrião',
            'connection.username' => 'Nome de utilizador',
            'connection.password' => 'Palavra-passe',
            'connection.ftp_warning' => 'FTP não é cifrado. Prefira FTPS quando o servidor o suportar.',
            'connection.connecting' => 'A ligar…',
            'connection.connect' => 'Ligar',
        ],
        'zh' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => '私密、直接的文件传输',
            'connection.saved' => '已保存的连接',
            'connection.forget' => '忘记已保存的连接',
            'connection.title' => '连接',
            'connection.protocol' => '协议',
            'connection.host' => '主机',
            'connection.username' => '用户名',
            'connection.password' => '密码',
            'connection.ftp_warning' => 'FTP 未加密。服务器支持时请优先使用 FTPS。',
            'connection.connecting' => '正在连接…',
            'connection.connect' => '连接',
        ],
        'ru' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Прямая и конфиденциальная передача файлов',
            'connection.saved' => 'Сохранённое подключение',
            'connection.forget' => 'Забыть сохранённое подключение',
            'connection.title' => 'Подключение',
            'connection.protocol' => 'Протокол',
            'connection.host' => 'Хост',
            'connection.username' => 'Имя пользователя',
            'connection.password' => 'Пароль',
            'connection.ftp_warning' => 'FTP не шифруется. Используйте FTPS, если сервер его поддерживает.',
            'connection.connecting' => 'Подключение…',
            'connection.connect' => 'Подключиться',
        ],
        'hi' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'निजी, सीधा फ़ाइल स्थानांतरण',
            'connection.saved' => 'सहेजा गया कनेक्शन',
            'connection.forget' => 'सहेजा गया कनेक्शन भूलें',
            'connection.title' => 'कनेक्शन',
            'connection.protocol' => 'प्रोटोकॉल',
            'connection.host' => 'होस्ट',
            'connection.username' => 'उपयोगकर्ता नाम',
            'connection.password' => 'पासवर्ड',
            'connection.ftp_warning' => 'FTP एन्क्रिप्टेड नहीं है। सर्वर समर्थन करे तो FTPS को प्राथमिकता दें।',
            'connection.connecting' => 'कनेक्ट हो रहा है…',
            'connection.connect' => 'कनेक्ट करें',
        ],
        'ja' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'プライベートな直接ファイル転送',
            'connection.saved' => '保存済み接続',
            'connection.forget' => '保存済み接続を削除',
            'connection.title' => '接続',
            'connection.protocol' => 'プロトコル',
            'connection.host' => 'ホスト',
            'connection.username' => 'ユーザー名',
            'connection.password' => 'パスワード',
            'connection.ftp_warning' => 'FTP は暗号化されません。サーバーが対応している場合は FTPS を使用してください。',
            'connection.connecting' => '接続中…',
            'connection.connect' => '接続',
        ],
        'it' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Trasferimento file privato e diretto',
            'connection.saved' => 'Connessione salvata',
            'connection.forget' => 'Dimentica connessione salvata',
            'connection.title' => 'Connessione',
            'connection.protocol' => 'Protocollo',
            'connection.host' => 'Host',
            'connection.username' => 'Nome utente',
            'connection.password' => 'Password',
            'connection.ftp_warning' => 'FTP non è crittografato. Preferisci FTPS quando il server lo supporta.',
            'connection.connecting' => 'Connessione…',
            'connection.connect' => 'Connetti',
        ],
        'pl' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Prywatny, bezpośredni transfer plików',
            'connection.saved' => 'Zapisane połączenie',
            'connection.forget' => 'Zapomnij zapisane połączenie',
            'connection.title' => 'Połączenie',
            'connection.protocol' => 'Protokół',
            'connection.host' => 'Host',
            'connection.username' => 'Nazwa użytkownika',
            'connection.password' => 'Hasło',
            'connection.ftp_warning' => 'FTP nie jest szyfrowany. Używaj FTPS, gdy serwer go obsługuje.',
            'connection.connecting' => 'Łączenie…',
            'connection.connect' => 'Połącz',
        ],
        'nl' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Privé, directe bestandsoverdracht',
            'connection.saved' => 'Opgeslagen verbinding',
            'connection.forget' => 'Opgeslagen verbinding vergeten',
            'connection.title' => 'Verbinding',
            'connection.protocol' => 'Protocol',
            'connection.host' => 'Host',
            'connection.username' => 'Gebruikersnaam',
            'connection.password' => 'Wachtwoord',
            'connection.ftp_warning' => 'FTP is niet versleuteld. Gebruik bij voorkeur FTPS wanneer de server dit ondersteunt.',
            'connection.connecting' => 'Verbinden…',
            'connection.connect' => 'Verbinden',
        ],
        'cs' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Soukromý, přímý přenos souborů',
            'connection.saved' => 'Uložené připojení',
            'connection.forget' => 'Zapomenout uložené připojení',
            'connection.title' => 'Připojení',
            'connection.protocol' => 'Protokol',
            'connection.host' => 'Hostitel',
            'connection.username' => 'Uživatelské jméno',
            'connection.password' => 'Heslo',
            'connection.ftp_warning' => 'FTP není šifrované. Pokud to server podporuje, upřednostněte FTPS.',
            'connection.connecting' => 'Připojování…',
            'connection.connect' => 'Připojit',
        ],
        'uk' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Пряме та приватне передавання файлів',
            'connection.saved' => 'Збережене підключення',
            'connection.forget' => 'Забути збережене підключення',
            'connection.title' => 'Підключення',
            'connection.protocol' => 'Протокол',
            'connection.host' => 'Хост',
            'connection.username' => 'Ім’я користувача',
            'connection.password' => 'Пароль',
            'connection.ftp_warning' => 'FTP не шифрується. Використовуйте FTPS, якщо сервер його підтримує.',
            'connection.connecting' => 'Підключення…',
            'connection.connect' => 'Підключитися',
        ],
        'sv' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Privat, direkt filöverföring',
            'connection.saved' => 'Sparad anslutning',
            'connection.forget' => 'Glöm sparad anslutning',
            'connection.title' => 'Anslutning',
            'connection.protocol' => 'Protokoll',
            'connection.host' => 'Värd',
            'connection.username' => 'Användarnamn',
            'connection.password' => 'Lösenord',
            'connection.ftp_warning' => 'FTP är okrypterat. Föredra FTPS när servern stöder det.',
            'connection.connecting' => 'Ansluter…',
            'connection.connect' => 'Anslut',
        ],
        'ro' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Transfer privat și direct de fișiere',
            'connection.saved' => 'Conexiune salvată',
            'connection.forget' => 'Uită conexiunea salvată',
            'connection.title' => 'Conexiune',
            'connection.protocol' => 'Protocol',
            'connection.host' => 'Gazdă',
            'connection.username' => 'Nume utilizator',
            'connection.password' => 'Parolă',
            'connection.ftp_warning' => 'FTP nu este criptat. Preferă FTPS când serverul îl acceptă.',
            'connection.connecting' => 'Se conectează…',
            'connection.connect' => 'Conectare',
        ],
        'hu' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Privát, közvetlen fájlátvitel',
            'connection.saved' => 'Mentett kapcsolat',
            'connection.forget' => 'Mentett kapcsolat elfelejtése',
            'connection.title' => 'Kapcsolat',
            'connection.protocol' => 'Protokoll',
            'connection.host' => 'Kiszolgáló',
            'connection.username' => 'Felhasználónév',
            'connection.password' => 'Jelszó',
            'connection.ftp_warning' => 'Az FTP nincs titkosítva. Ha a kiszolgáló támogatja, használjon FTPS-t.',
            'connection.connecting' => 'Kapcsolódás…',
            'connection.connect' => 'Kapcsolódás',
        ],
        'da' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Privat, direkte filoverførsel',
            'connection.saved' => 'Gemt forbindelse',
            'connection.forget' => 'Glem gemt forbindelse',
            'connection.title' => 'Forbindelse',
            'connection.protocol' => 'Protokol',
            'connection.host' => 'Vært',
            'connection.username' => 'Brugernavn',
            'connection.password' => 'Adgangskode',
            'connection.ftp_warning' => 'FTP er ukrypteret. Foretræk FTPS, når serveren understøtter det.',
            'connection.connecting' => 'Opretter forbindelse…',
            'connection.connect' => 'Forbind',
        ],
        'fi' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Yksityinen, suora tiedostonsiirto',
            'connection.saved' => 'Tallennettu yhteys',
            'connection.forget' => 'Unohda tallennettu yhteys',
            'connection.title' => 'Yhteys',
            'connection.protocol' => 'Protokolla',
            'connection.host' => 'Palvelin',
            'connection.username' => 'Käyttäjänimi',
            'connection.password' => 'Salasana',
            'connection.ftp_warning' => 'FTP ei ole salattu. Suosi FTPS:ää, kun palvelin tukee sitä.',
            'connection.connecting' => 'Yhdistetään…',
            'connection.connect' => 'Yhdistä',
        ],
        'no' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => 'Privat, direkte filoverføring',
            'connection.saved' => 'Lagret tilkobling',
            'connection.forget' => 'Glem lagret tilkobling',
            'connection.title' => 'Tilkobling',
            'connection.protocol' => 'Protokoll',
            'connection.host' => 'Vert',
            'connection.username' => 'Brukernavn',
            'connection.password' => 'Passord',
            'connection.ftp_warning' => 'FTP er ukryptert. Foretrekk FTPS når serveren støtter det.',
            'connection.connecting' => 'Kobler til…',
            'connection.connect' => 'Koble til',
        ],
        'ko' => [
            'app.name' => 'Ghost FTP',
            'app.subtitle' => '비공개 직접 파일 전송',
            'connection.saved' => '저장된 연결',
            'connection.forget' => '저장된 연결 지우기',
            'connection.title' => '연결',
            'connection.protocol' => '프로토콜',
            'connection.host' => '호스트',
            'connection.username' => '사용자 이름',
            'connection.password' => '비밀번호',
            'connection.ftp_warning' => 'FTP는 암호화되지 않습니다. 서버가 지원하면 FTPS를 사용하세요.',
            'connection.connecting' => '연결 중…',
            'connection.connect' => '연결',
        ],
    ];

    public static function supportedCodes(): array
    {
        return array_keys(self::LANGUAGES);
    }

    public static function languages(): array
    {
        $out = [];
        foreach (self::LANGUAGES as $code => [$englishName, $nativeName]) {
            $out[] = ['code' => $code, 'englishName' => $englishName, 'nativeName' => $nativeName];
        }
        return $out;
    }

    public static function normalize(string $code): string
    {
        $code = strtolower(trim(str_replace('_', '-', $code)));
        if ($code === '') {
            return self::DEFAULT_LANGUAGE;
        }
        $code = self::ALIASES[$code] ?? $code;
        if (isset(self::LANGUAGES[$code])) {
            return $code;
        }
        $separator = strpos($code, '-');
        if ($separator !== false) {
            $primary = substr($code, 0, $separator);
            $primary = self::ALIASES[$primary] ?? $primary;
            if (isset(self::LANGUAGES[$primary])) {
                return $primary;
            }
        }
        return self::DEFAULT_LANGUAGE;
    }

    public static function catalog(string $code): array
    {
        $language = self::normalize($code);
        return array_replace(self::CORE[self::DEFAULT_LANGUAGE], self::CORE[$language] ?? []);
    }

    public static function t(string $code, string $key): string
    {
        $catalog = self::catalog($code);
        return $catalog[$key] ?? self::CORE[self::DEFAULT_LANGUAGE][$key] ?? $key;
    }
}
