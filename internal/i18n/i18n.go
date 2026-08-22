package i18n

import (
	"fmt"
	"sort"
	"strings"
)

const DefaultLanguage = "en"

type Language struct {
	Code       string
	EnglishName string
	NativeName string
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
}

func Languages() []Language {
	out := make([]Language, len(supportedLanguages))
	copy(out, supportedLanguages)
	return out
}

func IsSupported(code string) bool {
	code = Normalize(code)
	_, ok := catalogs[code]
	return ok
}

func Normalize(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "_", "-")
	if code == "" {
		return DefaultLanguage
	}
	if _, ok := catalogs[code]; ok {
		return code
	}
	if i := strings.IndexByte(code, '-'); i > 0 {
		if primary := code[:i]; catalogs[primary] != nil {
			return primary
		}
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

func ValidateCatalogs() error {
	english := catalogs[DefaultLanguage]
	if len(english) == 0 {
		return fmt.Errorf("English localization catalog is empty")
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
