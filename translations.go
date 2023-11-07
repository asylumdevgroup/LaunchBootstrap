package main

import (
	"embed"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/jeandeaual/go-locale"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed lang/locale.*.toml
var LocalesFS embed.FS

var localizer *i18n.Localizer = nil

func GetLocalizer() *i18n.Localizer {
	userLocales, _ := locale.GetLocales()

	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	available_langs := []string{"en", "fr"}

	for _, lang := range available_langs {
		_, err := bundle.LoadMessageFileFS(LocalesFS, "lang/locale."+lang+".toml")
		if err != nil {
			fmt.Println(err)
		}
	}

	return i18n.NewLocalizer(bundle, userLocales...)
}

func Localize(id string, data map[string]string) string {
	if localizer == nil {
		localizer = GetLocalizer()
	}

	return localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: id,
		},
		TemplateData: data,
	})
}
