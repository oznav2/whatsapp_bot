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

	// Send single status message
	ctx.Handler.SendResponse(ctx.MessageInfo, "📊 מקבל מידע ומתמלל עם Whisper V3 Turbo...")

	// Get video metadata from VibeGram service
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

	thumbnailURL := metadata.Thumbnail

	// Display video info
	ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("🎬 מתמלל: \"%s\"\n⏱️ משך: %s\n🤖 Whisper V3 Turbo", videoTitle, videoDuration))

	// Send thumbnail as separate message so WhatsApp displays it as image
	if thumbnailURL != "" {
		ctx.Handler.SendResponse(ctx.MessageInfo, thumbnailURL)
	}

	// Transcribe using Whisper V3 Turbo (multilingual model)
	request := framework.WSTranscriptionRequest{
		URL:         targetURL,
		Language:    "en",
		Model:       "whisper-v3-turbo",
		CaptureMode: "full",
	}

	lastPercent := 0.0
	progressCallback := func(msg framework.WSTranscriptionMessage) {
		switch msg.Type {
		case "download_progress":
			// Only show 50% to minimize messages
			if msg.Percent >= 50.0 && lastPercent < 50.0 {
				lastPercent = msg.Percent
				ctx.Handler.SendResponse(ctx.MessageInfo, "📥 מוריד... 50%")
			}
		case "transcription_chunk", "transcription":
			// Collect silently (VibeGram sends these)
		}
	}

	result, err := transcriptionSvc.TranscribeViaWebSocket(ctx.Context, request, progressCallback)
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
