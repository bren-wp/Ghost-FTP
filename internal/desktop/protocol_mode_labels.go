package desktop

import "github.com/bren-wp/Ghost-FTP/internal/i18n"

var protocolModeWords = map[string][2]string{
	"en": {"explicit", "implicit"},
	"hr": {"eksplicitni", "implicitni"},
	"de": {"explizit", "implizit"},
	"fr": {"explicite", "implicite"},
	"es": {"explícito", "implícito"},
	"tr": {"açık", "örtük"},
	"el": {"ρητό", "έμμεσο"},
	"pt": {"explícito", "implícito"},
	"zh": {"显式", "隐式"},
	"ru": {"явный", "неявный"},
	"hi": {"स्पष्ट", "निहित"},
	"ja": {"明示", "暗黙"},
	"it": {"esplicito", "implicito"},
	"pl": {"jawny", "niejawny"},
	"nl": {"expliciet", "impliciet"},
	"cs": {"explicitní", "implicitní"},
	"uk": {"явний", "неявний"},
	"sv": {"explicit", "implicit"},
	"ro": {"explicit", "implicit"},
	"hu": {"explicit", "implicit"},
	"da": {"eksplicit", "implicit"},
	"fi": {"eksplisiittinen", "implisiittinen"},
	"no": {"eksplisitt", "implisitt"},
	"ko": {"명시적", "암시적"},
}

func protocolModeLabelWords(language string) [2]string {
	pair, ok := protocolModeWords[i18n.Normalize(language)]
	if !ok || pair[0] == "" || pair[1] == "" {
		return protocolModeWords[i18n.DefaultLanguage]
	}
	return pair
}
