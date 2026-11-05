package i18n

import (
	"embed"
	"sync"

	"maps"

	"github.com/goccy/go-yaml"
	"github.com/krau/SaveAny-Bot/common/i18n/i18nk"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locale/*
var localesFS embed.FS

var (
	mu        sync.RWMutex
	bundle    *i18n.Bundle
	localizer *i18n.Localizer
)

func Init(lang string) {
	newBundle := i18n.NewBundle(language.SimplifiedChinese)
	newBundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	files, err := localesFS.ReadDir("locale")
	if err != nil {
		panic("failed to read locale directory: " + err.Error())
	}
	for _, file := range files {
		if _, err := newBundle.LoadMessageFileFS(localesFS, "locale/"+file.Name()); err != nil {
			panic("failed to load message file: " + err.Error())
		}
	}
	if lang == "" {
		lang = "zh-Hans"
	}
	newLocalizer := i18n.NewLocalizer(newBundle, lang)
	if newLocalizer == nil {
		panic("failed to create localizer, check your config for valid language setting")
	}
	mu.Lock()
	bundle = newBundle
	localizer = newLocalizer
	mu.Unlock()
}

func T(key i18nk.Key, templateData ...map[string]any) string {
	mu.RLock()
	loc := localizer
	mu.RUnlock()
	if loc == nil {
		Init("")
		mu.RLock()
		loc = localizer
		mu.RUnlock()
	}
	templateDataMap := make(map[string]any)
	for _, data := range templateData {
		maps.Copy(templateDataMap, data)
	}
	msg, err := loc.Localize(&i18n.LocalizeConfig{
		MessageID:    string(key),
		TemplateData: templateDataMap,
	})
	if err != nil {
		return string(key)
	}
	return msg
}
