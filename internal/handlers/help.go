package handlers

import (
	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
)

type HelpCommand struct {
	registry *framework.Registry
}

func NewHelpCommand(registry *framework.Registry) *HelpCommand {
	return &HelpCommand{registry: registry}
}

func (c *HelpCommand) Execute(ctx *framework.Context) error {
	if len(ctx.Args) > 0 {
		help := c.registry.GenerateCommandHelp(ctx.Args[0])
		return ctx.Handler.SendResponse(ctx.MessageInfo, help)
	}

	help := c.registry.GenerateHelp()
	return ctx.Handler.SendResponse(ctx.MessageInfo, help)
}

func (c *HelpCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:        "help",
		Description: "הצג פקודות זמינות",
		Category:    "כלי שירות",
		Usage:       "/help [command]",
		Examples: []string{
			"/help",
			"/help translate",
			"/help image",
		},
		Parameters: []framework.Parameter{
			{
				Name:        "command",
				Type:        framework.StringParam,
				Description: "Command to get help for",
				Required:    false,
			},
		},
	}
}
