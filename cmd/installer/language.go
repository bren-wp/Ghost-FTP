package main

import (
	"fmt"

	"github.com/bren-wp/by-ftp/internal/appdata"
	"github.com/bren-wp/by-ftp/internal/brand"
	"github.com/bren-wp/by-ftp/internal/config"
	"github.com/bren-wp/by-ftp/internal/i18n"
	"github.com/bren-wp/by-ftp/internal/platform"
)

func selectInstallerLanguage() (string, bool) {
	languages := i18n.Languages()
	options := make([]string, 0, len(languages))
	for _, language := range languages {
		label := language.EnglishName
		if language.NativeName != "" && language.NativeName != language.EnglishName {
			label += " — " + language.NativeName
		}
		options = append(options, label)
	}

	index, ok := platform.SelectLanguageDialog(
		brand.ProductName+" Setup",
		"Choose the language to use in Ghost FTP. You can change it later in Settings.",
		options,
		0,
	)
	if !ok {
		return "", false
	}
	if index < 0 || index >= len(languages) {
		index = 0
	}
	return languages[index].Code, true
}

func persistInstallerLanguage(language string) error {
	if !i18n.IsSupported(language) {
		return fmt.Errorf("unsupported installer language %q", language)
	}
	dataDir, err := appdata.Dir()
	if err != nil {
		return err
	}
	settings := config.NewSettings(config.New(dataDir))
	current, err := settings.Get()
	if err != nil {
		return err
	}
	current.Language = i18n.Normalize(language)
	_, err = settings.Set(current)
	return err
}
