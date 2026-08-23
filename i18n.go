package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"golang.org/x/text/language"
)

// localeFiles is embedded so localized pages work in the final scratch image.
//
//go:embed locales/*.json
var localeFiles embed.FS

type locale string

const (
	localeEN   locale = "en"
	localePtBR locale = "pt-BR"
	localeEsAR locale = "es-AR"
)

var supportedLocales = []locale{localeEN, localePtBR, localeEsAR}

var acceptLanguageMatcher = language.NewMatcher([]language.Tag{
	language.English,
	language.BrazilianPortuguese,
	language.MustParse("es-AR"),
})

type translationCatalog struct {
	values map[locale]map[string]string
}

func mustLoadTranslationCatalog() *translationCatalog {
	catalog, err := loadTranslationCatalog(localeFiles)
	if err != nil {
		panic(err)
	}
	return catalog
}

func loadTranslationCatalog(files interface {
	ReadFile(string) ([]byte, error)
}) (*translationCatalog, error) {
	values := make(map[locale]map[string]string, len(supportedLocales))
	var referenceKeys []string

	for _, currentLocale := range supportedLocales {
		data, err := files.ReadFile("locales/" + string(currentLocale) + ".json")
		if err != nil {
			return nil, fmt.Errorf("read locale %s: %w", currentLocale, err)
		}

		translations := make(map[string]string)
		if err := json.Unmarshal(data, &translations); err != nil {
			return nil, fmt.Errorf("parse locale %s: %w", currentLocale, err)
		}
		if len(translations) == 0 {
			return nil, fmt.Errorf("locale %s is empty", currentLocale)
		}
		for key, value := range translations {
			if strings.TrimSpace(value) == "" {
				return nil, fmt.Errorf("locale %s has empty translation for %q", currentLocale, key)
			}
		}

		keys := make([]string, 0, len(translations))
		for key := range translations {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if referenceKeys == nil {
			referenceKeys = keys
		} else if !equalStrings(referenceKeys, keys) {
			return nil, fmt.Errorf("locale %s does not have the same translation keys as %s", currentLocale, localeEN)
		}
		values[currentLocale] = translations
	}

	return &translationCatalog{values: values}, nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (catalog *translationCatalog) translations(currentLocale locale) map[string]string {
	if translations, ok := catalog.values[currentLocale]; ok {
		return translations
	}
	return catalog.values[localeEN]
}

func (catalog *translationCatalog) translationsJSON(currentLocale locale) string {
	data, err := json.Marshal(catalog.translations(currentLocale))
	if err != nil {
		panic(fmt.Errorf("marshal locale %s: %w", currentLocale, err))
	}
	return string(data)
}

func (catalog *translationCatalog) detect(request *http.Request) locale {
	if requested := localeFromValue(request.URL.Query().Get("lang")); requested != "" {
		return requested
	}
	if cookie, err := request.Cookie("testkit_locale"); err == nil {
		if requested := localeFromValue(cookie.Value); requested != "" {
			return requested
		}
	}

	tags, _, err := language.ParseAcceptLanguage(request.Header.Get("Accept-Language"))
	if err == nil && len(tags) > 0 {
		_, index, _ := acceptLanguageMatcher.Match(tags...)
		return supportedLocales[index]
	}
	return localeEN
}

func localeFromValue(value string) locale {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "en", "en-us", "en-gb":
		return localeEN
	case "pt", "pt-br", "pt-pt":
		return localePtBR
	case "es", "es-ar", "es-es":
		return localeEsAR
	default:
		return ""
	}
}
