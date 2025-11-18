package transcription

import (
	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/services/transcription"
)

type TDisableCommand struct{}

func NewTDisableCommand() *TDisableCommand {
	return &TDisableCommand{}
}

func (c *TDisableCommand) Execute(ctx *framework.Context) error {
	// Get state manager from handler
	stateManager := ctx.Handler.(interface {
		GetStateManager() *transcription.StateManager
	}).GetStateManager()

	// Disable transcription for this chat
	err := stateManager.Disable(ctx.Context, ctx.MessageInfo.Chat)
	if err != nil {
		return ctx.Handler.SendResponse(
			ctx.MessageInfo,
			framework.Error("Failed to disable transcription"),
		)
	}

	// Send success message
	return ctx.Handler.SendResponse(
		ctx.MessageInfo,
		framework.Success("🔇 Transcription disabled for this chat"),
	)
}

func (c *TDisableCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:         "tdisable",
		Description:  "Disable audio transcription for this chat",
		Category:     "Transcription",
		Usage:        "/tdisable",
		Examples:     []string{"/tdisable"},
		RequireOwner: true,
	}
}
