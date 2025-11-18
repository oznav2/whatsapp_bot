package utility

import (
	"fmt"
	"sort"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/constants"
)

type SupportedLangsCommand struct{}

func NewSupportedLangsCommand() *SupportedLangsCommand {
	return &SupportedLangsCommand{}
}

func (c *SupportedLangsCommand) Execute(ctx *framework.Context) error {
	builder := framework.NewResponseBuilder()
	builder.AddHeading("Supported Languages")

	// Create sorted list of languages
	type langInfo struct {
		code string
		name string
	}

	langs := make([]langInfo, 0, len(constants.SupportedLanguages))
	for code, lang := range constants.SupportedLanguages {
		langs = append(langs, langInfo{code: code, name: lang.String()})
	}

	// Sort by language name
	sort.Slice(langs, func(i, j int) bool {
		return langs[i].name < langs[j].name
	})

	// Build language list
	langList := make([]string, len(langs))
	for i, lang := range langs {
		langList[i] = fmt.Sprintf("*/%s* - %s", lang.code, lang.name)
	}

	builder.AddList(langList...)
	builder.AddEmptyLine()
	builder.AddItalic("Use any language code above to translate text to that language")

	return ctx.Handler.SendResponse(ctx.MessageInfo, builder.Build())
}

func (c *SupportedLangsCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:        "supportedlangs",
		Aliases:     []string{"langs", "languages"},
		Description: "הצג שפות תרגום נתמכות",
		Category:    "כלי שירות",
		Usage:       "/supportedlangs",
	}
}
