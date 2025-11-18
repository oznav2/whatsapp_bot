package translation

import (
	"context"
	"fmt"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/constants"
	"github.com/asparkoffire/whatsapp-livetranslate-go/internal/utils"
	"github.com/pemistahl/lingua-go"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
)

type TranslateCommand struct {
	langCode   string
	targetLang lingua.Language
}

func NewTranslateCommand(langCode string) *TranslateCommand {
	return &TranslateCommand{
		langCode:   langCode,
		targetLang: utils.GetLangByCode(langCode),
	}
}

func (c *TranslateCommand) Execute(ctx *framework.Context) error {
	fmt.Printf("[TRANSLATE] Command: /%s, Target: %s\n", c.langCode, c.targetLang.String())
	
	// Handle media caption translation
	if c.handleMediaCaptionTranslation(ctx) {
		return nil
	}

	// Handle quoted message translation
	if c.handleQuotedMessageTranslation(ctx) {
		return nil
	}

	// Handle inline translation
	if !c.handleInlineTranslation(ctx) {
		return fmt.Errorf("translation failed")
	}
	return nil
}

func (c *TranslateCommand) Metadata() *framework.Metadata {
	langName := constants.SupportedLanguages[c.langCode]
	return &framework.Metadata{
		Name:        c.langCode,
		Description: fmt.Sprintf("תרגם ל%s", langName),
		Category:    "תרגום",
		Usage:       fmt.Sprintf("/%s <text>", c.langCode),
		Examples: []string{
			fmt.Sprintf("/%s Hello world", c.langCode),
			fmt.Sprintf("Quote a message and reply with /%s", c.langCode),
			fmt.Sprintf("Media caption: /%s <text> (returns translation)", c.langCode),
		},
	}
}

func (c *TranslateCommand) handleMediaCaptionTranslation(ctx *framework.Context) bool {
	if !isMediaMessage(ctx.Message) {
		return false
	}

	if len(ctx.Args) == 0 {
		return false
	}

	textToTranslate := ctx.RawArgs
	detectedLang, err := ctx.Handler.GetLangDetector().DetectLanguage(textToTranslate)
	if err != nil {
		ctx.Handler.SendResponse(ctx.MessageInfo, framework.Error("Could not detect source language"))
		return true
	}

	fmt.Printf("[TRANSLATE] Media caption translation: detected=%s, target=%s, text=%s\n", detectedLang, c.langCode, textToTranslate)

	translated, err := ctx.Handler.GetTranslator().TranslateText(
		context.Background(), textToTranslate, detectedLang, c.langCode)
	if err != nil {
		ctx.Handler.SendResponse(ctx.MessageInfo, framework.Error(fmt.Sprintf("Translation failed: %v", err)))
		return true
	}

	fmt.Printf("[TRANSLATE] Media caption translated: %s\n", translated)

	// For media messages from the user, we need to edit the caption
	if ctx.MessageInfo.IsFromMe {
		// Use the new method that passes the original message for proper media caption editing
		if err := ctx.Handler.EditMessageWithOriginal(ctx.MessageInfo, translated, ctx.Message); err != nil {
			fmt.Printf("[TRANSLATE] Caption edit failed: %v\n", err)
			// Fallback to sending as text response
			ctx.Handler.SendResponse(ctx.MessageInfo, translated)
		}
	} else {
		// For messages from others, send as text response
		ctx.Handler.SendResponse(ctx.MessageInfo, translated)
	}

	return true
}

func (c *TranslateCommand) handleQuotedMessageTranslation(ctx *framework.Context) bool {
	if len(ctx.Args) > 0 {
		return false // Has args, so it's inline translation
	}

	quotedMsg, msgType, err := getQuotedMessageAndType(ctx.Message)
	if err != nil {
		return false // No quoted message
	}

	quotedText := extractText(quotedMsg)
	if quotedText == "" {
		ctx.Handler.SendResponse(ctx.MessageInfo, framework.Warning("Quoted message has no translatable text"))
		return true
	}

	detectedLang, err := ctx.Handler.GetLangDetector().DetectLanguage(quotedText)
	if err != nil {
		ctx.Handler.SendResponse(ctx.MessageInfo, framework.Error("Could not detect source language"))
		return true
	}

	fmt.Printf("[TRANSLATE] Quoted message translation: detected=%s, target=%s, text=%s\n", detectedLang, c.langCode, quotedText)

	translated, err := ctx.Handler.GetTranslator().TranslateText(
		context.Background(), quotedText, detectedLang, c.langCode)
	if err != nil {
		ctx.Handler.SendResponse(ctx.MessageInfo, framework.Error(fmt.Sprintf("Translation failed: %v", err)))
		return true
	}

	fmt.Printf("[TRANSLATE] Translation result: %s\n", translated)

	quotedMsgID := ctx.Message.GetExtendedTextMessage().GetContextInfo().GetStanzaID()
	isMedia := isMediaMessage(quotedMsg)

	if ctx.MessageInfo.IsFromMe && isMedia && quotedMsgID != "" {
		fmt.Printf("Attempting to edit %s caption\n", msgType)
		if err := ctx.Handler.EditMessage(ctx.MessageInfo, translated); err != nil {
			fmt.Printf("Quoted %s caption edit failed: %v\n", msgType, err)
		}
	} else if ctx.MessageInfo.IsFromMe {
		if err := ctx.Handler.EditMessage(ctx.MessageInfo, translated); err != nil {
			fmt.Printf("Edit failed: %v\n", err)
		}
	} else {
		ctx.Handler.SendResponse(ctx.MessageInfo, translated)
	}

	return true
}

func (c *TranslateCommand) handleInlineTranslation(ctx *framework.Context) bool {
	if len(ctx.Args) == 0 {
		return false
	}

	textToTranslate := ctx.RawArgs

	detectedLang, err := ctx.Handler.GetLangDetector().DetectLanguage(textToTranslate)
	if err != nil {
		ctx.Handler.SendResponse(ctx.MessageInfo, framework.Error("Could not detect source language"))
		return false
	}

	fmt.Printf("[TRANSLATE] Inline translation: detected=%s, target=%s, text=%s\n", detectedLang, c.langCode, textToTranslate)

	translated, err := ctx.Handler.GetTranslator().TranslateText(
		context.Background(), textToTranslate, detectedLang, c.langCode)
	if err != nil {
		ctx.Handler.SendResponse(ctx.MessageInfo, framework.Error(fmt.Sprintf("Translation failed: %v", err)))
		return false
	}

	fmt.Printf("[TRANSLATE] Translation result: %s\n", translated)

	ctx.Handler.SendResponse(ctx.MessageInfo, translated)
	return true
}

// Helper functions - these should ideally be moved to a utility package
func isMediaMessage(msg *waProto.Message) bool {
	return msg.GetImageMessage() != nil ||
		msg.GetVideoMessage() != nil ||
		msg.GetDocumentMessage() != nil
}

func extractText(msg *waProto.Message) string {
	if msg == nil {
		return ""
	}

	if text := msg.GetConversation(); text != "" {
		return text
	}
	if text := msg.GetExtendedTextMessage().GetText(); text != "" {
		return text
	}
	if caption := msg.GetImageMessage().GetCaption(); caption != "" {
		return caption
	}
	if caption := msg.GetVideoMessage().GetCaption(); caption != "" {
		return caption
	}
	if caption := msg.GetDocumentMessage().GetCaption(); caption != "" {
		return caption
	}

	return ""
}

func getQuotedMessageAndType(msg *waProto.Message) (*waProto.Message, string, error) {
	contextInfo := msg.GetExtendedTextMessage().GetContextInfo()
	if contextInfo == nil || contextInfo.GetQuotedMessage() == nil {
		return nil, "", fmt.Errorf("no quoted message")
	}

	quotedMsg := contextInfo.GetQuotedMessage()

	var msgType string
	switch {
	case quotedMsg.GetImageMessage() != nil:
		msgType = "image"
	case quotedMsg.GetVideoMessage() != nil:
		msgType = "video"
	case quotedMsg.GetDocumentMessage() != nil:
		msgType = "document"
	default:
		msgType = "text"
	}

	return quotedMsg, msgType, nil
}

// RegisterTranslationCommands registers all supported language translation commands
func RegisterTranslationCommands(registry *framework.Registry) error {
	for langCode := range constants.SupportedLanguages {
		cmd := NewTranslateCommand(langCode)
		if err := registry.Register(cmd); err != nil {
			return fmt.Errorf("failed to register translation command %s: %w", langCode, err)
		}
	}
	return nil
}
