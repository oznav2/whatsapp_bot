package utility

import (
	"context"
	"fmt"
	"strings"
	"time"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

type TranscribeCommand struct{}

func NewTranscribeCommand() *TranscribeCommand {
	return &TranscribeCommand{}
}

func (c *TranscribeCommand) Execute(ctx *framework.Context) error {
	if len(ctx.Args) == 0 {
		return c.sendMessage(ctx, "❌ אנא ספק URL של וידאו\n\nדוגמה:\n/transcribe https://youtube.com/watch?v=...")
	}

	targetURL := ctx.Args[0]

	if !strings.HasPrefix(targetURL, "https:") && !strings.HasPrefix(targetURL, "http:") {
		return c.sendMessage(ctx, "❌ URL לא תקין")
	}

	transcriptionSvc := ctx.Handler.GetTranscriptionService()
	if transcriptionSvc == nil {
		return c.sendMessage(ctx, "❌ שירות התמלול לא זמין")
	}

	senderJID := ctx.MessageInfo.Chat
	if ctx.MessageInfo.Chat.Server == "g.us" {
		senderJID = types.NewJID(ctx.MessageInfo.Chat.User, "s.whatsapp.net")
	}

	initialMsg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String("📊 מקבל מידע על הווידאו..."),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:    proto.String(ctx.MessageInfo.ID),
				Participant: proto.String(senderJID.String()),
			},
		},
	}

	client := ctx.Handler.GetClient()
	resp, err := client.SendMessage(ctx.Context, ctx.MessageInfo.Chat, initialMsg)
	if err != nil {
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	statusMessageID := resp.ID

	metadata, err := transcriptionSvc.GetVideoMetadata(ctx.Context, targetURL)
	if err != nil {
		c.editStatusMessage(ctx.Context, client, ctx.MessageInfo.Chat, statusMessageID, ctx.MessageInfo.ID, senderJID,
			fmt.Sprintf("❌ שגיאה בקבלת מידע: %v", err))
		return nil
	}

	videoTitle := metadata.Title
	if videoTitle == "" {
		videoTitle = "Unknown Video"
	}
	videoDuration := formatDuration(metadata.DurationSeconds)

	c.editStatusMessage(ctx.Context, client, ctx.MessageInfo.Chat, statusMessageID, ctx.MessageInfo.ID, senderJID,
		fmt.Sprintf("🔍 מזהה שפה...\n\n📹 סרטון: \"%s\"\n⏱️ משך: %s", videoTitle, videoDuration))

	detectedLangCode, err := transcriptionSvc.QuickLanguageDetection(ctx.Context, targetURL)
	if err != nil {
		c.editStatusMessage(ctx.Context, client, ctx.MessageInfo.Chat, statusMessageID, ctx.MessageInfo.ID, senderJID,
			fmt.Sprintf("❌ שגיאה בזיהוי שפה: %v", err))
		return nil
	}

	model := "deepgram"
	languageName := detectedLangCode
	if detectedLangCode == "he" || detectedLangCode == "iw" {
		model = "ivrit-ct2"
		languageName = "Hebrew"
	}

	verboseMsg := fmt.Sprintf("🎬 מתמלל עכשיו \"%s\"\n⏱️ משך: %s\n🔤 שפה מזוהה: %s",
		videoTitle, videoDuration, languageName)
	c.editStatusMessage(ctx.Context, client, ctx.MessageInfo.Chat, statusMessageID, ctx.MessageInfo.ID, senderJID, verboseMsg)

	request := framework.WSTranscriptionRequest{
		URL:         targetURL,
		Language:    detectedLangCode,
		Model:       model,
		CaptureMode: "full",
	}

	lastPercent := 0.0
	lastUpdate := time.Now()
	progressCallback := func(msg framework.WSTranscriptionMessage) {
		switch msg.Type {
		case "download_progress":
			now := time.Now()
			if msg.Percent-lastPercent >= 25.0 || msg.Percent >= 99.0 || now.Sub(lastUpdate) > 2*time.Second {
				lastPercent = msg.Percent
				lastUpdate = now
				progressMsg := fmt.Sprintf("🎬 מתמלל עכשיו \"%s\"\n⏱️ משך: %s\n🔤 שפה מזוהה: %s\n\n📥 הורדה: %.0f%%",
					videoTitle, videoDuration, languageName, msg.Percent)
				c.editStatusMessage(ctx.Context, client, ctx.MessageInfo.Chat, statusMessageID, ctx.MessageInfo.ID, senderJID, progressMsg)
			}
		case "transcription_chunk":
			// Collect silently
		}
	}

	result, err := transcriptionSvc.TranscribeViaWebSocket(ctx.Context, request, progressCallback)
	if err != nil {
		c.editStatusMessage(ctx.Context, client, ctx.MessageInfo.Chat, statusMessageID, ctx.MessageInfo.ID, senderJID,
			fmt.Sprintf("❌ שגיאה בתמלול: %v", err))
		return nil
	}

	finalResponse := fmt.Sprintf("🎬 *תמלול הושלם*\n\n📹 סרטון: \"%s\"\n⏱️ משך: %s\n🔤 שפה: %s\n\n📝 *תמלול:*\n\n%s",
		videoTitle, videoDuration, languageName, result.Text)

	c.editStatusMessage(ctx.Context, client, ctx.MessageInfo.Chat, statusMessageID, ctx.MessageInfo.ID, senderJID, finalResponse)

	return nil
}

func (c *TranscribeCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:        "transcribe",
		Description: "Transcribe video from URL",
		Usage:       "/transcribe [url]",
		Examples:    []string{"/transcribe https://youtube.com/watch?v=xyz"},
		Category:    "Utility",
	}
}

func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d seconds", seconds)
	}

	minutes := seconds / 60
	remainingSeconds := seconds % 60

	if minutes < 60 {
		if remainingSeconds == 0 {
			return fmt.Sprintf("%dmin", minutes)
		}
		return fmt.Sprintf("%dmin and %d seconds", minutes, remainingSeconds)
	}

	hours := minutes / 60
	remainingMinutes := minutes % 60

	if remainingMinutes == 0 && remainingSeconds == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	if remainingSeconds == 0 {
		return fmt.Sprintf("%dh and %dmin", hours, remainingMinutes)
	}
	return fmt.Sprintf("%dh, %dmin and %d seconds", hours, remainingMinutes, remainingSeconds)
}

func (c *TranscribeCommand) sendMessage(ctx *framework.Context, text string) error {
	client := ctx.Handler.GetClient()
	msg := &waProto.Message{
		Conversation: proto.String(text),
	}
	_, err := client.SendMessage(ctx.Context, ctx.MessageInfo.Chat, msg)
	return err
}

func (c *TranscribeCommand) editStatusMessage(ctx context.Context, client interface{}, chatJID types.JID, messageID string, replyToID string, senderJID types.JID, text string) {
	type ClientInterface interface {
		BuildEdit(chatJID types.JID, messageID string, newMessage *waProto.Message) *waProto.Message
		SendMessage(ctx context.Context, chatJID types.JID, message *waProto.Message) (types.MessageInfo, error)
	}

	clientTyped, ok := client.(ClientInterface)
	if !ok {
		fmt.Printf("Failed to cast client\n")
		return
	}

	updatedMsg := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: proto.String(text),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:    proto.String(replyToID),
				Participant: proto.String(senderJID.String()),
			},
		},
	}

	editMsg := clientTyped.BuildEdit(chatJID, messageID, updatedMsg)
	_, err := clientTyped.SendMessage(ctx, chatJID, editMsg)
	if err != nil {
		fmt.Printf("Failed to edit message: %v\n", err)
	}
}
