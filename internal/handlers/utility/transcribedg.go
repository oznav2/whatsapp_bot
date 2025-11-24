package utility

import (
	"fmt"
	"strings"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
)

type TranscribeDGCommand struct{}

func NewTranscribeDGCommand() *TranscribeDGCommand {
	return &TranscribeDGCommand{}
}

func (c *TranscribeDGCommand) Execute(ctx *framework.Context) error {
	if len(ctx.Args) == 0 {
		return ctx.Handler.SendResponse(ctx.MessageInfo, "❌ אנא ספק URL של וידאו\n\nדוגמה:\n/transcribedg https://youtube.com/watch?v=...")
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

	// Transcribe using Deepgram Nova-3 cloud API - silently, no progress messages
	request := framework.WSTranscriptionRequest{
		URL:         targetURL,
		Language:    "",      // Deepgram auto-detects
		Model:       "deepgram",
		CaptureMode: "full",
	}

	result, err := transcriptionSvc.TranscribeViaWebSocket(ctx.Context, request, nil)
	if err != nil {
		return ctx.Handler.SendMessage(ctx.MessageInfo.Chat, fmt.Sprintf("❌ שגיאה בתמלול: %v", err))
	}

	// Use detected language from Deepgram result
	detectedLanguage := result.DetectedLanguage
	if detectedLanguage == "" {
		detectedLanguage = result.Language
	}
	if detectedLanguage == "" {
		detectedLanguage = "Unknown"
	}

	// Final response - send as new message directly to chat (not tied to msgInfo)
	finalResponse := fmt.Sprintf("🎬 *תמלול הושלם* (Deepgram Nova-3 ☁️)\n\n📹 סרטון: \"%s\"\n⏱️ משך: %s\n🔤 שפה: %s\n\n📝 *תמלול:*\n\n%s",
		videoTitle, videoDuration, detectedLanguage, result.Text)

	return ctx.Handler.SendMessage(ctx.MessageInfo.Chat, finalResponse)
}

func (c *TranscribeDGCommand) Metadata() *framework.Metadata {
	return &framework.Metadata{
		Name:        "transcribedg",
		Description: "Transcribe video using Deepgram Nova-3 cloud API",
		Usage:       "/transcribedg [url]",
		Examples:    []string{"/transcribedg https://youtube.com/watch?v=xyz"},
		Category:    "Utility",
	}
}
