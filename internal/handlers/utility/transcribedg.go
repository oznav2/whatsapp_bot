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

	// Send single status message
	ctx.Handler.SendResponse(ctx.MessageInfo, "📊 מקבל מידע ומתמלל עם Deepgram ☁️...")

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

	// Display video info (single consolidated message)
	ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("🎬 מתמלל: \"%s\"\n⏱️ משך: %s\n☁️ Deepgram Nova-3\n🖼️ %s", videoTitle, videoDuration, thumbnailURL))

	// Transcribe using Deepgram Nova-3 cloud API
	request := framework.WSTranscriptionRequest{
		URL:         targetURL,
		Language:    "",      // Deepgram auto-detects
		Model:       "deepgram",
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

	// Use detected language from Deepgram result
	detectedLanguage := result.DetectedLanguage
	if detectedLanguage == "" {
		detectedLanguage = result.Language
	}
	if detectedLanguage == "" {
		detectedLanguage = "Unknown"
	}

	// Final response with thumbnail URL visible
	finalResponse := fmt.Sprintf("🎬 *תמלול הושלם* (Deepgram Nova-3 ☁️)\n\n📹 סרטון: \"%s\"\n⏱️ משך: %s\n🔤 שפה: %s\n🖼️ %s\n\n📝 *תמלול:*\n\n%s",
		videoTitle, videoDuration, detectedLanguage, thumbnailURL, result.Text)

	return ctx.Handler.SendResponse(ctx.MessageInfo, finalResponse)
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
