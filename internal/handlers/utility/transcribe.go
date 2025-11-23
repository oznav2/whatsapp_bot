package utility

import (
	"fmt"
	"strings"
	"time"

	framework "github.com/asparkoffire/whatsapp-livetranslate-go/internal/cmdframework"
)

type TranscribeCommand struct{}

func NewTranscribeCommand() *TranscribeCommand {
	return &TranscribeCommand{}
}

func (c *TranscribeCommand) Execute(ctx *framework.Context) error {
	if len(ctx.Args) == 0 {
		return ctx.Handler.SendResponse(ctx.MessageInfo, "❌ אנא ספק URL של וידאו\n\nדוגמה:\n/transcribe https://youtube.com/watch?v=...")
	}

	targetURL := ctx.Args[0]

	if !strings.HasPrefix(targetURL, "https:") && !strings.HasPrefix(targetURL, "http:") {
		return ctx.Handler.SendResponse(ctx.MessageInfo, "❌ URL לא תקין")
	}

	transcriptionSvc := ctx.Handler.GetTranscriptionService()
	if transcriptionSvc == nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo, "❌ שירות התמלול לא זמין")
	}

	// Send initial status
	ctx.Handler.SendResponse(ctx.MessageInfo, "📊 מקבל מידע על הווידאו...")

	metadata, err := transcriptionSvc.GetVideoMetadata(ctx.Context, targetURL)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("❌ שגיאה בקבלת מידע: %v", err))
	}

	videoTitle := metadata.Title
	if videoTitle == "" {
		videoTitle = "Unknown Video"
	}
	videoDuration := formatDuration(metadata.DurationSeconds)

	ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("🔍 מזהה שפה...\n\n📹 סרטון: \"%s\"\n⏱️ משך: %s", videoTitle, videoDuration))

	detectedLangCode, err := transcriptionSvc.QuickLanguageDetection(ctx.Context, targetURL)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("❌ שגיאה בזיהוי שפה: %v", err))
	}

	// Handle unknown language - default to English and use detect mode
	if detectedLangCode == "unknown" || detectedLangCode == "" {
		detectedLangCode = "en"
	}

	model := "deepgram"
	languageName := detectedLangCode
	if detectedLangCode == "he" || detectedLangCode == "iw" {
		model = "ivrit-ct2"
		languageName = "Hebrew"
	}

	verboseMsg := fmt.Sprintf("🎬 מתמלל עכשיו \"%s\"\n⏱️ משך: %s\n🔤 שפה מזוהה: %s",
		videoTitle, videoDuration, languageName)
	ctx.Handler.SendResponse(ctx.MessageInfo, verboseMsg)

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
				progressMsg := fmt.Sprintf("📥 הורדה: %.0f%%", msg.Percent)
				ctx.Handler.SendResponse(ctx.MessageInfo, progressMsg)
			}
		case "transcription_chunk":
			// Collect silently
		}
	}

	result, err := transcriptionSvc.TranscribeViaWebSocket(ctx.Context, request, progressCallback)
	if err != nil {
		return ctx.Handler.SendResponse(ctx.MessageInfo, fmt.Sprintf("❌ שגיאה בתמלול: %v", err))
	}

	finalResponse := fmt.Sprintf("🎬 *תמלול הושלם*\n\n📹 סרטון: \"%s\"\n⏱️ משך: %s\n🔤 שפה: %s\n\n📝 *תמלול:*\n\n%s",
		videoTitle, videoDuration, languageName, result.Text)

	return ctx.Handler.SendResponse(ctx.MessageInfo, finalResponse)
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
