package utility

import (
	"fmt"
	"strings"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
)

type TranscribeEnCommand struct{}

func NewTranscribeEnCommand() *TranscribeEnCommand {
	return &TranscribeEnCommand{}
}

func (c *TranscribeEnCommand) Execute(ctx *framework.Context) error {
	if len(ctx.Args) == 0 {
		return ctx.Handler.SendResponse(ctx.MessageInfo, "❌ אנא ספק URL של וידאו\n\nדוגמה:\n/transcriben https://youtube.com/watch?v=...")
	}

	targetURL := ctx.Args[0]

	if !strings.HasPrefix(targetURL, "https:") && !strings.HasPrefix(targetURL, "http:") {
		return ctx.Handler.SendResponse(ctx.MessageInfo, "❌ URL לא תקין")
	}

	transcriptionSvc := ctx.Handler.GetTranscriptionService()
	if transcriptionSvc == nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo, "❌ שירות התמלול לא זמין")
	}

	// Send YouTube URL first to create preview that stays visible
	ctx.Handler.SendResponse(ctx.MessageInfo, targetURL)

	// Get video metadata from VibeGram service (silently in background)
	metadata, err := transcriptionSvc.GetVideoMetadata(ctx.Context, targetURL)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("❌ שגיאה בקבלת מידע על הווידאו: %v", err))
	}

	videoTitle := metadata.Title
	if videoTitle == "" {
		videoTitle = "Unknown Video"
	}

	videoDuration := metadata.DurationFormatted
	if videoDuration == "" {
		videoDuration = formatDuration(metadata.DurationSeconds)
	}

	// Transcribe using Whisper V3 Turbo (multilingual model) - silently, no progress messages
	request := framework.WSTranscriptionRequest{
		URL:         targetURL,
		Language:    "en",
		Model:       "whisper-v3-turbo",
		CaptureMode: "full",
	}

	result, err := transcriptionSvc.TranscribeViaWebSocket(ctx.Context, request, nil)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("❌ שגיאה בתמלול: %v", err))
	}

	// Final response (thumbnail already sent as separate image)
	finalResponse := fmt.Sprintf("🎬 *תמלול הושלם* (Whisper V3 Turbo)\n\n📹 סרטון: \"%s\"\n⏱️ משך: %s\n\n📝 *תמלול:*\n\n%s",
		videoTitle, videoDuration, result.Text)

	return ctx.Handler.SendResponse(ctx.MessageInfo, finalResponse)
}

func (c *TranscribeEnCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:        "transcriben",
		Description: "Transcribe video using Whisper V3 Turbo (multilingual)",
		Usage:       "/transcriben [url]",
		Examples:    []string{"/transcriben https://youtube.com/watch?v=xyz"},
		Category:    "Utility",
	}
}
