package desktop

import "github.com/bren-wp/Ghost-FTP/internal/i18n"

type conflictPolicyWords struct {
	Title         string
	Skip          string
	Replace       string
	ReplaceBackup string
}

var localizedConflictPolicyWords = map[string]conflictPolicyWords{
	"en": {"When a destination file already exists", "Skip existing files", "Replace existing files", "Replace and keep a backup"},
	"hr": {"Kada odredišna datoteka već postoji", "Preskoči postojeće datoteke", "Zamijeni postojeće datoteke", "Zamijeni i zadrži sigurnosnu kopiju"},
	"de": {"Wenn eine Zieldatei bereits vorhanden ist", "Vorhandene Dateien überspringen", "Vorhandene Dateien ersetzen", "Ersetzen und Sicherung behalten"},
	"fr": {"Lorsqu’un fichier de destination existe déjà", "Ignorer les fichiers existants", "Remplacer les fichiers existants", "Remplacer et conserver une sauvegarde"},
	"es": {"Cuando ya existe un archivo de destino", "Omitir archivos existentes", "Reemplazar archivos existentes", "Reemplazar y conservar una copia de seguridad"},
	"tr": {"Hedef dosya zaten varsa", "Mevcut dosyaları atla", "Mevcut dosyaları değiştir", "Değiştir ve yedeği koru"},
	"el": {"Όταν υπάρχει ήδη αρχείο προορισμού", "Παράλειψη υπαρχόντων αρχείων", "Αντικατάσταση υπαρχόντων αρχείων", "Αντικατάσταση και διατήρηση αντιγράφου ασφαλείας"},
	"pt": {"Quando um arquivo de destino já existe", "Ignorar arquivos existentes", "Substituir arquivos existentes", "Substituir e manter uma cópia de segurança"},
	"zh": {"目标文件已存在时", "跳过现有文件", "替换现有文件", "替换并保留备份"},
	"ru": {"Если файл назначения уже существует", "Пропускать существующие файлы", "Заменять существующие файлы", "Заменять и сохранять резервную копию"},
	"hi": {"जब गंतव्य फ़ाइल पहले से मौजूद हो", "मौजूदा फ़ाइलें छोड़ें", "मौजूदा फ़ाइलें बदलें", "बदलें और बैकअप रखें"},
	"ja": {"保存先に同名ファイルがある場合", "既存ファイルをスキップ", "既存ファイルを置換", "置換してバックアップを保持"},
	"it": {"Quando un file di destinazione esiste già", "Ignora i file esistenti", "Sostituisci i file esistenti", "Sostituisci e conserva un backup"},
	"pl": {"Gdy plik docelowy już istnieje", "Pomiń istniejące pliki", "Zastąp istniejące pliki", "Zastąp i zachowaj kopię zapasową"},
	"nl": {"Wanneer een doelbestand al bestaat", "Bestaande bestanden overslaan", "Bestaande bestanden vervangen", "Vervangen en een back-up bewaren"},
	"cs": {"Když cílový soubor již existuje", "Přeskočit existující soubory", "Nahradit existující soubory", "Nahradit a zachovat zálohu"},
	"uk": {"Якщо файл призначення вже існує", "Пропускати наявні файли", "Замінювати наявні файли", "Замінювати й зберігати резервну копію"},
	"sv": {"När en målfil redan finns", "Hoppa över befintliga filer", "Ersätt befintliga filer", "Ersätt och behåll en säkerhetskopia"},
	"ro": {"Când un fișier de destinație există deja", "Omite fișierele existente", "Înlocuiește fișierele existente", "Înlocuiește și păstrează o copie de siguranță"},
	"hu": {"Ha a célfájl már létezik", "Meglévő fájlok kihagyása", "Meglévő fájlok cseréje", "Csere és biztonsági másolat megtartása"},
	"da": {"Når en destinationsfil allerede findes", "Spring eksisterende filer over", "Erstat eksisterende filer", "Erstat og behold en sikkerhedskopi"},
	"fi": {"Kun kohdetiedosto on jo olemassa", "Ohita olemassa olevat tiedostot", "Korvaa olemassa olevat tiedostot", "Korvaa ja säilytä varmuuskopio"},
	"no": {"Når en målfil allerede finnes", "Hopp over eksisterende filer", "Erstatt eksisterende filer", "Erstatt og behold en sikkerhetskopi"},
	"ko": {"대상 파일이 이미 있는 경우", "기존 파일 건너뛰기", "기존 파일 바꾸기", "바꾸고 백업 유지"},
}

func conflictPolicyText(language string) conflictPolicyWords {
	if words, ok := localizedConflictPolicyWords[i18n.Normalize(language)]; ok {
		return words
	}
	return localizedConflictPolicyWords[i18n.DefaultLanguage]
}
