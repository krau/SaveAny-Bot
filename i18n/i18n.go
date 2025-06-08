package i18n

import (
	"embed"

	"maps"

	"github.com/krau/SaveAny-Bot/common"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/language"
)

//go:embed locale/*.toml
var localesFS embed.FS

var (
	bundle    *i18n.Bundle
	localizer *i18n.Localizer
)

func Init(lang string) {
	bundle = i18n.NewBundle(language.SimplifiedChinese)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	files, err := localesFS.ReadDir("locale")
	if err != nil {
		panic("failed to read locale directory: " + err.Error())
	}
	for _, file := range files {
		if _, err := bundle.LoadMessageFileFS(localesFS, "locale/"+file.Name()); err != nil {
			panic("failed to load message file: " + err.Error())
		}
	}
	if lang == "" {
		lang = "zh-Hans"
	}
	localizer = i18n.NewLocalizer(bundle, lang)
	if localizer == nil {
		panic("failed to create localizer, check your config for valid language setting")
	}
}

func T(key string, templateData ...map[string]any) string {
	if localizer == nil || bundle == nil {
		panic("localizer or bundle is not initialized, call Init() first")
	}
	templateDataMap := make(map[string]any)
	for _, data := range templateData {
		maps.Copy(templateDataMap, data)
	}
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: templateDataMap,
	})
	if err != nil {
		common.Log.Errorf("failed to localize message for key '%s': %v", key, err)
		return key
	}
	return msg
}
