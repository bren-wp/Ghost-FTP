package i18n

import "strings"

func supportedLanguageCodeList() string {
	codes := make([]string, 0, len(supportedLanguages))
	for _, language := range supportedLanguages {
		codes = append(codes, language.Code)
	}
	return strings.Join(codes, "|")
}

func expandTerminalLanguageUsage(raw string) string {
	codes := supportedLanguageCodeList()
	open := strings.IndexByte(raw, '<')
	close := strings.LastIndexByte(raw, '>')
	if open < 0 || close <= open {
		return "Usage: language <" + codes + ">"
	}
	return raw[:open+1] + codes + raw[close:]
}

func init() {
	// The language command accepts every language in the canonical registry.
	// Expand the localized usage template from that registry at startup so a
	// newly added locale cannot be accepted by the engine while being omitted
	// from terminal help. The surrounding translated text remains untouched.
	for _, catalog := range catalogs {
		catalog["terminal.language_usage"] = expandTerminalLanguageUsage(catalog["terminal.language_usage"])
	}
}
