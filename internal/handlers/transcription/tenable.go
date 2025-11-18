package transcription

import (
	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
)

type TEnableCommand struct{}

func NewTEnableCommand() *TEnableCommand {
	return &TEnableCommand{}
}

func (c *TEnableCommand) Execute(ctx *framework.Context) error {
	// Get state manager from handler
	stateManager := ctx.Handler.(interface {
		GetStateManager() *transcription.StateManager
	}).GetStateManager()

	// Determine language (default to Hebrew)
	language := "he"
	if len(ctx.Args) > 0 {
		language = ctx.Args[0]
	}

	// Enable transcription for this chat
	err := stateManager.Enable(ctx.Context, ctx.MessageInfo.Chat, language)
	if err != nil {
		return ctx.Handler.SendResponse(
			ctx.MessageInfo,
			framework.Error("Failed to enable transcription"),
		)
	}

	// Send success message
	message := framework.Success("✅ Transcription enabled for this chat")
	if language != "he" {
		message += "\n🌐 Language: " + language
	}

	return ctx.Handler.SendResponse(ctx.MessageInfo, message)
}

func (c *TEnableCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:        "tenable",
		Description: "Enable audio transcription for this chat",
		Category:    "Transcription",
		Usage:       "/tenable [language]",
		Examples: []string{
			"/tenable",
			"/tenable he",
			"/tenable en",
		},
		Parameters: []framework.Parameter{
			{
				Name:        "language",
				Type:        framework.StringParam,
				Description: "Transcription language code (default: he)",
				Required:    false,
				Default:     "he",
			},
		},
		RequireOwner: true,
	}
}
