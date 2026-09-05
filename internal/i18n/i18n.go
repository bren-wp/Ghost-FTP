package i18n

import (
	"fmt"
	"sort"
	"strings"
)

const DefaultLanguage = "en"

type Language struct {
	Code        string
	EnglishName string
	NativeName  string
}

var supportedLanguages = []Language{
	{Code: "en", EnglishName: "English", NativeName: "English"},
	{Code: "hr", EnglishName: "Croatian", NativeName: "Hrvatski"},
	{Code: "de", EnglishName: "German", NativeName: "Deutsch"},
	{Code: "fr", EnglishName: "French", NativeName: "Français"},
	{Code: "es", EnglishName: "Spanish", NativeName: "Español"},
	{Code: "tr", EnglishName: "Turkish", NativeName: "Türkçe"},
	{Code: "el", EnglishName: "Greek", NativeName: "Ελληνικά"},
	{Code: "pt", EnglishName: "Portuguese", NativeName: "Português"},
	{Code: "zh", EnglishName: "Chinese (Simplified)", NativeName: "简体中文"},
	{Code: "ru", EnglishName: "Russian", NativeName: "Русский"},
	{Code: "hi", EnglishName: "Hindi", NativeName: "हिन्दी"},
	{Code: "ja", EnglishName: "Japanese", NativeName: "日本語"},
	{Code: "it", EnglishName: "Italian", NativeName: "Italiano"},
	{Code: "pl", EnglishName: "Polish", NativeName: "Polski"},
	{Code: "nl", EnglishName: "Dutch", NativeName: "Nederlands"},
	{Code: "cs", EnglishName: "Czech", NativeName: "Čeština"},
	{Code: "uk", EnglishName: "Ukrainian", NativeName: "Українська"},
	{Code: "sv", EnglishName: "Swedish", NativeName: "Svenska"},
	{Code: "ro", EnglishName: "Romanian", NativeName: "Română"},
	{Code: "hu", EnglishName: "Hungarian", NativeName: "Magyar"},
	{Code: "da", EnglishName: "Danish", NativeName: "Dansk"},
	{Code: "fi", EnglishName: "Finnish", NativeName: "Suomi"},
	{Code: "no", EnglishName: "Norwegian", NativeName: "Norsk"},
	{Code: "ko", EnglishName: "Korean", NativeName: "한국어"},
}

var languageAliases = map[string]string{
	"nb": "no",
	"nn": "no",
	"zh-cn": "zh",
	"zh-hans": "zh",
	"zh-sg": "zh",
}

func Languages() []Language {
	out := make([]Language, len(supportedLanguages))
	copy(out, supportedLanguages)
	return out
}

func canonical(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "_", "-")
	if code == "" {
		return ""
	}
	if alias, ok := languageAliases[code]; ok {
		code = alias
	}
	if _, ok := catalogs[code]; ok {
		return code
	}
	if i := strings.IndexByte(code, '-'); i > 0 {
		primary := code[:i]
		if alias, ok := languageAliases[primary]; ok {
			primary = alias
		}
		if _, ok := catalogs[primary]; ok {
			return primary
		}
	}
	return ""
}

func IsSupported(code string) bool { return canonical(code) != "" }

func Normalize(code string) string {
	if normalized := canonical(code); normalized != "" {
		return normalized
	}
	return DefaultLanguage
}

func LanguageByCode(code string) Language {
	code = Normalize(code)
	for _, language := range supportedLanguages {
		if language.Code == code {
			return language
		}
	}
	return supportedLanguages[0]
}

func T(language, key string, args ...any) string {
	language = Normalize(language)
	template := ""
	if catalog := catalogs[language]; catalog != nil {
		template = catalog[key]
	}
	if template == "" {
		template = catalogs[DefaultLanguage][key]
	}
	if template == "" {
		template = key
	}
	if len(args) == 0 {
		return template
	}
	return fmt.Sprintf(template, args...)
}

func IsAffirmative(language, answer string) bool {
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer == "y" || answer == "yes" {
		return true
	}
	accepted := map[string][]string{
		"en": {"yes"}, "hr": {"da", "d"}, "de": {"ja", "j"}, "fr": {"oui", "o"},
		"es": {"sí", "si", "s"}, "tr": {"evet", "e"}, "el": {"ναι", "ν"}, "pt": {"sim", "s"},
		"zh": {"是", "是的"}, "ru": {"да", "д"}, "hi": {"हाँ", "हां", "ह"}, "ja": {"はい"},
		"it": {"sì", "si", "s"}, "pl": {"tak", "t"}, "nl": {"ja", "j"}, "cs": {"ano", "a"},
		"uk": {"так", "т"}, "sv": {"ja", "j"}, "ro": {"da", "d"}, "hu": {"igen", "i"},
		"da": {"ja", "j"}, "fi": {"kyllä", "kylla", "k"}, "no": {"ja", "j"}, "ko": {"예", "네"},
	}
	for _, value := range accepted[Normalize(language)] {
		if answer == value {
			return true
		}
	}
	return false
}

// TranslationCoverage reports how many runtime strings differ from the canonical
// English catalog. It is intentionally simple: CI uses it to prevent a locale
// from being advertised while almost every user-visible string is only the
// English fallback.
func TranslationCoverage(code string) (translated, total int) {
	code = Normalize(code)
	english := catalogs[DefaultLanguage]
	catalog := catalogs[code]
	for key, englishValue := range english {
		total++
		if value, ok := catalog[key]; ok && strings.TrimSpace(value) != "" && value != englishValue {
			translated++
		}
	}
	return translated, total
}

func ValidateCatalogs() error {
	english := catalogs[DefaultLanguage]
	if len(english) == 0 {
		return fmt.Errorf("English localization catalog is empty")
	}
	if len(supportedLanguages) < 21 {
		return fmt.Errorf("Ghost FTP must expose at least 21 supported languages")
	}
	if supportedLanguages[0].Code != DefaultLanguage {
		return fmt.Errorf("English must be the first and default supported language")
	}
	codes := make(map[string]struct{}, len(supportedLanguages))
	for _, language := range supportedLanguages {
		if _, exists := codes[language.Code]; exists {
			return fmt.Errorf("duplicate language code %q", language.Code)
		}
		codes[language.Code] = struct{}{}
		catalog := catalogs[language.Code]
		if catalog == nil {
			return fmt.Errorf("missing catalog for %s", language.Code)
		}
		if len(catalog) != len(english) {
			return fmt.Errorf("catalog %s has %d keys; English has %d", language.Code, len(catalog), len(english))
		}
		for key, englishValue := range english {
			if strings.TrimSpace(englishValue) == "" {
				return fmt.Errorf("empty English localization value for %q", key)
			}
			if strings.TrimSpace(catalog[key]) == "" {
				return fmt.Errorf("catalog %s is missing key %q", language.Code, key)
			}
		}
		for key := range catalog {
			if _, ok := english[key]; !ok {
				return fmt.Errorf("catalog %s has unknown key %q", language.Code, key)
			}
		}
	}
	if len(catalogs) != len(codes) {
		extra := make([]string, 0)
		for code := range catalogs {
			if _, ok := codes[code]; !ok {
				extra = append(extra, code)
			}
		}
		sort.Strings(extra)
		return fmt.Errorf("catalogs contain unsupported language codes: %s", strings.Join(extra, ", "))
	}
	return nil
}
